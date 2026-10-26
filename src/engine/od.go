package engine

// Management of output directory.

import (
	"os"
	"strings"

	"github.com/MathieuMoalic/amumax/src/cuda"
	"github.com/MathieuMoalic/amumax/src/fsutil"
	amuhdf5 "github.com/MathieuMoalic/amumax/src/hdf5"
	"github.com/MathieuMoalic/amumax/src/log"
	"github.com/MathieuMoalic/amumax/src/zarr"
)

var (
	outputdir       string // Output directory
	InputFile       string
	outputLock      string
	outputLockOwner []byte
	outputOwned     bool

	CacheDir       string
	SkipExists     bool
	ForceClean     bool
	HideProgresBar bool
	SelfTest       bool
	SyncAndLog     bool
)

func OD() string {
	if outputdir == "" {
		panic("output not yet initialized")
	}
	return outputdir
}

// InitIO SetOD sets the output directory where auto-saved files will be stored.
// The -o flag can also be used for this purpose.

func InitIO(mx3Path, od, cachedir string, skipexists, forceclean, hideprogressbar, selftest, syncandlog bool) {
	CacheDir = cachedir
	SkipExists = skipexists
	ForceClean = forceclean
	HideProgresBar = hideprogressbar
	SelfTest = selftest
	SyncAndLog = syncandlog

	if outputdir != "" {
		panic("output directory already set")
	}
	InputFile = mx3Path
	if !strings.HasSuffix(od, "/") {
		od += "/"
	}
	outputdir = od
	if strings.HasPrefix(outputdir, "http://") {
		fsutil.SetWD(outputdir + "/../")
	}
	if fsutil.IsDir(od) && SkipExists {
		log.Log.Warn("Directory `%s` exists, skipping `%s` because of --skip-exist flag.", od, mx3Path)
		os.Exit(0)
	}
	var lockErr error
	outputLock, lockErr = fsutil.AcquireOutputLock(od)
	log.Log.PanicIfError(lockErr)
	outputLockOwner, lockErr = fsutil.Read(outputLock + "/owner.json")
	log.Log.PanicIfError(lockErr)
	initialized := false
	defer func() {
		if !initialized && outputLock != "" {
			_ = fsutil.ReleaseOutputLock(outputLock, outputLockOwner)
			outputLock = ""
		}
	}()
	if fsutil.IsDir(od) {
		if ForceClean {
			log.Log.Warn("Cleaning `%s`", od)
			log.Log.PanicIfError(fsutil.Remove(od))
			log.Log.PanicIfError(fsutil.Mkdir(od))
		} else {
			entries, err := os.ReadDir(od)
			log.Log.PanicIfError(err)
			if len(entries) != 0 {
				panic("output directory is not empty; choose a new directory or explicitly use --force-clean")
			}
		}
	} else {
		log.Log.PanicIfError(fsutil.Mkdir(od))
	}
	if StorageFormat == StorageFormatHDF5 {
		amuhdf5.InitMultiWriter(od)
	} else {
		zarr.InitZgroup("", OD())
	}
	EngineState.Metadata.SetInputPath(InputFile)
	EngineState.Metadata.Init(OD(), StartTime, cuda.GPUInfoOld)
	initialized = true
	outputOwned = true
}
