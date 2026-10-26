//go:build windows

// Windows 凭据管理器：用 advapi32 的 CredWriteW、CredReadW、CredDeleteW 保存密钥，目标名是 Yoyo/ 加名字。
// 不解析 vault.json，也不读环境变量。vault.go 在打开仓库、Set 和 GetStored 时调用这里。

package vault

import (
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric  = 1 // CRED_TYPE_GENERIC
	credPersistLocal = 2 // CRED_PERSIST_LOCAL_MACHINE；3 是 CRED_PERSIST_ENTERPRISE
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredFree   = advapi32.NewProc("CredFree")
	procCredDelete = advapi32.NewProc("CredDeleteW")
)

// nativeCred 的字段顺序与 Windows CREDENTIALW 一致，指针直接传给 CredWriteW 和 CredReadW。
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

// keychainAvailable 在 Windows 上返回 true，打开仓库时会走凭据管理器。
func keychainAvailable() bool { return true }

// targetName 拼出凭据目标名，形式是 Yoyo/ 加 name。
func targetName(name string) string { return "Yoyo/" + name }

// keychainSet 用 CredWriteW 写入通用凭据。目标名是 Yoyo/ 加 name，持久化取值是 credPersistLocal。
// blob 为空时指针是 nil。API 返回 0 表示失败：系统 errno 非 0 则返回它，否则返回 EINVAL。
// value 作为凭据 blob；长度为 0 时指针是 nil。
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

// keychainGet 用 CredReadW 读取同一目标名的通用凭据，原样拷贝 blob，不去空白。
// 调用失败且 errno 为 0，或 blob 为空时，返回 ENOENT。
// name 经 targetName 拼成凭据目标名，交给 CredReadW。
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
	// CredFree 会释放这块缓冲，必须先拷贝出来。
	return string(append([]byte(nil), b...)), nil
}

// keychainDelete 用 CredDeleteW 删除同一目标名的通用凭据。失败时的 errno 处理和 keychainSet 相同。
// name 经 targetName 拼成凭据目标名，交给 CredDeleteW。
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
