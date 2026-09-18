//go:build darwin

package vault

import (
	"os/exec"
	"strings"
)

func keychainAvailable() bool { return true }

func keychainSet(name, value string) error {
	cmd := exec.Command("security", "add-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name, "-w", value, "-U")
	return cmd.Run()
}

func keychainGet(name string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name, "-w").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func keychainDelete(name string) error {
	return exec.Command("security", "delete-generic-password", "-a", "yoyo", "-s", "yoyo.vault."+name).Run()
}
