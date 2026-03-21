// Package engine implements the core simulation engine.
package engine

import (
	"os"
	"sync"
	"time"

	amuhdf5 "github.com/MathieuMoalic/amumax/src/hdf5"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/timer"
)

var StartTime = time.Now()

var busyLock sync.Mutex

// We set setBusy(true) when the simulation is too busy too accept GUI input on Inject channel.
// E.g. during kernel init.
func setBusy(_b bool) {
	// TODO is it needed?
	_ = _b
	busyLock.Lock()
	defer busyLock.Unlock()
}

// CleanExit Cleanly exits the simulation, assuring all output is flushed.
func CleanExit() {
	if outputdir == "" {
		return
	}
	drainOutput()
	log.Log.Info("**************** Simulation Ended ****************** //")
	Table.Flush()
	if StorageFormat == StorageFormatHDF5 {
		Table.FlushFinalHDF5()
	}
	SaveAllTimestampsHDF5()
	amuhdf5.CloseGlobal()
	if SyncAndLog {
		timer.Print(os.Stdout)
	}
	EngineState.Metadata.Add("steps", NSteps)
	EngineState.Metadata.End()
	log.Log.FlushToFile()
}
