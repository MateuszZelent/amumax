package main

import (
	"fmt"
	"math"
	"os"

	"github.com/scigolib/hdf5"
)

func main() {
	// Simulate amumax output: magnetization (T, Nz, Ny, Nx, 3) float32
	Nz, Ny, Nx, Nc := 4, 32, 64, 3
	nSteps := 5

	filename := "test_output.h5"
	os.Remove(filename)

	// Create new HDF5 file (SuperblockV0 = max compat with h5py)
	fw, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateForWrite: %v\n", err)
		os.Exit(1)
	}
	defer fw.Close()

	// Create group "m"
	mGroup, err := fw.CreateGroup("/m")
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateGroup: %v\n", err)
		os.Exit(1)
	}
	mGroup.WriteAttribute("unit", "")

	// Write ALL data at once (non-incremental approach)
	frameSize := Nz * Ny * Nx * Nc
	totalSize := nSteps * frameSize

	allData := make([]float32, totalSize)
	for step := 0; step < nSteps; step++ {
		for i := 0; i < frameSize; i++ {
			allData[step*frameSize+i] = float32(math.Sin(float64(step)*0.1 + float64(i)*0.001))
		}
	}

	dims := []uint64{uint64(nSteps), uint64(Nz), uint64(Ny), uint64(Nx), uint64(Nc)}
	chunkDims := []uint64{1, uint64(Nz), uint64(Ny), uint64(Nx), uint64(Nc)}

	ds, err := fw.CreateDataset(
		"/m/data",
		hdf5.Float32,
		dims,
		hdf5.WithChunkDims(chunkDims),
		hdf5.WithGZIPCompression(1),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateDataset: %v\n", err)
		os.Exit(1)
	}
	if err := ds.Write(allData); err != nil {
		fmt.Fprintf(os.Stderr, "Write: %v\n", err)
		os.Exit(1)
	}
	ds.Close()
	fmt.Println("Wrote magnetization data")

	// Create timestamps dataset
	times := make([]float64, nSteps)
	for i := range times {
		times[i] = float64(i) * 1e-12
	}
	tDS, err := fw.CreateDataset(
		"/m/t",
		hdf5.Float64,
		[]uint64{uint64(nSteps)},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateDataset t: %v\n", err)
		os.Exit(1)
	}
	tDS.Write(times)
	tDS.Close()

	// Mesh metadata group
	meshGroup, _ := fw.CreateGroup("/mesh")
	meshGroup.WriteAttribute("dx", 1e-9)
	meshGroup.WriteAttribute("dy", 1e-9)
	meshGroup.WriteAttribute("dz", 1e-9)
	meshGroup.WriteAttribute("Nx", int64(Nx))
	meshGroup.WriteAttribute("Ny", int64(Ny))
	meshGroup.WriteAttribute("Nz", int64(Nz))

	fmt.Printf("\nDone! File: %s\n", filename)
	info, _ := os.Stat(filename)
	fmt.Printf("File size:         %.2f KB\n", float64(info.Size())/1024)
	fmt.Printf("Raw data equiv:    %.2f KB\n", float64(totalSize*4)/1024)
	fmt.Printf("Compression ratio: %.1fx\n", float64(totalSize*4)/float64(info.Size()))
}
