// Package tabular stores aligned Zarr table columns behind one committed prefix.
package tabular

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/DataDog/zstd"
	"github.com/MathieuMoalic/amumax/src/fsutil"
)

const ChunkRows = 1024

type Manifest struct {
	Version   int      `json:"version"`
	Format    string   `json:"format"`
	Columns   []string `json:"columns"`
	Units     []string `json:"units"`
	Rows      int      `json:"committed_rows"`
	ChunkRows int      `json:"chunk_rows"`
}

type Writer struct {
	dir      string
	manifest Manifest
	tail     [][]float64
}

// New creates a table in an otherwise empty Zarr group.
func New(dir string, columns, units []string) (*Writer, error) {
	if len(columns) == 0 || len(columns) != len(units) {
		return nil, errors.New("table: invalid columns or units")
	}
	seen := make(map[string]bool)
	for _, column := range columns {
		if column == "" || column == "." || column == ".." || strings.HasPrefix(column, ".") || strings.ContainsAny(column, "/\\") || column == "manifest.json" || seen[column] {
			return nil, fmt.Errorf("table: invalid or duplicate column %q", column)
		}
		seen[column] = true
	}
	entries, err := fsutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry != ".zgroup" {
			return nil, fmt.Errorf("table: directory is not empty: %s", dir)
		}
	}
	w := &Writer{dir: dir, manifest: Manifest{Version: 1, Format: "zarr", Columns: append([]string(nil), columns...), Units: append([]string(nil), units...), ChunkRows: ChunkRows}, tail: make([][]float64, len(columns))}
	if err := w.publish(0); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) publish(rows int) error {
	m := w.manifest
	m.Rows = rows
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return fsutil.Put(filepath.Join(w.dir, "manifest.json"), b)
}

// Append writes all columns before publishing the new common row count.
func (w *Writer) Append(values [][]float64) error {
	if len(values) != len(w.manifest.Columns) {
		return errors.New("table: column count changed")
	}
	n := len(values[0])
	for _, column := range values {
		if len(column) != n {
			return errors.New("table: unaligned rows")
		}
	}
	if n == 0 {
		return nil
	}
	tail := make([][]float64, len(values))
	for c := range tail {
		tail[c] = append([]float64(nil), w.tail[c]...)
	}
	rows := w.manifest.Rows
	for start := 0; start < n; {
		part := min(n-start, ChunkRows-rows%ChunkRows)
		for c := range values {
			tail[c] = append(tail[c], values[c][start:start+part]...)
		}
		if err := w.writeChunk(rows/ChunkRows, tail); err != nil {
			return err
		}
		rows += part
		start += part
		if rows%ChunkRows == 0 {
			for c := range tail {
				tail[c] = nil
			}
		}
	}
	for c, name := range w.manifest.Columns {
		metadata := map[string]any{"chunks": [1]int{ChunkRows}, "compressor": map[string]any{"id": "zstd", "level": 1}, "dtype": "<f8", "fill_value": 0, "filters": nil, "order": "C", "shape": [1]int{rows}, "zarr_format": 2}
		b, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		if err := fsutil.Put(filepath.Join(w.dir, name, ".zarray"), b); err != nil {
			return err
		}
		attrs, err := json.Marshal(map[string]string{"name": name, "unit": w.manifest.Units[c]})
		if err != nil {
			return err
		}
		if err := fsutil.Put(filepath.Join(w.dir, name, ".zattrs"), attrs); err != nil {
			return err
		}
	}
	if err := w.publish(rows); err != nil {
		return err
	}
	w.manifest.Rows, w.tail = rows, tail
	return nil
}

func (w *Writer) writeChunk(index int, values [][]float64) error {
	for c, name := range w.manifest.Columns {
		raw := make([]byte, ChunkRows*8)
		for i, value := range values[c] {
			binary.LittleEndian.PutUint64(raw[8*i:], math.Float64bits(value))
		}
		compressed, err := zstd.CompressLevel(nil, raw, 1)
		if err != nil {
			return err
		}
		if err := fsutil.Put(filepath.Join(w.dir, name, fmt.Sprint(index)), compressed); err != nil {
			return err
		}
	}
	return nil
}
