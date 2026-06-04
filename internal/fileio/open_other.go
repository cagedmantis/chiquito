//go:build !unix

package fileio

import "os"

// openForRead opens name for reading. On non-unix platforms (notably Windows)
// the blocking-FIFO concern does not apply in the same way, so a plain open is
// used; the regular-file check on the descriptor still applies.
func openForRead(name string) (*os.File, error) {
	return os.Open(name)
}
