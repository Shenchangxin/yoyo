//go:build !windows && !darwin && !linux

package vault

func keychainAvailable() bool { return false }

func keychainSet(name, value string) error { return errNoKeychain }

func keychainGet(name string) (string, error) { return "", errNoKeychain }

func keychainDelete(name string) error { return nil }

type keychainErr string

func (e keychainErr) Error() string { return string(e) }

const errNoKeychain keychainErr = "vault: no os keychain"
