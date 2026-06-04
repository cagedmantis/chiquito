package fileio

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadRegularFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	want := []byte("hello\nworld\n")
	if err := os.WriteFile(p, want, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("Read = %q, want %q", got, want)
	}
}

func TestReadRejectsDirectory(t *testing.T) {
	if _, err := Read(t.TempDir()); err == nil {
		t.Error("expected error reading a directory")
	}
}

func TestReadRejectsFIFO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no FIFOs on Windows")
	}
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := mkfifo(fifo); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	if _, err := Read(fifo); err == nil {
		t.Error("expected error reading a FIFO")
	}
}

func TestReadMissingFile(t *testing.T) {
	if _, err := Read(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteAtomicCreatesFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new.txt")
	data := []byte("content")
	if err := WriteAtomic(p, data); err != nil {
		t.Fatalf("WriteAtomic error: %v", err)
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != "content" {
		t.Fatalf("readback = %q, %v", got, err)
	}
	// New files default to 0600.
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(p)
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("new file perm = %o, want 600", perm)
		}
	}
	// No temp files left behind.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file in dir, found %d", len(entries))
	}
}

func TestWriteAtomicPreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits differ on Windows")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(p, []byte("new contents")); err != nil {
		t.Fatalf("WriteAtomic error: %v", err)
	}
	info, _ := os.Stat(p)
	if perm := info.Mode().Perm(); perm != 0o640 {
		t.Errorf("perm = %o, want 640 (preserved)", perm)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "new contents" {
		t.Errorf("contents = %q", got)
	}
}

func TestWriteAtomicThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privileged on Windows")
	}
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(realPath, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteAtomic(link, []byte("updated")); err != nil {
		t.Fatalf("WriteAtomic error: %v", err)
	}

	// The link must still be a symlink pointing at real.txt...
	li, err := os.Lstat(link)
	if err != nil || li.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link is no longer a symlink: mode=%v err=%v", li.Mode(), err)
	}
	// ...and the target must hold the new contents.
	got, _ := os.ReadFile(realPath)
	if string(got) != "updated" {
		t.Errorf("target contents = %q, want updated", got)
	}
}

func TestWriteAtomicRejectsEmptyName(t *testing.T) {
	if err := WriteAtomic("", []byte("x")); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestWriteAtomicRefusesDirectory(t *testing.T) {
	if err := WriteAtomic(t.TempDir(), []byte("x")); err == nil {
		t.Error("expected error writing over a directory")
	}
}

func TestWriteAtomicRefusesFIFO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no FIFOs on Windows")
	}
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := mkfifo(fifo); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	if err := WriteAtomic(fifo, []byte("x")); err == nil {
		t.Error("expected error writing over a FIFO")
	}
}

func TestWriteAtomicRefusesSymlinkToDevice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks/devices differ on Windows")
	}
	dir := t.TempDir()
	link := filepath.Join(dir, "dev")
	if err := os.Symlink("/dev/null", link); err != nil {
		t.Skipf("cannot symlink to /dev/null: %v", err)
	}
	if err := WriteAtomic(link, []byte("x")); err == nil {
		t.Error("expected error writing through a symlink to a device")
	}
}

func TestReadWriteBinary(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bin")
	data := []byte{0x00, 0x01, 0xff, '\n', 0x00, 'A'}
	if err := WriteAtomic(p, data); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("binary round trip = %v, want %v", got, data)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rt.txt")
	data := []byte("héllo, 世界\n🌍\n")
	if err := WriteAtomic(p, data); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("round trip mismatch: %q != %q", got, data)
	}
}
