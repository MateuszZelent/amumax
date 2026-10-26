package fsutil

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAcquireOutputLockAllowsOneOwnerAndSupportsReacquisition(t *testing.T) {
	output := filepath.Join(t.TempDir(), "simulation.zarr")
	lockPath := output + ".zarr-owner"

	type result struct {
		path string
		err  error
	}
	results := make(chan result, 32)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for i := 0; i < cap(results); i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			path, err := AcquireOutputLock(output + "///")
			results <- result{path: path, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	var successful []string
	for result := range results {
		if result.err == nil {
			successful = append(successful, result.path)
		}
	}
	if len(successful) != 1 {
		t.Fatalf("successful lock acquisitions = %d, want exactly 1", len(successful))
	}
	if successful[0] != lockPath {
		t.Fatalf("lock path = %q, want %q", successful[0], lockPath)
	}
	if info, err := os.Stat(lockPath); err != nil {
		t.Fatalf("stat owner lock: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("owner lock is %s, want directory", info.Mode().Type())
	}

	if _, err := AcquireOutputLock(output); err == nil {
		t.Fatal("second acquisition succeeded while the first owner was active")
	}
	if err := Remove(lockPath); err != nil {
		t.Fatalf("release owner lock: %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("owner lock still exists after release: %v", err)
	}

	reacquired, err := AcquireOutputLock(output)
	if err != nil {
		t.Fatalf("reacquire after release: %v", err)
	}
	if reacquired != lockPath {
		t.Fatalf("reacquired path = %q, want %q", reacquired, lockPath)
	}
	if err := Remove(reacquired); err != nil {
		t.Fatalf("cleanup reacquired lock: %v", err)
	}
}

func TestReleaseOutputLockRefusesChangedOwner(t *testing.T) {
	output := filepath.Join(t.TempDir(), "run.zarr")
	lock, err := AcquireOutputLock(output)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := Read(filepath.Join(lock, "owner.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lock, "owner.json"), []byte(`{"token":"other"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseOutputLock(lock, owner); err == nil {
		t.Fatal("released a different owner's lock")
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("new owner's lock disappeared: %v", err)
	}
}
