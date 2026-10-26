package engine

import (
	"sync"
	"time"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/MathieuMoalic/amumax/src/fsutil"
	amuhdf5 "github.com/MathieuMoalic/amumax/src/hdf5"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/script"
	"github.com/MathieuMoalic/amumax/src/tabular"
)

func init() {
	Table = tableStruct{
		Data:           make(map[string][]float64),
		Step:           -1,
		AutoSavePeriod: 0.0,
		FlushInterval:  5 * time.Second,
	}
}

var Table tableStruct

// the Table is kept in RAM and used for the API
type tableStruct struct {
	quantities     []Quantity
	Columns        []column
	Data           map[string][]float64 `json:"data"`
	AutoSavePeriod float64              `json:"autoSavePeriod"`
	AutoSaveStart  float64              `json:"autoSaveStart"`
	Step           int                  `json:"step"`
	FlushInterval  time.Duration        `json:"flushInterval"`
	Mu             sync.Mutex
	flushMu        sync.Mutex
	writer         *tabular.Writer
	persisted      int
	stopFlush      chan struct{}
	flushDone      chan struct{}
	stopOnce       sync.Once
}

type column struct {
	Name string
	Unit string
}

func (ts *tableStruct) WriteToBuffer() {
	ts.Mu.Lock()
	ts.Step++
	step := ts.Step
	quantities := append([]Quantity(nil), ts.quantities...)
	ts.Mu.Unlock()
	buf := []float64{}
	buf = append(buf, float64(step))
	// always save the current time
	buf = append(buf, Time)
	// for each quantity we append each component to the buffer
	for _, q := range quantities {
		buf = append(buf, AverageOf(q)...)
	}
	// size of buf should be same as size of []Ztable
	ts.Mu.Lock() // Lock the mutex before modifying the map
	defer ts.Mu.Unlock()
	for i, b := range buf {
		ts.Data[ts.Columns[i].Name] = append(ts.Data[ts.Columns[i].Name], b)
	}
}

func (ts *tableStruct) Flush() {
	if StorageFormat == StorageFormatHDF5 {
		// HDF5: data is kept in ts.Data map (in memory).
		// Final write happens in FlushFinalHDF5() called from CleanExit.
		// We skip periodic flushes because HDF5 can't overwrite datasets.
		return
	}
	ts.flushMu.Lock()
	defer ts.flushMu.Unlock()
	ts.Mu.Lock()
	if len(ts.Columns) == 0 || len(ts.Data[ts.Columns[0].Name]) <= ts.persisted {
		ts.Mu.Unlock()
		return
	}
	names := make([]string, len(ts.Columns))
	units := make([]string, len(ts.Columns))
	values := make([][]float64, len(ts.Columns))
	for i, col := range ts.Columns {
		names[i], units[i] = col.Name, col.Unit
		values[i] = append([]float64(nil), ts.Data[col.Name][ts.persisted:]...)
	}
	ts.Mu.Unlock()
	if ts.writer == nil {
		writer, err := tabular.New(OD()+"table", names, units)
		log.Log.PanicIfError(err)
		ts.writer = writer
	}
	log.Log.PanicIfError(ts.writer.Append(values))
	ts.Mu.Lock()
	ts.persisted += len(values[0])
	ts.Mu.Unlock()
}

// FlushFinalHDF5 writes all accumulated table data to HDF5 at shutdown.
func (ts *tableStruct) FlushFinalHDF5() {
	if amuhdf5.GlobalWriter == nil {
		return
	}
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	for _, col := range ts.Columns {
		if colData, ok := ts.Data[col.Name]; ok && len(colData) > 0 {
			err := amuhdf5.GlobalWriter.SaveTableColumn(col.Name, colData)
			if err != nil {
				log.Log.Err("HDF5 table write %s: %v", col.Name, err)
			}
		}
	}
}

func (ts *tableStruct) NeedSave() bool {
	return ts.AutoSavePeriod != 0 && (Time-ts.AutoSaveStart)-float64(ts.Step)*ts.AutoSavePeriod >= ts.AutoSavePeriod
}

func (ts *tableStruct) Exists(q Quantity, name string) bool {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	suffixes := []string{"x", "y", "z"}
	for _, i := range ts.Columns {
		if q.NComp() == 1 {
			if i.Name == name {
				return true
			}
		} else {
			for comp := 0; comp < q.NComp(); comp++ {
				if name+suffixes[comp] == i.Name {
					return true
				}
			}
		}
	}
	return false
}

func (ts *tableStruct) AddColumn(name, unit string) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	ts.Columns = append(ts.Columns, column{Name: name, Unit: unit})
}

func tableInit() {
	if StorageFormat != StorageFormatHDF5 {
		log.Log.PanicIfError(fsutil.Mkdir(OD() + "table"))
		log.Log.PanicIfError(fsutil.Put(OD()+"table/.zgroup", []byte(`{"zarr_format":2}`)))
	}
	Table.AddColumn("step", "")
	Table.AddColumn("t", "s")
	tableAdd(&NormMag)
	Table.stopFlush, Table.flushDone = make(chan struct{}), make(chan struct{})
	go tablesAutoFlush(Table.stopFlush, Table.flushDone)
}

func tablesAutoFlush(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(Table.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			func() {
				defer captureOutputPanic()
				Table.Flush()
			}()
			if outputError() != nil {
				return
			}
		}
	}
}

func (ts *tableStruct) stopAutoFlush() {
	ts.stopOnce.Do(func() {
		if ts.stopFlush != nil {
			close(ts.stopFlush)
			<-ts.flushDone
		}
	})
}

func tableSave() {
	if len(Table.Columns) == 0 {
		tableInit()
	}
	Table.WriteToBuffer()
}

func tableAdd(q Quantity) {
	tableAddAs(q, nameOf(q))
}

func tableAddAs(q Quantity, name string) {
	suffixes := []string{"x", "y", "z"}
	if Table.Step != -1 {
		log.Log.Warn("You cannot add a new quantity to the table after the simulation has started. Ignoring.")
		return
	}
	if len(Table.Columns) == 0 {
		tableInit()
	}

	if Table.Exists(q, name) {
		log.Log.Warn("%s is already in the table. Ignoring.", name)
		return
	}
	Table.Mu.Lock()
	Table.quantities = append(Table.quantities, q)
	Table.Mu.Unlock()
	if q.NComp() == 1 {
		Table.AddColumn(name, unitOf(q))
	} else {
		for comp := 0; comp < q.NComp(); comp++ {
			Table.AddColumn(name+suffixes[comp], unitOf(q))
		}
	}
}

func tableAutoSave(period float64) {
	Table.AutoSaveStart = Time
	Table.AutoSavePeriod = period
}

func tableAddVar(customvar script.ScalarFunction, name, unit string) {
	tableAdd(&userVar{customvar, name, unit})
}

type userVar struct {
	value      script.ScalarFunction
	name, unit string
}

func (x *userVar) Name() string       { return x.name }
func (x *userVar) NComp() int         { return 1 }
func (x *userVar) Unit() string       { return x.unit }
func (x *userVar) average() []float64 { return []float64{x.value.Float()} }
func (x *userVar) EvalTo(dst *data.Slice) {
	avg := x.average()
	for c := 0; c < x.NComp(); c++ {
		cuda.Memset(dst.Comp(c), float32(avg[c]))
	}
}
