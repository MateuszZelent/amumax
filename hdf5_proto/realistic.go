package main

import (
	"fmt"
	"math"
	"os"

	"github.com/scigolib/hdf5"
)

func main() {
	Nz, Ny, Nx, Nc := 4, 32, 64, 3
	nSteps := 5

	filename := "amumax_test.h5"
	os.Remove(filename)

	fw, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
	if err != nil { fmt.Println(err); os.Exit(1) }
	defer fw.Close()

	fw.CreateGroup("/m")

	// Single large contiguous 5D dataset (T, Nz, Ny, Nx, C)
	frameSize := Nz * Ny * Nx * Nc
	totalSize := nSteps * frameSize
	allData := make([]float32, totalSize)
	for step := 0; step < nSteps; step++ {
		for i := 0; i < frameSize; i++ {
			allData[step*frameSize+i] = float32(math.Sin(float64(step)*0.1 + float64(i)*0.001))
		}
	}

	dims := []uint64{uint64(nSteps), uint64(Nz), uint64(Ny), uint64(Nx), uint64(Nc)}
	ds, err := fw.CreateDataset("/m/data", hdf5.Float32, dims)
	if err != nil { fmt.Println("CreateDataset:", err); os.Exit(1) }
	ds.Write(allData)
	ds.Close()

	// Timestamps
	times := make([]float64, nSteps)
	for i := range times { times[i] = float64(i) * 1e-12 }
	tds, _ := fw.CreateDataset("/m/t", hdf5.Float64, []uint64{uint64(nSteps)})
	tds.Write(times)
	tds.Close()

	// Mesh metadata
	mg, _ := fw.CreateGroup("/mesh")
	mg.WriteAttribute("dx", 1e-9)
	mg.WriteAttribute("dy", 1e-9)
	mg.WriteAttribute("dz", 1e-9)
	mg.WriteAttribute("Nx", int64(Nx))
	mg.WriteAttribute("Ny", int64(Ny))
	mg.WriteAttribute("Nz", int64(Nz))

	info, _ := os.Stat(filename)
	fmt.Printf("File: %s, size: %.1f KB (raw: %.1f KB)\n", filename, float64(info.Size())/1024, float64(totalSize*4)/1024)
}
