package hdf5

import (
	"fmt"
	"strings"

	"github.com/MathieuMoalic/amumax/src/data"
	"github.com/scigolib/hdf5"
)

// ReadArray reads a single dataset from an HDF5 file and returns it as a data.Slice.
// The dsPath should be like "/m/0" (using "/" delimiters).
// The dataset is expected to be 4D: (Nz, Ny, Nx, Ncomp) stored as float32.
// Read() auto-converts to float64, and we convert back to float32 for the Slice.
func ReadArray(filename, dsPath string) (*data.Slice, error) {
	f, err := hdf5.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("hdf5: open %s: %w", filename, err)
	}
	defer f.Close()

	// Navigate to the dataset by walking the path
	ds, err := findDataset(f, dsPath)
	if err != nil {
		return nil, fmt.Errorf("hdf5: find dataset %s in %s: %w", dsPath, filename, err)
	}

	// Read the data (auto-converts any numeric type to []float64)
	flat64, err := ds.Read()
	if err != nil {
		return nil, fmt.Errorf("hdf5: read dataset %s: %w", dsPath, err)
	}

	// Get dataset info for dimensions
	info, err := ds.Info()
	if err != nil {
		return nil, fmt.Errorf("hdf5: info dataset %s: %w", dsPath, err)
	}

	// Parse dimensions from info string (format: "... dims=[nz ny nx ncomp] ...")
	dims, err := parseDimsFromInfo(info)
	if err != nil {
		return nil, fmt.Errorf("hdf5: parse dims from %s: %w", dsPath, err)
	}
	if len(dims) != 4 {
		return nil, fmt.Errorf("hdf5: expected 4D dataset, got %dD", len(dims))
	}

	nz, ny, nx, ncomp := dims[0], dims[1], dims[2], dims[3]

	if len(flat64) != nz*ny*nx*ncomp {
		return nil, fmt.Errorf("hdf5: data size mismatch: got %d, expected %d", len(flat64), nz*ny*nx*ncomp)
	}

	// Create data.Slice and fill tensors
	// Data order in flat is [z][y][x][comp] (as written by SaveArray)
	array := data.NewSlice(ncomp, [3]int{nx, ny, nz})
	tensors := array.Tensors()

	idx := 0
	for iz := 0; iz < nz; iz++ {
		for iy := 0; iy < ny; iy++ {
			for ix := 0; ix < nx; ix++ {
				for ic := 0; ic < ncomp; ic++ {
					tensors[ic][iz][iy][ix] = float32(flat64[idx])
					idx++
				}
			}
		}
	}

	return array, nil
}

// findDataset navigates from root to find a dataset by path like "/m/0".
func findDataset(f *hdf5.File, path string) (*hdf5.Dataset, error) {
	// Clean path: remove leading/trailing slashes, split
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	// Walk the tree to find matching dataset
	var found *hdf5.Dataset
	f.Walk(func(objPath string, obj hdf5.Object) {
		if found != nil {
			return
		}
		// Normalize the walk path
		cleanWalkPath := strings.Trim(objPath, "/")
		if cleanWalkPath == path {
			if ds, ok := obj.(*hdf5.Dataset); ok {
				found = ds
			}
		}
	})

	if found != nil {
		return found, nil
	}

	// Fallback: try navigating by group children
	current := f.Root()
	for i, part := range parts {
		isLast := i == len(parts)-1
		childFound := false
		for _, child := range current.Children() {
			if child.Name() == part {
				if isLast {
					if ds, ok := child.(*hdf5.Dataset); ok {
						return ds, nil
					}
					return nil, fmt.Errorf("object at path is not a dataset")
				}
				if g, ok := child.(*hdf5.Group); ok {
					current = g
					childFound = true
					break
				}
			}
		}
		if !childFound && !isLast {
			return nil, fmt.Errorf("group %q not found", part)
		}
	}

	return nil, fmt.Errorf("dataset %q not found", path)
}

// parseDimsFromInfo extracts dimensions from Dataset.Info() string.
// The Info format is like: "Dataset: float (size=4 bytes), 4D array [1 2 3 2], contiguous ..."
func parseDimsFromInfo(info string) ([]int, error) {
	// Look for "array [" which contains the dimensions
	idx := strings.Index(info, "array [")
	if idx == -1 {
		// Fallback: look for "dims=[" or "Dimensions: ["
		idx = strings.Index(info, "dims=[")
		if idx != -1 {
			idx += len("dims=[")
		} else {
			idx = strings.Index(info, "Dimensions: [")
			if idx != -1 {
				idx += len("Dimensions: [")
			} else {
				return nil, fmt.Errorf("cannot find dimensions in info: %s", info)
			}
		}
	} else {
		idx += len("array [")
	}

	end := strings.Index(info[idx:], "]")
	if end == -1 {
		return nil, fmt.Errorf("cannot find closing bracket for dims in: %s", info)
	}

	dimStr := info[idx : idx+end]
	dimParts := strings.Fields(dimStr)

	dims := make([]int, len(dimParts))
	for i, p := range dimParts {
		p = strings.TrimRight(p, ",×x")
		var d int
		_, err := fmt.Sscanf(p, "%d", &d)
		if err != nil {
			return nil, fmt.Errorf("cannot parse dimension %q: %w", p, err)
		}
		dims[i] = d
	}

	return dims, nil
}
