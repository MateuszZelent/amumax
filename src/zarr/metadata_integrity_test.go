package zarr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataSaveKeepsLastValidJSONWhenSerializationFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zattrs")
	metadata := Metadata{
		Path: path,
		Fields: map[string]any{
			"run_id": "before-failure",
			"step":   7,
		},
	}
	metadata.Save()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read initial metadata: %v", err)
	}
	if !json.Valid(want) {
		t.Fatalf("initial metadata is not valid JSON: %q", want)
	}

	metadata.Fields["unserializable"] = func() {}
	defer func() {
		if recover() == nil {
			t.Fatal("Metadata.Save did not report the serialization failure")
		}
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read metadata after failed save: %v", readErr)
		}
		if string(got) != string(want) {
			t.Fatalf("failed save replaced valid metadata:\n got %q\nwant %q", got, want)
		}
	}()
	metadata.Save()
}

func TestMetadataInitWritesValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.zarr")
	metadata := Metadata{}
	metadata.Init(path, time.Unix(123, 0), "test-gpu")

	contents, err := os.ReadFile(path + ".zattrs")
	if err != nil {
		t.Fatalf("read initialized metadata: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(contents, &fields); err != nil {
		t.Fatalf("initialized metadata is not valid JSON: %v", err)
	}
	if fields["gpu"] != "test-gpu" {
		t.Fatalf("gpu metadata = %v, want test-gpu", fields["gpu"])
	}
}
