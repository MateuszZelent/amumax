package hdf5

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// helper: create test 4D data [ncomp][nz][ny][nx]
func makeTestData4D(ncomp, nz, ny, nx int) [][][][]float32 {
	data4 := make([][][][]float32, ncomp)
	for c := range data4 {
		data4[c] = make([][][]float32, nz)
		for z := range data4[c] {
			data4[c][z] = make([][]float32, ny)
			for y := range data4[c][z] {
				data4[c][z][y] = make([]float32, nx)
				for x := range data4[c][z][y] {
					data4[c][z][y][x] = float32(c*1000 + z*100 + y*10 + x)
				}
			}
		}
	}
	return data4
}

// newTestMultiWriter creates a MultiWriter in a temp directory.
func newTestMultiWriter(t *testing.T) *MultiWriter {
	t.Helper()
	baseDir := t.TempDir() + "/"
	return &MultiWriter{
		baseDir: baseDir,
		writers: make(map[string]*Writer),
	}
}

// ---------- Writer tests ----------

func TestCreateAndClose(t *testing.T) {
	mw := newTestMultiWriter(t)

	// SaveArray will lazily create m/m.h5
	data4 := makeTestData4D(1, 1, 2, 2)
	if err := mw.SaveArray("m", 0, data4, [3]int{2, 2, 1}, 1); err != nil {
		t.Fatalf("SaveArray: %v", err)
	}

	// File should exist
	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	if _, err := os.Stat(h5File); os.IsNotExist(err) {
		t.Fatalf("HDF5 file not created at %s", h5File)
	}

	mw.Close()

	// Double close should be safe
	mw.Close()
}

func TestSaveArray_SmallVector(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	data4 := makeTestData4D(3, 2, 3, 4)
	if err := mw.SaveArray("m", 0, data4, [3]int{4, 3, 2}, 3); err != nil {
		t.Fatalf("SaveArray: %v", err)
	}
}

func TestSaveArray_Scalar(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	data4 := makeTestData4D(1, 1, 10, 10)
	if err := mw.SaveArray("Edens_total", 0, data4, [3]int{10, 10, 1}, 1); err != nil {
		t.Fatalf("SaveArray scalar: %v", err)
	}

	// Verify file is in correct location
	h5File := filepath.Join(mw.baseDir, "Edens_total", "Edens_total.h5")
	if _, err := os.Stat(h5File); os.IsNotExist(err) {
		t.Fatalf("Expected file at %s", h5File)
	}
}

func TestSaveArray_MultipleSteps(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	for step := 0; step < 5; step++ {
		data4 := makeTestData4D(3, 1, 4, 4)
		for c := range data4 {
			for z := range data4[c] {
				for y := range data4[c][z] {
					for x := range data4[c][z][y] {
						data4[c][z][y][x] += float32(step * 10000)
					}
				}
			}
		}
		if err := mw.SaveArray("m", step, data4, [3]int{4, 4, 1}, 3); err != nil {
			t.Fatalf("SaveArray step %d: %v", step, err)
		}
	}
}

func TestSaveTimestamps(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	// need at least one array to create the file
	data4 := makeTestData4D(1, 1, 2, 2)
	mw.SaveArray("m", 0, data4, [3]int{2, 2, 1}, 1)

	times := []float64{0.0, 1e-12, 2e-12, 3e-12, 4e-12}
	if err := mw.SaveTimestamps("m", times); err != nil {
		t.Fatalf("SaveTimestamps: %v", err)
	}
}

func TestSaveTableColumn(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	steps := make([]float64, 100)
	mx := make([]float64, 100)
	for i := 0; i < 100; i++ {
		steps[i] = float64(i)
		mx[i] = math.Cos(float64(i) * 0.1)
	}

	if err := mw.SaveTableColumn("step", steps); err != nil {
		t.Fatalf("SaveTableColumn step: %v", err)
	}
	if err := mw.SaveTableColumn("mx", mx); err != nil {
		t.Fatalf("SaveTableColumn mx: %v", err)
	}

	// Verify file location
	h5File := filepath.Join(mw.baseDir, "table", "table.h5")
	if _, err := os.Stat(h5File); os.IsNotExist(err) {
		t.Fatalf("Expected table file at %s", h5File)
	}
}

func TestMultipleQuantities_SeparateFiles(t *testing.T) {
	mw := newTestMultiWriter(t)
	defer mw.Close()

	data4 := makeTestData4D(3, 1, 4, 4)
	mw.SaveArray("m", 0, data4, [3]int{4, 4, 1}, 3)
	mw.SaveArray("B_eff", 0, data4, [3]int{4, 4, 1}, 3)

	scalar := makeTestData4D(1, 1, 4, 4)
	mw.SaveArray("Edens", 0, scalar, [3]int{4, 4, 1}, 1)

	// Each quantity should have its own file
	for _, name := range []string{"m", "B_eff", "Edens"} {
		h5File := filepath.Join(mw.baseDir, name, name+".h5")
		if _, err := os.Stat(h5File); os.IsNotExist(err) {
			t.Errorf("Expected file at %s", h5File)
		}
	}
}

// ---------- Roundtrip (write + read) tests ----------

func TestRoundtrip_Vector3D(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz, ncomp := 4, 3, 2, 3
	data4 := makeTestData4D(ncomp, nz, ny, nx)

	if err := mw.SaveArray("m", 0, data4, [3]int{nx, ny, nz}, ncomp); err != nil {
		t.Fatalf("SaveArray: %v", err)
	}
	mw.Close()

	// Read from m/m.h5, dataset /0
	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	slice, err := ReadArray(h5File, "/0")
	if err != nil {
		t.Fatalf("ReadArray: %v", err)
	}

	if slice.NComp() != ncomp {
		t.Errorf("NComp: got %d, want %d", slice.NComp(), ncomp)
	}
	if slice.Size() != [3]int{nx, ny, nz} {
		t.Errorf("Size: got %v, want [%d %d %d]", slice.Size(), nx, ny, nz)
	}

	tensors := slice.Tensors()
	for c := 0; c < ncomp; c++ {
		for z := 0; z < nz; z++ {
			for y := 0; y < ny; y++ {
				for x := 0; x < nx; x++ {
					got := tensors[c][z][y][x]
					want := data4[c][z][y][x]
					if got != want {
						t.Errorf("[%d][%d][%d][%d]: got %v, want %v", c, z, y, x, got, want)
					}
				}
			}
		}
	}
}

func TestRoundtrip_Scalar(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz, ncomp := 8, 8, 1, 1
	data4 := makeTestData4D(ncomp, nz, ny, nx)

	mw.SaveArray("Edens", 0, data4, [3]int{nx, ny, nz}, ncomp)
	mw.Close()

	h5File := filepath.Join(mw.baseDir, "Edens", "Edens.h5")
	slice, err := ReadArray(h5File, "/0")
	if err != nil {
		t.Fatalf("ReadArray: %v", err)
	}

	if slice.NComp() != 1 || slice.Size() != [3]int{8, 8, 1} {
		t.Errorf("Shape wrong: NComp=%d Size=%v", slice.NComp(), slice.Size())
	}
}

func TestRoundtrip_MultipleSteps(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz, ncomp := 3, 3, 1, 3
	numSteps := 5
	allData := make([][][][][]float32, numSteps)

	for step := 0; step < numSteps; step++ {
		data4 := makeTestData4D(ncomp, nz, ny, nx)
		for c := range data4 {
			for z := range data4[c] {
				for y := range data4[c][z] {
					for x := range data4[c][z][y] {
						data4[c][z][y][x] += float32(step * 10000)
					}
				}
			}
		}
		allData[step] = data4
		mw.SaveArray("m", step, data4, [3]int{nx, ny, nz}, ncomp)
	}
	mw.Close()

	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	for step := 0; step < numSteps; step++ {
		dsPath := fmt.Sprintf("/%d", step)
		slice, err := ReadArray(h5File, dsPath)
		if err != nil {
			t.Fatalf("ReadArray step %d: %v", step, err)
		}

		tensors := slice.Tensors()
		for c := 0; c < ncomp; c++ {
			for z := 0; z < nz; z++ {
				for y := 0; y < ny; y++ {
					for x := 0; x < nx; x++ {
						got := tensors[c][z][y][x]
						want := allData[step][c][z][y][x]
						if got != want {
							t.Errorf("step %d [%d][%d][%d][%d]: got %v, want %v", step, c, z, y, x, got, want)
						}
					}
				}
			}
		}
	}
}

func TestRoundtrip_LargeArray(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz, ncomp := 128, 64, 4, 3
	data4 := makeTestData4D(ncomp, nz, ny, nx)

	mw.SaveArray("m", 0, data4, [3]int{nx, ny, nz}, ncomp)
	mw.Close()

	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	slice, err := ReadArray(h5File, "/0")
	if err != nil {
		t.Fatalf("ReadArray large: %v", err)
	}

	if slice.NComp() != ncomp || slice.Size() != [3]int{nx, ny, nz} {
		t.Errorf("Shape wrong: NComp=%d Size=%v", slice.NComp(), slice.Size())
	}

	tensors := slice.Tensors()
	checks := [][4]int{{0, 0, 0, 0}, {1, 2, 31, 63}, {2, 3, 63, 127}, {0, 0, 32, 100}}
	for _, idx := range checks {
		c, z, y, x := idx[0], idx[1], idx[2], idx[3]
		if tensors[c][z][y][x] != data4[c][z][y][x] {
			t.Errorf("[%d][%d][%d][%d]: got %v, want %v", c, z, y, x, tensors[c][z][y][x], data4[c][z][y][x])
		}
	}

	t.Logf("Large array roundtrip OK: %dx%dx%dx%d = %d floats", nx, ny, nz, ncomp, nx*ny*nz*ncomp)
}

func TestReadArray_NonexistentFile(t *testing.T) {
	_, err := ReadArray("/tmp/does_not_exist.h5", "/0")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestReadArray_NonexistentDataset(t *testing.T) {
	mw := newTestMultiWriter(t)
	data4 := makeTestData4D(1, 1, 2, 2)
	mw.SaveArray("m", 0, data4, [3]int{2, 2, 1}, 1)
	mw.Close()

	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	_, err := ReadArray(h5File, "/999")
	if err == nil {
		t.Fatal("Expected error for nonexistent dataset")
	}
}

// ---------- Full simulation-like test ----------

func TestFullSimulation(t *testing.T) {
	mw := newTestMultiWriter(t)

	nx, ny, nz := 16, 16, 1

	// Save 10 magnetization steps
	for step := 0; step < 10; step++ {
		data4 := makeTestData4D(3, nz, ny, nx)
		for c := range data4 {
			for z := range data4[c] {
				for y := range data4[c][z] {
					for x := range data4[c][z][y] {
						data4[c][z][y][x] = float32(step)*0.1 + float32(c)*0.01
					}
				}
			}
		}
		mw.SaveArray("m", step, data4, [3]int{nx, ny, nz}, 3)
	}

	// Save timestamps
	times := make([]float64, 10)
	for i := range times {
		times[i] = float64(i) * 1e-12
	}
	mw.SaveTimestamps("m", times)

	// Save table data
	for i := 0; i < 100; i++ {
		// Each column individually
	}
	tableMx := make([]float64, 100)
	for i := range tableMx {
		tableMx[i] = math.Cos(float64(i) * 0.01)
	}
	mw.SaveTableColumn("mx", tableMx)

	// Save a scalar quantity
	scalar := makeTestData4D(1, nz, ny, nx)
	mw.SaveArray("Edens_total", 0, scalar, [3]int{nx, ny, nz}, 1)

	mw.Close()

	// Verify: separate files exist
	for _, name := range []string{"m", "Edens_total", "table"} {
		h5File := filepath.Join(mw.baseDir, name, name+".h5")
		if _, err := os.Stat(h5File); os.IsNotExist(err) {
			t.Errorf("Expected file at %s", h5File)
		}
	}

	// Read back magnetization
	h5File := filepath.Join(mw.baseDir, "m", "m.h5")
	slice, err := ReadArray(h5File, "/0")
	if err != nil {
		t.Fatalf("ReadArray m/0: %v", err)
	}
	if slice.NComp() != 3 {
		t.Errorf("m NComp: got %d, want 3", slice.NComp())
	}

	// Step 5
	slice, err = ReadArray(h5File, "/5")
	if err != nil {
		t.Fatalf("ReadArray m/5: %v", err)
	}
	tensors := slice.Tensors()
	got := tensors[0][0][0][0]
	wantApprox := float32(5)*0.1 + float32(0)*0.01
	if math.Abs(float64(got-wantApprox)) > 0.001 {
		t.Errorf("m/5 [0][0][0][0]: got %v, want ~%v", got, wantApprox)
	}

	// Scalar
	h5File = filepath.Join(mw.baseDir, "Edens_total", "Edens_total.h5")
	slice, err = ReadArray(h5File, "/0")
	if err != nil {
		t.Fatalf("ReadArray Edens: %v", err)
	}
	if slice.NComp() != 1 {
		t.Errorf("Edens NComp: got %d, want 1", slice.NComp())
	}

	t.Logf("Full simulation test passed: 10 vector steps + timestamps + table + scalar, all in separate files")
}

// ---------- parseDimsFromInfo tests ----------

func TestParseDimsFromInfo(t *testing.T) {
	tests := []struct {
		name    string
		info    string
		want    []int
		wantErr bool
	}{
		{
			name: "standard format",
			info: "Dataset: float (size=4 bytes), 4D array [1 2 3 2], contiguous (address=0x277E, size=48)",
			want: []int{1, 2, 3, 2},
		},
		{
			name: "large dims",
			info: "Dataset: float (size=4 bytes), 4D array [4 64 128 3], contiguous",
			want: []int{4, 64, 128, 3},
		},
		{
			name: "1D array",
			info: "Dataset: float64 (size=8 bytes), 1D array [100], contiguous",
			want: []int{100},
		},
		{
			name:    "no dims",
			info:    "Dataset: unknown format",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDimsFromInfo(tt.info)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("dims length: got %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("dims[%d]: got %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
