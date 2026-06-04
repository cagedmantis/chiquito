//go:build !unix

package fileio

import "errors"

func mkfifo(string) error {
	return errors.New("mkfifo not supported on this platform")
}
