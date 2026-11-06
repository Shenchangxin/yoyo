// Copyright 2026 Hypit.AI
// Translated from Node fs.realpath, used by @hypit/workspace.
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build windows

package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// realPath matches Node fs.realpath. The \\?\ prefix keeps a directory whose
// name starts with ".." literal, and GetFinalPathNameByHandle resolves junctions.
func realPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	prefixed := absolute
	if strings.HasPrefix(absolute, `\\`) {
		prefixed = `\\?\UNC\` + strings.TrimPrefix(absolute, `\\`)
	} else {
		prefixed = `\\?\` + absolute
	}
	name, err := windows.UTF16PtrFromString(prefixed)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(
		name,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return "", &os.PathError{Op: "realpath", Path: path, Err: err}
	}
	defer windows.CloseHandle(handle)
	buf := make([]uint16, 32768)
	n, err := windows.GetFinalPathNameByHandle(handle, &buf[0], uint32(len(buf)), 0)
	if err != nil {
		return "", &os.PathError{Op: "realpath", Path: path, Err: err}
	}
	if n == 0 || int(n) > len(buf) {
		return "", &os.PathError{Op: "realpath", Path: path, Err: windows.ERROR_INSUFFICIENT_BUFFER}
	}
	return stripNamespace(windows.UTF16ToString(buf[:n])), nil
}

func stripNamespace(path string) string {
	switch {
	case strings.HasPrefix(path, `\\?\UNC\`):
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	case strings.HasPrefix(path, `\\?\`):
		return strings.TrimPrefix(path, `\\?\`)
	default:
		return path
	}
}
