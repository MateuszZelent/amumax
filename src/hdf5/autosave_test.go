package hdf5

import (
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"testing"
)

// TestAutoSave_IncrementalSimulation simulates a real amumax simulation:
// incremental SaveArray calls, timestamps at shutdown, table flush at shutdown.
func TestAutoSave_IncrementalSimulation(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz := 32, 32, 1
	ncomp := 3
	totalSteps := 50
	saveInterval := 5

	savedSteps := []int{}
	timestamps := []float64{}

	for step := 0; step < totalSteps; step++ {
		simTime := float64(step) * 1e-12

		if step%saveInterval == 0 {
			data4 := make([][][][]float32, ncomp)
			for c := 0; c < ncomp; c++ {
				data4[c] = make([][][]float32, nz)
				for z := 0; z < nz; z++ {
					data4[c][z] = make([][]float32, ny)
					for y := 0; y < ny; y++ {
						data4[c][z][y] = make([]float32, nx)
						for x := 0; x < nx; x++ {
							data4[c][z][y][x] = float32(
								math.Sin(float64(x)*0.1+float64(step)*0.01) *
									math.Cos(float64(y)*0.1) * float64(c+1) * 0.33,
							)
						}
					}
				}
			}

			idx := step / saveInterval
			if err := mw.SaveArray("m", idx, data4, [3]int{nx, ny, nz}, ncomp); err != nil {
				t.Fatalf("SaveArray step %d: %v", step, err)
			}
			savedSteps = append(savedSteps, idx)
			timestamps = append(timestamps, simTime)
		}
	}

	// Shutdown: write timestamps + table
	mw.SaveTimestamps("m", timestamps)

	tableMx := make([]float64, totalSteps)
	for i := range tableMx {
		tableMx[i] = math.Cos(float64(i) * 0.01)
	}
	mw.SaveTableColumn("mx", tableMx)
	mw.Close()

	// Verify: each saved step readable
	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	for _, step := range savedSteps {
		slice, err := ReadArray(h5File, fmt.Sprintf("/%d", step))
		if err != nil {
			t.Fatalf("ReadArray /%d: %v", step, err)
		}
		if slice.NComp() != ncomp || slice.Size() != [3]int{nx, ny, nz} {
			t.Errorf("step %d shape wrong", step)
		}
	}

	t.Logf("Incremental simulation: %d snapshots in m/m.h5, verified", len(savedSteps))
}

// TestAutoSave_ConcurrentWriters tests thread safety with concurrent goroutines.
func TestAutoSave_ConcurrentWriters(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz := 16, 16, 1
	numSteps := 20

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Goroutine 1: magnetization
	wg.Add(1)
	go func() {
		defer wg.Done()
		for step := 0; step < numSteps; step++ {
			data4 := makeTestData4D(3, nz, ny, nx)
			for c := range data4 {
				for z := range data4[c] {
					for y := range data4[c][z] {
						for x := range data4[c][z][y] {
							data4[c][z][y][x] = float32(step) + float32(c)*0.1
						}
					}
				}
			}
			if err := mw.SaveArray("m", step, data4, [3]int{nx, ny, nz}, 3); err != nil {
				errCh <- fmt.Errorf("m step %d: %w", step, err)
				return
			}
		}
	}()

	// Goroutine 2: energy density at half rate
	wg.Add(1)
	go func() {
		defer wg.Done()
		for step := 0; step < numSteps; step += 2 {
			data4 := makeTestData4D(1, nz, ny, nx)
			if err := mw.SaveArray("Edens", step/2, data4, [3]int{nx, ny, nz}, 1); err != nil {
				errCh <- fmt.Errorf("Edens step %d: %w", step/2, err)
				return
			}
		}
	}()

	// Goroutine 3: B_eff at quarter rate
	wg.Add(1)
	go func() {
		defer wg.Done()
		for step := 0; step < numSteps; step += 4 {
			data4 := makeTestData4D(3, nz, ny, nx)
			if err := mw.SaveArray("B_eff", step/4, data4, [3]int{nx, ny, nz}, 3); err != nil {
				errCh <- fmt.Errorf("B_eff step %d: %w", step/4, err)
				return
			}
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("Concurrent write error: %v", err)
	}

	mw.Close()

	// Verify: each quantity got its own file
	for _, qty := range []string{"m", "Edens", "B_eff"} {
		h5File := filepath.Join(mw.baseDir, qty, qty+".h5")
		slice, err := ReadArray(h5File, "/0")
		if err != nil {
			t.Fatalf("ReadArray %s/0: %v", qty, err)
		}
		if qty == "Edens" && slice.NComp() != 1 {
			t.Errorf("Edens NComp: got %d, want 1", slice.NComp())
		}
		if qty != "Edens" && slice.NComp() != 3 {
			t.Errorf("%s NComp: got %d, want 3", qty, slice.NComp())
		}
	}

	t.Logf("Concurrent writes: 3 goroutines → 3 separate HDF5 files, verified")
}

// TestAutoSave_RapidSmallWrites tests 200 rapid writes.
func TestAutoSave_RapidSmallWrites(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz := 4, 4, 1
	numSteps := 200

	for step := 0; step < numSteps; step++ {
		data4 := makeTestData4D(3, nz, ny, nx)
		for c := range data4 {
			for z := range data4[c] {
				for y := range data4[c][z] {
					for x := range data4[c][z][y] {
						data4[c][z][y][x] = float32(step)
					}
				}
			}
		}
		if err := mw.SaveArray("m", step, data4, [3]int{nx, ny, nz}, 3); err != nil {
			t.Fatalf("SaveArray step %d: %v", step, err)
		}
	}

	mw.Close()

	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	for _, step := range []int{0, 1, 50, 99, 100, 150, 199} {
		slice, err := ReadArray(h5File, fmt.Sprintf("/%d", step))
		if err != nil {
			t.Fatalf("ReadArray /%d: %v", step, err)
		}
		got := slice.Tensors()[0][0][0][0]
		if got != float32(step) {
			t.Errorf("step %d: got %v, want %v", step, got, float32(step))
		}
	}

	t.Logf("Rapid write test: 200 steps in m/m.h5, verified 7")
}

// TestAutoSave_MultipleQuantities tests 5 different quantities.
func TestAutoSave_MultipleQuantities(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz := 16, 16, 1

	quantities := []struct {
		name  string
		ncomp int
		steps int
	}{
		{"m", 3, 20},
		{"B_eff", 3, 10},
		{"Edens_total", 1, 10},
		{"torque", 3, 5},
		{"B_ext", 3, 20},
	}

	for _, q := range quantities {
		for step := 0; step < q.steps; step++ {
			data4 := makeTestData4D(q.ncomp, nz, ny, nx)
			for c := range data4 {
				for z := range data4[c] {
					for y := range data4[c][z] {
						for x := range data4[c][z][y] {
							data4[c][z][y][x] = float32(step*100 + len(q.name))
						}
					}
				}
			}
			mw.SaveArray(q.name, step, data4, [3]int{nx, ny, nz}, q.ncomp)
		}
		times := make([]float64, q.steps)
		for i := range times {
			times[i] = float64(i) * 1e-12
		}
		mw.SaveTimestamps(q.name, times)
	}

	mw.Close()

	// Verify: each quantity has its own file
	for _, q := range quantities {
		h5File := filepath.Join(mw.baseDir, q.name, q.name+".h5")
		slice, err := ReadArray(h5File, "/0")
		if err != nil {
			t.Fatalf("ReadArray %s/0: %v", q.name, err)
		}
		if slice.NComp() != q.ncomp {
			t.Errorf("%s NComp: got %d, want %d", q.name, slice.NComp(), q.ncomp)
		}

		lastStep := q.steps - 1
		slice, err = ReadArray(h5File, fmt.Sprintf("/%d", lastStep))
		if err != nil {
			t.Fatalf("ReadArray %s/%d: %v", q.name, lastStep, err)
		}
		got := slice.Tensors()[0][0][0][0]
		want := float32(lastStep*100 + len(q.name))
		if got != want {
			t.Errorf("%s/%d: got %v, want %v", q.name, lastStep, got, want)
		}
	}

	t.Logf("Multiple quantities: 5 quantities → 5 separate HDF5 files, verified")
}
