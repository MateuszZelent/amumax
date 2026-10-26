package fsutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const crashPayloadSize = 256 << 20

func TestPutReplacesContentsAndPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := Put(path, []byte("old generation")); err != nil {
		t.Fatalf("write initial generation: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod initial generation: %v", err)
	}

	if err := Put(path, []byte("new generation")); err != nil {
		t.Fatalf("replace generation: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replaced generation: %v", err)
	}
	if string(contents) != "new generation" {
		t.Fatalf("contents = %q, want new generation", contents)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced generation: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("permissions = %o, want 600", got)
	}
}

func TestPutCrashLeavesACompleteGeneration(t *testing.T) {
	if os.Getenv("AMUMAX_FSUTIL_PUT_HELPER") == "1" {
		path := os.Getenv("AMUMAX_FSUTIL_PUT_PATH")
		payload := bytes.Repeat([]byte{'N'}, crashPayloadSize)
		if err := Put(path, payload); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")
	old := []byte("old generation")
	if err := Put(path, old); err != nil {
		t.Fatalf("write initial generation: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPutCrashLeavesACompleteGeneration$")
	cmd.Env = append(os.Environ(),
		"AMUMAX_FSUTIL_PUT_HELPER=1",
		"AMUMAX_FSUTIL_PUT_PATH="+path,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start writer helper: %v", err)
	}

	// The temporary-file name is the synchronization point. It proves that the
	// helper has opened its replacement before it is interrupted, without
	// depending on a sleep or on a particular scheduler decision.
	deadline := time.Now().Add(10 * time.Second)
	observedTemp := false
	for time.Now().Before(deadline) {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("list output directory: %v", readErr)
		}
		for _, entry := range entries {
			if len(entry.Name()) >= len(".amumax-write-") &&
				entry.Name()[:len(".amumax-write-")] == ".amumax-write-" {
				observedTemp = true
				break
			}
		}
		if observedTemp {
			break
		}
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			// The helper may have completed before the next directory poll.
			break
		}
		time.Sleep(time.Millisecond)
	}

	if !observedTemp {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("did not observe the temporary replacement before the helper exited")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("interrupt writer helper: %v", err)
	}
	_ = cmd.Wait()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generation after interruption: %v", err)
	}
	if bytes.Equal(contents, old) {
		return
	}
	if len(contents) != crashPayloadSize || !bytes.Equal(contents[:1], []byte{'N'}) || !bytes.Equal(contents[len(contents)-1:], []byte{'N'}) {
		t.Fatalf("interruption left a partial generation: length=%d", len(contents))
	}
	for _, value := range contents {
		if value != 'N' {
			t.Fatalf("interruption left mixed generation at length=%d", len(contents))
		}
	}
}
