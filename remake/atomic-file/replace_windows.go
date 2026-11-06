// Copyright 2026 Hypit.AI
// Translated from @hypit/atomic-file/src/replace-file-windows.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

//go:build windows

package atomicfile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	replaceIfExists = 0x1
	posixSemantics  = 0x2
)

// windowsReplaceError is the Error thrown by the Windows replacement path.
type windowsReplaceError struct {
	Win32Code uint32
	Syscall   string
	Path      string
	Dest      string
}

func (e *windowsReplaceError) Error() string {
	return fmt.Sprintf("Windows file replacement failed (%d): '%s' -> '%s'", e.Win32Code, e.Path, e.Dest)
}

// Code is WIN32_<n>, matching the Node error's code field.
func (e *windowsReplaceError) Code() string {
	return fmt.Sprintf("WIN32_%d", e.Win32Code)
}

func replaceFile(from, to string) error {
	return replaceWindowsFile(from, to)
}

// replaceWindowsFile uses FileRenameInfoEx so an open reader keeps the bytes it opened.
// Node's Windows rename uses MoveFileExW, whose replacement semantics differ here.
func replaceWindowsFile(from, to string) error {
	info, err := renameInformation(to)
	if err != nil {
		return err
	}
	source, err := namespaced(from)
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		name,
		windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return windowsFailure("CreateFileW", from, to, err)
	}
	defer windows.CloseHandle(handle)
	err = windows.SetFileInformationByHandle(handle, windows.FileRenameInfoEx, &info[0], uint32(len(info)))
	if err != nil {
		return windowsFailure("SetFileInformationByHandle", from, to, err)
	}
	return nil
}

func windowsFailure(syscall, from, to string, cause error) error {
	return &windowsReplaceError{
		Win32Code: win32Code(cause),
		Syscall:   syscall,
		Path:      from,
		Dest:      to,
	}
}

func win32Code(err error) uint32 {
	var errno windows.Errno
	if errors.As(err, &errno) {
		return uint32(errno)
	}
	if last := windows.GetLastError(); errors.As(last, &errno) {
		return uint32(errno)
	}
	return 0
}

func renameInformation(to string) ([]byte, error) {
	target, err := namespaced(to)
	if err != nil {
		return nil, err
	}
	units := utf16.Encode([]rune(target))
	nameBytes := len(units) * 2
	nameAt := 20
	lengthAt := 16
	if unsafe.Sizeof(uintptr(0)) == 4 {
		nameAt = 12
		lengthAt = 8
	}
	// FileNameLength excludes the extra NUL wchar Win32 still requires.
	buf := make([]byte, nameAt+nameBytes+2)
	binary.LittleEndian.PutUint32(buf[0:], replaceIfExists|posixSemantics)
	binary.LittleEndian.PutUint32(buf[lengthAt:], uint32(nameBytes))
	for i, unit := range units {
		binary.LittleEndian.PutUint16(buf[nameAt+i*2:], unit)
	}
	return buf, nil
}

func namespaced(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if len(abs) >= 4 && abs[:4] == `\\?\` {
		return abs, nil
	}
	if len(abs) >= 2 && abs[0] == '\\' && abs[1] == '\\' {
		rest := abs[2:]
		if len(rest) > 0 && (rest[0] == '?' || rest[0] == '.') {
			return abs, nil
		}
		return `\\?\UNC\` + rest, nil
	}
	return `\\?\` + abs, nil
}
