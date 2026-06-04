//go:build unix

package fileio

import (
	"os"
	"syscall"
)

// openForRead opens name for reading with O_NONBLOCK so that opening a FIFO,
// socket, or device returns immediately instead of blocking until a peer
// appears. For a regular file O_NONBLOCK has no effect on subsequent reads, and
// the caller rejects anything that is not a regular file. O_NOFOLLOW is
// deliberately not set: chiquito intentionally follows symlinks to regular
// files (the type check on the resolved descriptor remains the safety gate).
func openForRead(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
