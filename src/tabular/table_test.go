package tabular

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/DataDog/zstd"
)

func TestWriterPublishesColumnDataAcrossChunkBoundary(t *testing.T) {
	dir := newTableTestGroup(t)
	writer, err := New(dir, []string{"mx", "my"}, []string{"A/m", "A/m"})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	first := tableTestRows(0, 1023)
	if err := writer.Append(first); err != nil {
		t.Fatalf("append first rows: %v", err)
	}
	assertTableManifest(t, dir, 1023)
	assertTableColumnValues(t, dir, "mx", first[0], 0)
	assertTableColumnValues(t, dir, "my", first[1], 0)

	second := tableTestRows(1023, 2)
	if err := writer.Append(second); err != nil {
		t.Fatalf("append rows across chunk boundary: %v", err)
	}
	assertTableManifest(t, dir, 1025)
	assertTableColumnValues(t, dir, "mx", append(append([]float64(nil), first[0]...), second[0]...), 1)
	assertTableColumnValues(t, dir, "my", append(append([]float64(nil), first[1]...), second[1]...), 1)
}

func TestWriterFailureDoesNotAdvanceManifestOrCorruptCommittedPrefix(t *testing.T) {
	dir := newTableTestGroup(t)
	writer, err := New(dir, []string{"mx", "my"}, []string{"A/m", "A/m"})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	first := tableTestRows(0, 1023)
	if err := writer.Append(first); err != nil {
		t.Fatalf("append first rows: %v", err)
	}
	manifestBefore, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read initial manifest: %v", err)
	}

	// An empty directory at the next chunk path makes the second chunk
	// publication fail deterministically after chunk zero has been rewritten.
	if err := os.Mkdir(filepath.Join(dir, "mx", "1"), 0o755); err != nil {
		t.Fatalf("create deterministic chunk failure: %v", err)
	}
	second := tableTestRows(1023, 2)
	if err := writer.Append(second); err == nil {
		t.Fatal("append unexpectedly succeeded with a directory at mx/1")
	}

	manifestAfter, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest after failed append: %v", err)
	}
	if string(manifestAfter) != string(manifestBefore) {
		t.Fatalf("failed append changed manifest:\n before=%s\n after=%s", manifestBefore, manifestAfter)
	}
	assertTableManifest(t, dir, 1023)
	assertTableColumnValues(t, dir, "mx", first[0], 0)
	assertTableColumnValues(t, dir, "my", first[1], 0)

	if err := os.RemoveAll(filepath.Join(dir, "mx", "1")); err != nil {
		t.Fatalf("remove deterministic failure: %v", err)
	}
	if err := writer.Append(second); err != nil {
		t.Fatalf("retry append after fixing destination: %v", err)
	}
	assertTableManifest(t, dir, 1025)
}

func newTableTestGroup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zgroup"), []byte(`{"zarr_format":2}`), 0o644); err != nil {
		t.Fatalf("create Zarr group: %v", err)
	}
	return dir
}

func tableTestRows(start, count int) [][]float64 {
	rows := make([][]float64, 2)
	for column := range rows {
		rows[column] = make([]float64, count)
		for i := range rows[column] {
			value := float64(start+i) + 0.25
			if column == 1 {
				value = -value
			}
			rows[column][i] = value
		}
	}
	return rows
}

func assertTableManifest(t *testing.T, dir string, wantRows int) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Version != 1 || manifest.Format != "zarr" {
		t.Fatalf("manifest header = %#v", manifest)
	}
	if manifest.Rows != wantRows || manifest.ChunkRows != ChunkRows {
		t.Fatalf("manifest rows/chunk_rows = %d/%d, want %d/%d", manifest.Rows, manifest.ChunkRows, wantRows, ChunkRows)
	}
	if len(manifest.Columns) != 2 || manifest.Columns[0] != "mx" || manifest.Columns[1] != "my" {
		t.Fatalf("manifest columns = %#v", manifest.Columns)
	}
	if len(manifest.Units) != 2 || manifest.Units[0] != "A/m" || manifest.Units[1] != "A/m" {
		t.Fatalf("manifest units = %#v", manifest.Units)
	}
}

func assertTableColumnValues(t *testing.T, dir, column string, want []float64, maxChunk int) {
	t.Helper()
	for chunk := 0; chunk <= maxChunk; chunk++ {
		path := filepath.Join(dir, column, fmt.Sprint(chunk))
		compressed, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s chunk %d: %v", column, chunk, err)
		}
		if len(compressed) == 0 {
			t.Fatalf("%s chunk %d is empty", column, chunk)
		}
		raw, err := zstd.Decompress(nil, compressed)
		if err != nil {
			t.Fatalf("decompress %s chunk %d: %v", column, chunk, err)
		}
		if len(raw) != ChunkRows*8 {
			t.Fatalf("%s chunk %d raw bytes = %d, want %d", column, chunk, len(raw), ChunkRows*8)
		}
		start := chunk * ChunkRows
		end := min(start+ChunkRows, len(want))
		for i := start; i < end; i++ {
			got := math.Float64frombits(binary.LittleEndian.Uint64(raw[(i-start)*8:]))
			if got != want[i] {
				t.Fatalf("%s row %d = %v, want %v", column, i, got, want[i])
			}
		}
	}
}
