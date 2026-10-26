// Package engine implements the core simulation engine.
package engine

import (
	"os"
	"sync"
	"time"

	"github.com/MathieuMoalic/amumax/src/fsutil"
	amuhdf5 "github.com/MathieuMoalic/amumax/src/hdf5"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/timer"
	"github.com/MathieuMoalic/amumax/src/zarr"
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
	if outputdir == "" || !outputOwned {
		return
	}
	defer func() {
		if outputLock != "" {
			log.Log.PanicIfError(fsutil.ReleaseOutputLock(outputLock, outputLockOwner))
			outputLock = ""
		}
	}()
	waitOutput()
	Table.stopAutoFlush()
	if StorageFormat != StorageFormatHDF5 {
		for _, quantity := range savedQuantities.Quantities {
			if quantity.output != nil && quantity.output.committed > 0 {
				log.Log.PanicIfError(zarr.PublishProjection(OD() + quantity.name))
			}
		}
	}
	log.Log.Info("**************** Simulation Ended ****************** //")
	if outputError() == nil {
		Table.Flush()
	}
	if StorageFormat == StorageFormatHDF5 {
		Table.FlushFinalHDF5()
	}
	SaveAllTimestampsHDF5()
	amuhdf5.CloseGlobal()
	if SyncAndLog {
		timer.Print(os.Stdout)
	}
	EngineState.Metadata.Add("steps", NSteps)
	if outputError() != nil || runFailure.Load() {
		EngineState.Metadata.Add("completion", "failed")
	} else if terminationSignal.Load() != 0 {
		EngineState.Metadata.Add("completion", "interrupted")
	} else {
		EngineState.Metadata.Add("completion", "complete")
	}
	EngineState.Metadata.End()
	log.Log.FlushToFile()
	checkOutputError()
}
