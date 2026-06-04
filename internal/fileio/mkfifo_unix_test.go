//go:build unix

package fileio

import "syscall"

func mkfifo(path string) error {
	return syscall.Mkfifo(path, 0o600)
}
