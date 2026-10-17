package ir

import "fmt"

func wrapf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
