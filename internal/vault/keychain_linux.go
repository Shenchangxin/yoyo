//go:build linux

package vault

import (
	"os/exec"
	"strings"
)

func keychainAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func keychainSet(name, value string) error {
	cmd := exec.Command("secret-tool", "store", "--label=Yoyo "+name, "service", "yoyo", "account", name)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func keychainGet(name string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup", "service", "yoyo", "account", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func keychainDelete(name string) error {
	return exec.Command("secret-tool", "clear", "service", "yoyo", "account", name).Run()
}
