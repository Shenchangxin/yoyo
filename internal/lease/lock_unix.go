//go:build !windows

package lease

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLock 不等待地拿 f 的排他 flock。别人拿着时返回 ErrHeld。
func tryLock(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return ErrHeld
	}
	return err
}

// lockWait 等到拿到 f 的排他 flock。被信号打断时重试。
func lockWait(f *os.File) error {
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func unlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
