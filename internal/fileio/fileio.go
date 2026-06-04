// Package fileio provides chiquito's secure file access primitives: a guarded
// reader and an atomic, durability- and permission-preserving writer.
package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxFileSize caps how many bytes Read will load, guarding against accidentally
// opening an enormous (or maliciously crafted) file.
const MaxFileSize = 1 << 30 // 1 GiB

// Read loads the named file. It opens without blocking (so a FIFO or device
// cannot stall the editor at open time), stats the open descriptor — avoiding a
// stat/open TOCTOU window — refuses anything that is not a regular file, and
// reads at most MaxFileSize bytes.
func Read(name string) ([]byte, error) {
	f, err := openForRead(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("fileio: refusing to read non-regular file %q (%s)", name, info.Mode().Type())
	}
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("fileio: %q is %d bytes, exceeds limit of %d", name, info.Size(), MaxFileSize)
	}

	// LimitReader guards against the file growing between stat and read.
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileSize {
		return nil, fmt.Errorf("fileio: %q exceeds limit of %d bytes", name, MaxFileSize)
	}
	return data, nil
}

// WriteAtomic writes data to name atomically. It writes to a 0600 temporary
// file in the destination directory, fsyncs the contents, then renames it over
// the target so that a crash or concurrent reader never observes a partially
// written file; the directory is fsynced so the rename itself is durable.
//
// If name is a symlink, the link is resolved and its target is replaced,
// keeping the symlink intact (the conventional editor behaviour). Existing file
// permissions are preserved; newly created files are mode 0600.
func WriteAtomic(name string, data []byte) (err error) {
	// Resolve symlinks so we replace the real file rather than clobbering the
	// link. EvalSymlinks fails for a not-yet-existing file; in that case we
	// keep the original name.
	target := name
	if resolved, lerr := filepath.EvalSymlinks(name); lerr == nil {
		target = resolved
	}
	dir := filepath.Dir(target)

	perm := os.FileMode(0o600)
	if info, serr := os.Stat(target); serr == nil {
		perm = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".chiquito-*.tmp")
	if err != nil {
		return fmt.Errorf("fileio: create temp: %w", err)
	}
	tmpName := tmp.Name()
	// On any failure after creation, clean up the temp file.
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err = tmp.Chmod(perm); err != nil {
		return fmt.Errorf("fileio: chmod temp: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("fileio: write temp: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("fileio: fsync temp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("fileio: close temp: %w", err)
	}
	if err = os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("fileio: rename: %w", err)
	}

	// Best-effort directory fsync to make the rename durable. Failure here does
	// not invalidate the write, so it is not fatal.
	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
