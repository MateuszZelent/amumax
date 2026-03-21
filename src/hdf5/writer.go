// Package hdf5 provides an HDF5 output backend for amumax simulation data.
// It uses the pure Go scigolib/hdf5 library with contiguous layout.
//
// Architecture: MultiWriter manages per-quantity HDF5 files.
// Each quantity (m, B_eff, Edens, etc.) gets its own folder and .h5 file,
// mirroring the Zarr directory structure:
//
//	simulation.zarr/
//	├── m/m.h5            ← magnetization steps + timestamps
//	├── Edens_total/Edens_total.h5
//	├── table/table.h5    ← all table columns
//	└── ...
package hdf5

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/scigolib/hdf5"
)

// Writer manages a single HDF5 file for one quantity.
type Writer struct {
	fw            *hdf5.FileWriter
	filename      string
	createdGroups map[string]bool
}

// create opens a new HDF5 file for writing.
func create(filename string) (*Writer, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("hdf5: mkdir %s: %w", dir, err)
	}

	fw, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
	if err != nil {
		return nil, fmt.Errorf("hdf5: create %s: %w", filename, err)
	}
	return &Writer{fw: fw, filename: filename, createdGroups: make(map[string]bool)}, nil
}

// close closes this single HDF5 file.
func (w *Writer) close() error {
	if w.fw == nil {
		return nil
	}
	err := w.fw.Close()
	w.fw = nil
	return err
}

// writeArray writes a 4D array as a contiguous dataset at the given path.
func (w *Writer) writeArray(dsPath string, data4 [][][][]float32, size [3]int, ncomp int) error {
	if w.fw == nil {
		return fmt.Errorf("hdf5: file %s is closed", w.filename)
	}

	nx, ny, nz := size[0], size[1], size[2]
	total := nz * ny * nx * ncomp
	flat := make([]float32, total)
	idx := 0
	for iz := 0; iz < nz; iz++ {
		for iy := 0; iy < ny; iy++ {
			for ix := 0; ix < nx; ix++ {
				for ic := 0; ic < ncomp; ic++ {
					flat[idx] = data4[ic][iz][iy][ix]
					idx++
				}
			}
		}
	}

	dims := []uint64{uint64(nz), uint64(ny), uint64(nx), uint64(ncomp)}
	ds, err := w.fw.CreateDataset(dsPath, hdf5.Float32, dims)
	if err != nil {
		return fmt.Errorf("hdf5: create dataset %s in %s: %w", dsPath, w.filename, err)
	}
	if err := ds.Write(flat); err != nil {
		ds.Close()
		return fmt.Errorf("hdf5: write dataset %s in %s: %w", dsPath, w.filename, err)
	}
	return ds.Close()
}

// writeFloat64Array writes a 1D float64 array at the given path.
func (w *Writer) writeFloat64Array(dsPath string, data []float64) error {
	if w.fw == nil {
		return fmt.Errorf("hdf5: file %s is closed", w.filename)
	}

	ds, err := w.fw.CreateDataset(dsPath, hdf5.Float64, []uint64{uint64(len(data))})
	if err != nil {
		return fmt.Errorf("hdf5: create dataset %s in %s: %w", dsPath, w.filename, err)
	}
	if err := ds.Write(data); err != nil {
		ds.Close()
		return fmt.Errorf("hdf5: write dataset %s in %s: %w", dsPath, w.filename, err)
	}
	return ds.Close()
}

// ============================================================================
// MultiWriter — manages per-quantity HDF5 files
// ============================================================================

// MultiWriter manages per-quantity HDF5 files.
// Thread-safe via internal mutex.
type MultiWriter struct {
	mu      sync.Mutex
	baseDir string             // output directory (e.g. "simulation.zarr/")
	writers map[string]*Writer // keyed by quantity name (e.g. "m", "table")
}

// Global multi-writer instance.
var GlobalWriter *MultiWriter

// InitMultiWriter creates a new MultiWriter for the given base directory.
func InitMultiWriter(baseDir string) {
	GlobalWriter = &MultiWriter{
		baseDir: baseDir,
		writers: make(map[string]*Writer),
	}
}

// CloseGlobal closes all open HDF5 files.
func CloseGlobal() {
	if GlobalWriter != nil {
		GlobalWriter.Close()
	}
}

// getOrCreateWriter returns the writer for a quantity, creating it if needed.
// Must be called with mu held.
func (mw *MultiWriter) getOrCreateWriter(qname string) (*Writer, error) {
	if w, ok := mw.writers[qname]; ok {
		return w, nil
	}

	// Create: baseDir/qname/qname.h5
	filename := filepath.Join(mw.baseDir, qname, qname+".h5")
	w, err := create(filename)
	if err != nil {
		return nil, err
	}
	mw.writers[qname] = w
	return w, nil
}

// SaveArray writes a 4D array to the quantity's HDF5 file.
// Dataset path: /<step> (e.g. /0, /1, /2, ...)
func (mw *MultiWriter) SaveArray(qname string, step int, data4 [][][][]float32, size [3]int, ncomp int) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	w, err := mw.getOrCreateWriter(qname)
	if err != nil {
		return err
	}

	dsPath := fmt.Sprintf("/%d", step)
	return w.writeArray(dsPath, data4, size, ncomp)
}

// SaveTimestamps writes timestamps to the quantity's HDF5 file.
// Dataset path: /t
func (mw *MultiWriter) SaveTimestamps(qname string, times []float64) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	w, err := mw.getOrCreateWriter(qname)
	if err != nil {
		return err
	}

	return w.writeFloat64Array("/t", times)
}

// SaveTableColumn writes a table column to table/table.h5.
// Dataset path: /<colName>
func (mw *MultiWriter) SaveTableColumn(colName string, data []float64) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	w, err := mw.getOrCreateWriter("table")
	if err != nil {
		return err
	}

	dsPath := fmt.Sprintf("/%s", colName)
	return w.writeFloat64Array(dsPath, data)
}

// Close closes all open HDF5 files managed by this MultiWriter.
func (mw *MultiWriter) Close() {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for name, w := range mw.writers {
		if err := w.close(); err != nil {
			log.Log.Err("Error closing HDF5 file for %s: %v", name, err)
		}
	}
	mw.writers = make(map[string]*Writer)
}
