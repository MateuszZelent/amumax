package zarr

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/MathieuMoalic/amumax/src/fsutil"
	"github.com/MathieuMoalic/amumax/src/log"
)

type ztableFile struct {
	Chunks     [1]int  `json:"chunks"`
	Compressor []int   `json:"compressor"`
	Dtype      string  `json:"dtype"`
	FillValue  float64 `json:"fill_value"`
	Filters    []int   `json:"filters"`
	Order      string  `json:"order"`
	Shape      [1]int  `json:"shape"`
	ZarrFormat int     `json:"zarr_format"`
}

func SaveFileTableZarray(path string, zTableAutoSaveStep int) {
	if !pathExists(path) {
		log.Log.PanicIfError(errors.New("error: `%s` does not exist"))
	}
	z := ztableFile{}
	z.Dtype = `<f8`
	z.FillValue = 0.0
	z.Order = "C"
	z.ZarrFormat = 2
	z.Chunks = [1]int{zTableAutoSaveStep + 1}
	z.Shape = [1]int{zTableAutoSaveStep + 1}

	b, err := json.MarshalIndent(z, "", "\t")
	log.Log.PanicIfError(err)
	log.Log.PanicIfError(fsutil.Put(path+"/.zarray", b))
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
