//go:build !windows && !darwin && !linux

// 既不是 Windows、darwin 也不是 Linux 时，没有操作系统钥匙串。写入和读取返回 errNoKeychain，删除返回 nil。
// 不解析 vault.json。vault.go 在这些平台上仍会调用这里，密钥只留在文件里。

package vault

// keychainAvailable 在这个构建目标上返回 false。
func keychainAvailable() bool { return false }

// keychainSet 不写入钥匙串，返回 errNoKeychain。
// name 本函数不读取。
// value 本函数不读取。
func keychainSet(name, value string) error { return errNoKeychain }

// keychainGet 不读取钥匙串，返回空串和 errNoKeychain。
// name 本函数不读取。
func keychainGet(name string) (string, error) { return "", errNoKeychain }

// keychainDelete 没有钥匙串条目可删，返回 nil。
// name 本函数不读取。
func keychainDelete(name string) error { return nil }

// keychainErr 是本文件使用的错误字符串。
type keychainErr string

// Error 返回该错误字符串。
func (e keychainErr) Error() string { return string(e) }

// errNoKeychain 表示当前平台没有操作系统钥匙串，文本是 vault: no os keychain。
const errNoKeychain keychainErr = "vault: no os keychain"
