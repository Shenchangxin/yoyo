// Copyright 2026 Hypit.AI
// Translated from @hypit/atomic-file (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

//go:build windows

package atomicfile

import (
	"os"

	"golang.org/x/sys/windows"
)

// openReader matches Node's fs.open: readers share delete, so a POSIX rename can publish over them.
func openReader(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), path), nil
}
