package zarr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/MathieuMoalic/amumax/src/fsutil"
)

// This schema is shared with MuMax3 and the MMPP committed-store reader.
type ChunkRecord struct {
	Key    string `json:"key"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

type FrameRecord struct {
	Step           int           `json:"step"`
	Time           float64       `json:"time"`
	PreviousSHA256 string        `json:"previous_sha256"`
	Chunks         []ChunkRecord `json:"chunks"`
}

type Commit struct {
	SchemaVersion    int        `json:"schema_version"`
	Generation       int        `json:"generation"`
	CommittedSteps   int        `json:"committed_steps"`
	Metadata         zarrayFile `json:"metadata"`
	Name             string     `json:"name"`
	Unit             string     `json:"unit"`
	LastRecordSHA256 string     `json:"last_record_sha256"`
}

func putJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return fsutil.Put(path, b)
}

func ReadCommit(dir string) (Commit, error) {
	var head Commit
	b, err := fsutil.Read(filepath.Join(dir, "commit.json"))
	if err != nil {
		return head, err
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return head, fmt.Errorf("zarr: invalid commit in %s: %w", dir, err)
	}
	m := head.Metadata
	if head.SchemaVersion != 1 || head.Generation != head.CommittedSteps || head.CommittedSteps < 0 || m.Shape[0] != head.CommittedSteps || m.Chunks[0] != 1 || m.ZarrFormat != 2 || m.Dtype != "<f4" || m.Order != "C" || m.Filters != nil || m.Compressor.ID != "zstd" || (head.CommittedSteps == 0 && head.LastRecordSHA256 != "") || (head.CommittedSteps > 0 && len(head.LastRecordSHA256) != 64) {
		return head, fmt.Errorf("zarr: invalid committed schema in %s", dir)
	}
	for i := 1; i < 5; i++ {
		if m.Shape[i] <= 0 || m.Chunks[i] <= 0 || m.Chunks[i] > m.Shape[i] {
			return head, fmt.Errorf("zarr: invalid committed shape in %s", dir)
		}
	}
	return head, nil
}

// PrepareFrame initializes an empty snapshot or verifies a contiguous append.
func PrepareFrame(dir string, size [3]int, ncomp, step, cz, cy, cx, cc int, name, unit string) (Commit, error) {
	metadata := zarrayFile{Chunks: [5]int{1, cz, cy, cx, cc}, Compressor: ZstdCompressor{"zstd", 1}, Dtype: "<f4", Order: "C", Shape: [5]int{step, size[2], size[1], size[0], ncomp}, ZarrFormat: 2}
	if step < 0 {
		return Commit{}, errors.New("zarr: negative step")
	}
	for i := 1; i < 5; i++ {
		if metadata.Shape[i] <= 0 || metadata.Chunks[i] <= 0 || metadata.Chunks[i] > metadata.Shape[i] {
			return Commit{}, errors.New("zarr: invalid shape or chunks")
		}
	}
	head, err := ReadCommit(dir)
	if os.IsNotExist(err) {
		if step != 0 {
			return Commit{}, fmt.Errorf("zarr: first step must be zero, got %d", step)
		}
		if _, oldErr := fsutil.Read(filepath.Join(dir, ".zarray")); !os.IsNotExist(oldErr) {
			if oldErr == nil {
				return Commit{}, fmt.Errorf("zarr: refusing to replace existing dataset %s", dir)
			}
			return Commit{}, oldErr
		}
		head = Commit{SchemaVersion: 1, Metadata: metadata, Name: name, Unit: unit}
		if err := putJSON(filepath.Join(dir, ".zarray"), metadata); err != nil {
			return Commit{}, err
		}
		if err := putJSON(filepath.Join(dir, ".zattrs"), map[string]any{"t": []float64{}, "name": name, "unit": unit}); err != nil {
			return Commit{}, err
		}
		if err := putJSON(filepath.Join(dir, "commit.json"), head); err != nil {
			return Commit{}, err
		}
	} else if err != nil {
		return Commit{}, err
	}
	old, oldErr := json.Marshal(head.Metadata)
	want, wantErr := json.Marshal(metadata)
	if oldErr != nil || wantErr != nil || string(old) != string(want) || head.CommittedSteps != step || head.Name != name || head.Unit != unit {
		return Commit{}, fmt.Errorf("zarr: schema changed or non-contiguous step %d in %s", step, dir)
	}
	if err := fsutil.Mkdir(filepath.Join(dir, ".frames")); err != nil && !os.IsExist(err) {
		return Commit{}, err
	}
	return head, nil
}

func validateRecords(head Commit, records []ChunkRecord) error {
	m := head.Metadata
	expected := make(map[string]bool)
	for iz := 0; iz < (m.Shape[1]+m.Chunks[1]-1)/m.Chunks[1]; iz++ {
		for iy := 0; iy < (m.Shape[2]+m.Chunks[2]-1)/m.Chunks[2]; iy++ {
			for ix := 0; ix < (m.Shape[3]+m.Chunks[3]-1)/m.Chunks[3]; ix++ {
				for ic := 0; ic < (m.Shape[4]+m.Chunks[4]-1)/m.Chunks[4]; ic++ {
					expected[fmt.Sprintf("%d.%d.%d.%d.%d", head.CommittedSteps, iz, iy, ix, ic)] = true
				}
			}
		}
	}
	for _, chunk := range records {
		if !expected[chunk.Key] || chunk.Size <= 0 || len(chunk.SHA256) != 64 {
			return errors.New("zarr: invalid chunk record")
		}
		if _, err := hex.DecodeString(chunk.SHA256); err != nil {
			return errors.New("zarr: invalid chunk checksum")
		}
		delete(expected, chunk.Key)
	}
	if len(expected) != 0 {
		return errors.New("zarr: incomplete frame chunk records")
	}
	return nil
}

// CommitFrame publishes the manifest after the caller has persisted all chunks.
func CommitFrame(dir string, head Commit, timestamp float64, records []ChunkRecord) error {
	if math.IsNaN(timestamp) || math.IsInf(timestamp, 0) {
		return errors.New("zarr: invalid timestamp")
	}
	if err := validateRecords(head, records); err != nil {
		return err
	}
	// CommitFrame is a public boundary between packages. Verify the actual
	// persisted bytes, not just a caller-provided size or checksum.
	for _, record := range records {
		payload, err := fsutil.Read(filepath.Join(dir, record.Key))
		if err != nil {
			return fmt.Errorf("zarr: read chunk %s before commit: %w", record.Key, err)
		}
		digest := sha256.Sum256(payload)
		if len(payload) != record.Size || hex.EncodeToString(digest[:]) != record.SHA256 {
			return fmt.Errorf("zarr: chunk %s does not match its record", record.Key)
		}
	}
	current, err := ReadCommit(dir)
	if err != nil {
		return err
	}
	if current.Generation != head.Generation || current.LastRecordSHA256 != head.LastRecordSHA256 {
		return errors.New("zarr: commit changed during frame write")
	}
	frame := FrameRecord{Step: head.CommittedSteps, Time: timestamp, PreviousSHA256: head.LastRecordSHA256, Chunks: records}
	b, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	if err := fsutil.Put(filepath.Join(dir, ".frames", fmt.Sprintf("%020d.json", frame.Step)), b); err != nil {
		return err
	}
	digest := sha256.Sum256(b)
	head.Generation++
	head.CommittedSteps++
	head.Metadata.Shape[0]++
	head.LastRecordSHA256 = hex.EncodeToString(digest[:])
	return putJSON(filepath.Join(dir, "commit.json"), head)
}

// PublishProjection exports ordinary Zarr metadata from one committed prefix.
func PublishProjection(dir string) error {
	head, err := ReadCommit(dir)
	if err != nil {
		return err
	}
	times := make([]float64, 0, head.CommittedSteps)
	previous := ""
	for step := 0; step < head.CommittedSteps; step++ {
		b, err := fsutil.Read(filepath.Join(dir, ".frames", fmt.Sprintf("%020d.json", step)))
		if err != nil {
			return err
		}
		var frame FrameRecord
		if err := json.Unmarshal(b, &frame); err != nil || frame.Step != step || frame.PreviousSHA256 != previous || math.IsNaN(frame.Time) || math.IsInf(frame.Time, 0) {
			return fmt.Errorf("zarr: invalid committed frame %d", step)
		}
		frameHead := head
		frameHead.CommittedSteps = step
		if err := validateRecords(frameHead, frame.Chunks); err != nil {
			return err
		}
		digest := sha256.Sum256(b)
		previous = hex.EncodeToString(digest[:])
		times = append(times, frame.Time)
	}
	if previous != head.LastRecordSHA256 {
		return errors.New("zarr: frame record chain mismatch")
	}
	if err := putJSON(filepath.Join(dir, ".zattrs"), map[string]any{"t": times, "name": head.Name, "unit": head.Unit}); err != nil {
		return err
	}
	return putJSON(filepath.Join(dir, ".zarray"), head.Metadata)
}
