//go:build darwin

// darwin 钥匙串：用 security 读写通用密码，账户是 yoyo，服务名是 yoyo.vault. 加密钥名。
// 不解析 vault.json，也不读环境变量。vault.go 在打开仓库、Set 和 GetStored 时调用这里。

package vault

import (
	"os/exec"
	"strings"
)

// keychainAvailable 在 darwin 上返回 true，不探测 security 是否存在。
func keychainAvailable() bool { return true }

// keychainSet 执行 security add-generic-password。-w 传入明文，-U 表示已存在则更新。命令失败时返回该错误。
func keychainSet(name, value string) error {
	cmd := exec.Command("security", "add-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name, "-w", value, "-U")
	return cmd.Run()
}

// keychainGet 执行 security find-generic-password -w，对标准输出做 TrimSpace。命令失败时返回空串和该错误。
func keychainGet(name string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name, "-w").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// keychainDelete 执行 security delete-generic-password，账户和服务名与写入时相同。
func keychainDelete(name string) error {
	return exec.Command("security", "delete-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name).Run()
}
