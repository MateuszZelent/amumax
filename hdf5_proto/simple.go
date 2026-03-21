package main

import (
	"fmt"
	"os"

	"github.com/scigolib/hdf5"
)

func main() {
	os.Remove("simple.h5")
	fw, err := hdf5.CreateForWrite("simple.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fw.Close()

	// Simplest possible: 1D float64, contiguous, no compression
	data := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	ds, err := fw.CreateDataset("/values", hdf5.Float64, []uint64{5})
	if err != nil {
		fmt.Println("CreateDataset:", err)
		os.Exit(1)
	}
	ds.Write(data)
	ds.Close()

	fmt.Println("wrote simple.h5")
}
