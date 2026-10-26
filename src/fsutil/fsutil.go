// Package fsutil provides utility functions for file system operations.
package fsutil

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var wd = "" // Working directory set by SetWD

const (
	DirPerm  = 0o755            // Permissions for new directories
	FilePerm = 0o644            // Permissions for new files
	BUFSIZE  = 16 * 1024 * 1024 // Buffer size for buffered writer
)

// SetWD sets the working directory, prefixed to all relative paths.
func SetWD(dir string) {
	if dir != "" && !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	wd = dir
}

// Mkdir creates a directory at the specified path.
func Mkdir(p string) error {
	p = addWorkDir(p)
	return os.Mkdir(p, DirPerm)
}

// AcquireOutputLock claims the common solver lock using one atomic mkdir.
// A lock left by a killed writer must be investigated and removed explicitly.
func AcquireOutputLock(output string) (string, error) {
	lock := strings.TrimRight(output, "/") + ".zarr-owner"
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	host, _ := os.Hostname()
	owner, err := json.Marshal(map[string]any{
		"token": hex.EncodeToString(token[:]), "pid": os.Getpid(), "host": host,
		"started_utc": time.Now().UTC().Format(time.RFC3339Nano), "output": output,
	})
	if err != nil {
		return "", err
	}
	if err := Mkdir(lock); err != nil {
		return "", fmt.Errorf("cannot claim output lock %s: %w", lock, err)
	}
	if err := Put(filepath.Join(lock, "owner.json"), owner); err != nil {
		_ = Remove(lock)
		return "", err
	}
	return lock, nil
}

// ReleaseOutputLock refuses to remove a lock whose owner record changed.
func ReleaseOutputLock(lock string, owner []byte) error {
	current, err := Read(filepath.Join(lock, "owner.json"))
	if err != nil {
		return err
	}
	if !bytes.Equal(current, owner) {
		return fmt.Errorf("output lock ownership changed: %s", lock)
	}
	return Remove(lock)
}

// Touch creates an empty file at the specified path.
func Touch(p string) error {
	p = addWorkDir(p)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, FilePerm)
	if err == nil {
		errClose := f.Close()
		if errClose != nil {
			return errClose
		}
	}
	return err
}

// Remove deletes the file or directory at the path, including any children.
func Remove(p string) error {
	p = addWorkDir(p)
	return os.RemoveAll(p)
}

// Read reads the entire file and returns its contents.
func Read(p string) ([]byte, error) {
	p = addWorkDir(p)
	return os.ReadFile(p)
}

// ReadDir lists names without suppressing permission and storage errors.
func ReadDir(p string) ([]string, error) {
	entries, err := os.ReadDir(addWorkDir(p))
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

// Exists checks if the file or directory at the path exists.
func Exists(p string) bool {
	p = addWorkDir(p)
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}

// IsDir checks if the path is a directory.
func IsDir(p string) bool {
	p = addWorkDir(p)
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// Append appends data to the file.
func Append(p string, data []byte) (retErr error) {
	p = addWorkDir(p)

	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, FilePerm) // add os.O_CREATE if desired
	if err != nil {
		return err
	}

	// Ensure we surface any close error as well.
	defer func() {
		if cerr := f.Close(); cerr != nil {
			if retErr == nil {
				retErr = cerr
			} else {
				retErr = fmt.Errorf("%w; close error: %v", retErr, cerr)
			}
		}
	}()

	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// Put atomically replaces a file after persisting its contents. It does not
// truncate the previous version when the new write fails or the process dies.
func Put(p string, data []byte) error {
	p = addWorkDir(p)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".amumax-write-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	mode := os.FileMode(FilePerm)
	if info, statErr := os.Stat(p); statErr == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(statErr) {
		return errors.Join(statErr, f.Close())
	}
	if err := f.Chmod(mode); err != nil {
		return errors.Join(err, f.Close())
	}
	n, err := f.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return errors.Join(err, f.Close())
	}
	if err := errors.Join(f.Sync(), f.Close()); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	return errors.Join(d.Sync(), d.Close())
}

// Create opens a file for writing, truncating it if it exists.
func Create(p string) (WriteCloseFlusher, error) {
	p = addWorkDir(p)
	err := os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, FilePerm)
	if err != nil {
		return nil, err
	}
	writer := &bufWriter{
		buf:  bufio.NewWriterSize(f, BUFSIZE),
		file: f,
	}
	return writer, nil
}

// Open opens a file for reading.
func Open(p string) (io.ReadCloser, error) {
	p = addWorkDir(p)
	return os.Open(p)
}

// WriteCloseFlusher represents a writer that can be flushed and closed.
type WriteCloseFlusher interface {
	io.WriteCloser
	Flush() error
}

type bufWriter struct {
	buf  *bufio.Writer
	file *os.File
}

func (w *bufWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *bufWriter) Flush() error {
	return w.buf.Flush()
}

func (w *bufWriter) Close() error {
	err := w.Flush()
	if err != nil {
		errClose := w.file.Close()
		if errClose != nil {
			return errClose
		}
		return err
	}
	return w.file.Close()
}

// Helper function to add the working directory to the path if it's relative.
func addWorkDir(p string) string {
	if !filepath.IsAbs(p) {
		return filepath.Join(wd, p)
	}
	return p
}
