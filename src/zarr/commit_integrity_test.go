package zarr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MathieuMoalic/amumax/src/fsutil"
)

func TestCommitFramePublishesOnlyCompleteFrames(t *testing.T) {
	dir := t.TempDir()
	size := [3]int{1, 1, 1}

	firstHead, err := PrepareFrame(dir, size, 1, 0, 1, 1, 1, 1, "m", "A/m")
	if err != nil {
		t.Fatalf("prepare first frame: %v", err)
	}
	firstPayload := []byte("first complete payload")
	firstRecord := putCommitTestChunk(t, dir, "0.0.0.0.0", firstPayload)
	if err := CommitFrame(dir, firstHead, 123.456789, []ChunkRecord{firstRecord}); err != nil {
		t.Fatalf("commit first frame: %v", err)
	}

	firstCommit, err := ReadCommit(dir)
	if err != nil {
		t.Fatalf("read first commit: %v", err)
	}
	if firstCommit.CommittedSteps != 1 {
		t.Fatalf("first committed steps = %d, want 1", firstCommit.CommittedSteps)
	}
	if firstCommit.LastRecordSHA256 == "" {
		t.Fatal("first commit has no record checksum")
	}

	secondHead, err := PrepareFrame(dir, size, 1, 1, 1, 1, 1, 1, "m", "A/m")
	if err != nil {
		t.Fatalf("prepare second frame: %v", err)
	}
	secondPayload := []byte("second complete payload")
	secondRecord := putCommitTestChunk(t, dir, "1.0.0.0.0", secondPayload)
	if err := CommitFrame(dir, secondHead, 987.654321, []ChunkRecord{secondRecord}); err != nil {
		t.Fatalf("commit second frame: %v", err)
	}

	secondCommit, err := ReadCommit(dir)
	if err != nil {
		t.Fatalf("read second commit: %v", err)
	}
	if secondCommit.CommittedSteps != 2 {
		t.Fatalf("second committed steps = %d, want 2", secondCommit.CommittedSteps)
	}
	if secondCommit.LastRecordSHA256 == "" || secondCommit.LastRecordSHA256 == firstCommit.LastRecordSHA256 {
		t.Fatalf("second commit checksum = %q, want a new non-empty checksum", secondCommit.LastRecordSHA256)
	}

	badHead, err := PrepareFrame(dir, size, 1, 2, 1, 1, 1, 1, "m", "A/m")
	if err != nil {
		t.Fatalf("prepare invalid third frame: %v", err)
	}
	badRecord := putCommitTestChunk(t, dir, "2.0.0.0.0", []byte("third payload"))
	badRecord.Size++
	if err := CommitFrame(dir, badHead, 111.222333, []ChunkRecord{badRecord}); err == nil {
		t.Fatal("commit accepted a record with an incorrect size")
	}

	afterRejected, err := ReadCommit(dir)
	if err != nil {
		t.Fatalf("read commit after rejected frame: %v", err)
	}
	if afterRejected.CommittedSteps != secondCommit.CommittedSteps {
		t.Fatalf("rejected frame advanced committed steps to %d, want %d", afterRejected.CommittedSteps, secondCommit.CommittedSteps)
	}
	if afterRejected.LastRecordSHA256 != secondCommit.LastRecordSHA256 {
		t.Fatalf("rejected frame changed last checksum from %q to %q", secondCommit.LastRecordSHA256, afterRejected.LastRecordSHA256)
	}

	if err := PublishProjection(dir); err != nil {
		t.Fatalf("publish projection: %v", err)
	}
	assertCommitProjection(t, dir, secondCommit.CommittedSteps)
	assertCommitTimestamps(t, dir, "123.456789", "987.654321")
}

func putCommitTestChunk(t *testing.T, dir, key string, payload []byte) ChunkRecord {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, key)), 0o755); err != nil {
		t.Fatalf("create chunk directory: %v", err)
	}
	if err := fsutil.Put(filepath.Join(dir, key), payload); err != nil {
		t.Fatalf("write chunk %q: %v", key, err)
	}
	digest := sha256.Sum256(payload)
	return ChunkRecord{Key: key, Size: len(payload), SHA256: hex.EncodeToString(digest[:])}
}

func assertCommitProjection(t *testing.T, dir string, committedSteps int) {
	t.Helper()
	var zarrayCount, attrsCount int
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		switch entry.Name() {
		case ".zarray":
			zarrayCount++
			var metadata map[string]any
			if err := json.Unmarshal(contents, &metadata); err != nil {
				return fmt.Errorf("%s is invalid JSON: %w", path, err)
			}
			shape, ok := metadata["shape"].([]any)
			if !ok || len(shape) == 0 {
				return fmt.Errorf("%s has no Zarr shape", path)
			}
			firstDimension, ok := shape[0].(float64)
			if !ok || int(firstDimension) != committedSteps {
				return fmt.Errorf("%s shape[0] = %v, want %d", path, shape[0], committedSteps)
			}
		case ".zattrs":
			attrsCount++
			var attrs map[string]any
			if err := json.Unmarshal(contents, &attrs); err != nil {
				return fmt.Errorf("%s is invalid JSON: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if zarrayCount == 0 {
		t.Fatal("PublishProjection produced no .zarray")
	}
	if attrsCount == 0 {
		t.Fatal("PublishProjection produced no .zattrs")
	}
}

func assertCommitTimestamps(t *testing.T, dir string, timestamps ...string) {
	t.Helper()
	var allJSON bytes.Buffer
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".json" || entry.Name() == ".zattrs" || entry.Name() == ".zarray" {
			allJSON.Write(contents)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, timestamp := range timestamps {
		if !bytes.Contains(allJSON.Bytes(), []byte(timestamp)) {
			t.Fatalf("committed timestamp %s not present in JSON projection", timestamp)
		}
	}
}
