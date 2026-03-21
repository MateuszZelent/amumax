package engine

import (
	"reflect"

	amuhdf5 "github.com/MathieuMoalic/amumax/src/hdf5"
	"github.com/MathieuMoalic/amumax/src/log"
)

// StorageFormatType controls the storage backend for saved quantities.
type StorageFormatType int

const (
	StorageFormatZarr StorageFormatType = iota
	StorageFormatHDF5
)

// StorageFormat is the active storage format. Default: Zarr.
var StorageFormat StorageFormatType = StorageFormatZarr

// sformat exposes StorageFormat to the mx3 script world.
type sformat struct{}

func (*sformat) Eval() any      { return StorageFormat }
func (*sformat) Type() reflect.Type { return reflect.TypeOf(StorageFormatType(StorageFormatZarr)) }

func (*sformat) SetValue(v any) {
	drainOutput()
	newFormat := v.(StorageFormatType)
	if newFormat == StorageFormatHDF5 && StorageFormat != StorageFormatHDF5 {
		// Late switch to HDF5: initialize multi-writer now
		if amuhdf5.GlobalWriter == nil && outputdir != "" {
			amuhdf5.InitMultiWriter(outputdir)
			log.Log.Info("Switched storage format to HDF5 (per-quantity files in %s)", outputdir)
		}
	}
	StorageFormat = newFormat
}

func init() {
	declROnly("ZARR", StorageFormatZarr, "StorageFormat = ZARR sets Zarr output (default)")
	declROnly("HDF5", StorageFormatHDF5, "StorageFormat = HDF5 sets per-quantity HDF5 output")
	declLValue("StorageFormat", &sformat{}, "Storage format: ZARR or HDF5")
}

