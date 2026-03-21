package main

import (
	"fmt"
	"os"

	"github.com/scigolib/hdf5"
)

func main() {
	os.Remove("test2.h5")
	fw, err := hdf5.CreateForWrite("test2.h5", hdf5.CreateTruncate)
	if err != nil { fmt.Println(err); os.Exit(1) }
	defer fw.Close()

	// Test 1: dataset in group (contiguous)
	fw.CreateGroup("/g1")
	ds1, _ := fw.CreateDataset("/g1/flat", hdf5.Float32, []uint64{4})
	ds1.Write([]float32{1.0, 2.0, 3.0, 4.0})
	ds1.Close()

	// Test 2: chunked, no compression
	ds2, _ := fw.CreateDataset("/chunked_only", hdf5.Float32, []uint64{8},
		hdf5.WithChunkDims([]uint64{4}),
	)
	ds2.Write([]float32{10, 20, 30, 40, 50, 60, 70, 80})
	ds2.Close()

	// Test 3: chunked + compressed
	ds3, _ := fw.CreateDataset("/chunked_gzip", hdf5.Float32, []uint64{8},
		hdf5.WithChunkDims([]uint64{4}),
		hdf5.WithGZIPCompression(1),
	)
	ds3.Write([]float32{100, 200, 300, 400, 500, 600, 700, 800})
	ds3.Close()

	// Test 4: multidim chunked + compressed
	ds4, _ := fw.CreateDataset("/multi_chunked", hdf5.Float32, []uint64{4, 3},
		hdf5.WithChunkDims([]uint64{2, 3}),
		hdf5.WithGZIPCompression(1),
	)
	ds4.Write([]float32{1,2,3, 4,5,6, 7,8,9, 10,11,12})
	ds4.Close()

	// Test 5: chunked in group
	fw.CreateGroup("/g2")
	ds5, _ := fw.CreateDataset("/g2/data", hdf5.Float32, []uint64{4},
		hdf5.WithChunkDims([]uint64{4}),
		hdf5.WithGZIPCompression(1),
	)
	ds5.Write([]float32{111, 222, 333, 444})
	ds5.Close()

	fmt.Println("wrote test2.h5")
}
