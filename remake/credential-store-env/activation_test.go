// Copyright 2026 Hypit.AI
// Translated from @hypit/credential-store-env/src/activation.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package credentialstoreenv

import "testing"

func TestPackageOpensTheProcessStoreAndRejectsConfigKeys(t *testing.T) {
	if HypitPackage.Format != "hypit.package@1" || len(HypitPackage.Facets) != 1 {
		t.Fatalf("%+v", HypitPackage)
	}
	item := HypitPackage.Facets[0]
	if item.ABI != "hypit.runtime-local-credential-store-adapter@1" || len(item.Offers) != 1 || item.Offers[0] != "@hypit/credential-store-env" {
		t.Fatalf("%+v", item)
	}
	adapter, ok := item.Implementation.(CredentialStoreAdapter)
	if !ok || adapter.Validate == nil || adapter.Open == nil {
		t.Fatalf("%#v", item.Implementation)
	}
	adapter.Validate(FactoryContext{Config: map[string]any{}})
	if message := recoverMessage(t, func() {
		adapter.Validate(FactoryContext{Config: map[string]any{"path": "x"}})
	}); message != "environment CredentialStore config does not accept path" {
		t.Fatal(message)
	}
	if message := recoverMessage(t, func() {
		adapter.Validate(FactoryContext{Config: map[string]any{"": true}})
	}); message != "environment CredentialStore config does not accept " {
		t.Fatalf("%q", message)
	}
	if message := recoverMessage(t, func() {
		adapter.Validate(FactoryContext{Config: map[string]string{"backend": "file"}})
	}); message != "environment CredentialStore config does not accept backend" {
		t.Fatal(message)
	}
	var nilMap map[string]any
	for _, config := range []any{nil, nilMap, []any{}, []any(nil), []string{"env"}, "env", 1, 1.5, true, false, struct{}{}, map[int]string{1: "a"}} {
		message := recoverMessage(t, func() {
			adapter.Validate(FactoryContext{Config: config})
		})
		if message != "environment CredentialStore config must be an object" {
			t.Fatalf("%#v: %s", config, message)
		}
	}
	runtimeConfigExact(map[string]any{"backend": "file"}, []string{"backend"}, "environment CredentialStore")
	if message := recoverMessage(t, func() {
		runtimeConfigExact(map[string]any{"backend": "file", "path": "x"}, []string{"backend"}, "environment CredentialStore")
	}); message != "environment CredentialStore config does not accept path" {
		t.Fatal(message)
	}

	const key = "HYPIT_CREDENTIAL_STORE_ENV_TEST_KEY"
	t.Setenv(key, "from-open")
	first := adapter.Open(FactoryContext{Config: "ignored"})
	second := adapter.Open(FactoryContext{})
	if first.Value == nil || second.Value == nil || first.Close != nil || second.Close != nil || first.Value == second.Value {
		t.Fatalf("%+v %+v", first, second)
	}
	got := first.Value.Resolve(credentialRef("env", key))
	if got == nil || got.Secret != "from-open" {
		t.Fatalf("%+v", got)
	}
}
