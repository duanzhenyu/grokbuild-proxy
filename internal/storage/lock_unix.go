//go:build !windows

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockFile(file *os.File, nonblocking bool) error {
	flags := unix.LOCK_EX
	if nonblocking {
		flags |= unix.LOCK_NB
	}
	return unix.Flock(int(file.Fd()), flags)
}

func unlockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func isLockBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}
