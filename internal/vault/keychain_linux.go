//go:build linux

// Linux 密钥环：PATH 里有 secret-tool 时，按 service=yoyo、account=密钥名读写。没有该命令则钥匙串不可用。
// 不解析 vault.json，也不读环境变量。vault.go 在打开仓库、Set 和 GetStored 时调用这里。

package vault

import (
	"os/exec"
	"strings"
)

// keychainAvailable 在 PATH 中能找到 secret-tool 时返回 true。
func keychainAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

// keychainSet 把 value 从标准输入交给 secret-tool store，标签是 Yoyo 加空格加 name。命令失败时返回该错误。
func keychainSet(name, value string) error {
	cmd := exec.Command("secret-tool", "store", "--label=Yoyo "+name, "service", "yoyo", "account", name)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

// keychainGet 执行 secret-tool lookup，只去掉输出末尾的回车和换行。命令失败时返回空串和该错误。
func keychainGet(name string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup", "service", "yoyo", "account", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// keychainDelete 执行 secret-tool clear，service 和 account 与写入时相同。
func keychainDelete(name string) error {
	return exec.Command("secret-tool", "clear", "service", "yoyo", "account", name).Run()
}
