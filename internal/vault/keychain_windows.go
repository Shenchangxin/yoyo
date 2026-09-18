//go:build windows

package vault

import (
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric  = 1
	credPersistLocal = 3
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredFree   = advapi32.NewProc("CredFree")
	procCredDelete = advapi32.NewProc("CredDeleteW")
)

type nativeCred struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

func keychainAvailable() bool { return true }

func targetName(name string) string { return "Yoyo/" + name }

func keychainSet(name, value string) error {
	t, err := syscall.UTF16PtrFromString(targetName(name))
	if err != nil {
		return err
	}
	blob := []byte(value)
	var blobPtr *byte
	if len(blob) > 0 {
		blobPtr = &blob[0]
	}
	cred := nativeCred{
		Type:               credTypeGeneric,
		TargetName:         t,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     blobPtr,
		Persist:            credPersistLocal,
	}
	r, _, e := procCredWrite.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if r == 0 {
		if e != syscall.Errno(0) {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}

func keychainGet(name string) (string, error) {
	t, err := syscall.UTF16PtrFromString(targetName(name))
	if err != nil {
		return "", err
	}
	var cred *nativeCred
	r, _, e := procCredRead.Call(uintptr(unsafe.Pointer(t)), uintptr(credTypeGeneric), 0, uintptr(unsafe.Pointer(&cred)))
	if r == 0 {
		if e != syscall.Errno(0) {
			return "", e
		}
		return "", syscall.ENOENT
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))
	if cred == nil || cred.CredentialBlob == nil || cred.CredentialBlobSize == 0 {
		return "", syscall.ENOENT
	}
	b := unsafe.Slice(cred.CredentialBlob, cred.CredentialBlobSize)
	return string(append([]byte(nil), b...)), nil
}

func keychainDelete(name string) error {
	t, err := syscall.UTF16PtrFromString(targetName(name))
	if err != nil {
		return err
	}
	r, _, e := procCredDelete.Call(uintptr(unsafe.Pointer(t)), uintptr(credTypeGeneric), 0)
	if r == 0 {
		if e != syscall.Errno(0) {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}
