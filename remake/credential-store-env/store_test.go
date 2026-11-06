// Copyright 2026 Hypit.AI
// Translated from @hypit/credential-store-env/test/store.test.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package credentialstoreenv

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestEnvironmentCredentialsResolveOnlyTheExplicitlyRequestedKey(t *testing.T) {
	store := NewEnvironmentCredentialStore(map[string]string{
		"PROVIDER_KEY":     "secret-value",
		"UNRELATED_SECRET": "must-not-be-enumerated",
	})
	got := store.Resolve(credentialRef("env", "PROVIDER_KEY"))
	if got == nil || *got != (CredentialValue{Secret: "secret-value"}) {
		t.Fatalf("%+v", got)
	}
	if store.Resolve(credentialRef("env", "MISSING")) != nil {
		t.Fatal("missing key was returned")
	}
	if store.Resolve(credentialRef("os", "PROVIDER_KEY")) != nil {
		t.Fatal("another store was resolved")
	}
}

func TestResolveUsesTheRawKeyAndRejectsAnEmptyValue(t *testing.T) {
	store := NewEnvironmentCredentialStore(map[string]string{
		"PROVIDER_KEY":   "",
		"SPACE":          " ",
		" PROVIDER_KEY ": "secret-value",
		"provider_key":   "other-case",
	})
	if store.Resolve(credentialRef("env", "PROVIDER_KEY")) != nil {
		t.Fatal("empty secret was returned")
	}
	got := store.Resolve(credentialRef("env", "SPACE"))
	if got == nil || got.Secret != " " {
		t.Fatalf("%+v", got)
	}
	got = store.Resolve(credentialRef("env", " PROVIDER_KEY "))
	if got == nil || got.Secret != "secret-value" {
		t.Fatalf("%+v", got)
	}
	got = store.Resolve(credentialRef("env", "provider_key"))
	if got == nil || got.Secret != "other-case" {
		t.Fatalf("%+v", got)
	}
	if store.Resolve(CredentialRef{Store: " env", Key: "PROVIDER_KEY"}) != nil {
		t.Fatal("store name was trimmed before comparison")
	}
}

func TestResolvePanicsWhenTheCredentialRefIsBlank(t *testing.T) {
	store := NewEnvironmentCredentialStore(map[string]string{
		"\uFEFF": "secret-value",
		"\u0085": "nel-value",
		"K":      "v",
	})
	if message := recoverMessage(t, func() {
		store.Resolve(CredentialRef{Store: "", Key: ""})
	}); message != "CredentialRef store is empty" {
		t.Fatal(message)
	}
	if message := recoverMessage(t, func() {
		store.Resolve(CredentialRef{Store: " \t", Key: "K"})
	}); message != "CredentialRef store is empty" {
		t.Fatal(message)
	}
	if message := recoverMessage(t, func() {
		store.Resolve(CredentialRef{Store: "env", Key: "\n"})
	}); message != "CredentialRef key is empty" {
		t.Fatal(message)
	}
	if message := recoverMessage(t, func() {
		store.Resolve(CredentialRef{Store: "env", Key: "\uFEFF"})
	}); message != "CredentialRef key is empty" {
		t.Fatal(message)
	}
	if message := recoverMessage(t, func() {
		store.Resolve(CredentialRef{Store: "\u1680", Key: "K"})
	}); message != "CredentialRef store is empty" {
		t.Fatal(message)
	}
	got := store.Resolve(CredentialRef{Store: "env", Key: "\u0085"})
	if got == nil || got.Secret != "nel-value" {
		t.Fatalf("%+v", got)
	}
}

func TestInjectedEnvironmentIsLiveAndIgnoresTheProcess(t *testing.T) {
	const key = "HYPIT_CREDENTIAL_STORE_ENV_TEST_KEY"
	t.Setenv(key, "from-process")
	environment := map[string]string{"OTHER": "kept"}
	store := NewEnvironmentCredentialStore(environment)
	if store.Resolve(credentialRef("env", key)) != nil {
		t.Fatal("injected store read the process environment")
	}
	environment["OTHER"] = "changed"
	got := store.Resolve(credentialRef("env", "OTHER"))
	if got == nil || got.Secret != "changed" {
		t.Fatalf("%+v", got)
	}
	empty := NewEnvironmentCredentialStore(map[string]string{})
	if empty.Resolve(credentialRef("env", key)) != nil {
		t.Fatal("empty environment read the process")
	}
}

func TestDefaultStoreReadsOnlyTheRequestedProcessVariable(t *testing.T) {
	const key = "HYPIT_CREDENTIAL_STORE_ENV_TEST_KEY"
	const other = "HYPIT_CREDENTIAL_STORE_ENV_TEST_OTHER"
	t.Setenv(key, "secret-value")
	t.Setenv(other, "must-not-be-enumerated")
	store := NewEnvironmentCredentialStore(nil)
	got := store.Resolve(credentialRef("env", key))
	if got == nil || got.Secret != "secret-value" {
		t.Fatalf("%+v", got)
	}
	if store.Resolve(credentialRef("env", "HYPIT_CREDENTIAL_STORE_ENV_TEST_MISSING")) != nil {
		t.Fatal("missing process variable was returned")
	}
	if store.Resolve(credentialRef("os", key)) != nil {
		t.Fatal("another store was resolved from the process")
	}
	if err := os.Setenv(key, "later"); err != nil {
		t.Fatal(err)
	}
	got = store.Resolve(credentialRef("env", key))
	if got == nil || got.Secret != "later" {
		t.Fatalf("%+v", got)
	}
	if err := os.Setenv(key, ""); err != nil {
		t.Fatal(err)
	}
	if store.Resolve(credentialRef("env", key)) != nil {
		t.Fatal("empty process value was returned")
	}
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	if store.Resolve(credentialRef("env", key)) != nil {
		t.Fatal("unset process variable was returned")
	}
	folded := store.Resolve(credentialRef("env", strings.ToLower(other)))
	if runtime.GOOS == "windows" {
		if folded == nil || folded.Secret != "must-not-be-enumerated" {
			t.Fatalf("%+v", folded)
		}
		return
	}
	if folded != nil {
		t.Fatalf("%+v", folded)
	}
}

func credentialRef(store, key string) CredentialRef {
	value := CredentialRef{Store: store, Key: key}
	verifyCredentialRef(value)
	return value
}

func recoverMessage(t *testing.T, fn func()) string {
	t.Helper()
	var message string
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(error)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			message = err.Error()
		}()
		fn()
	}()
	if message == "" {
		t.Fatal("expected a thrown error")
	}
	return message
}
