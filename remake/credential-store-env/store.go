// Copyright 2026 Hypit.AI
// Translated from @hypit/credential-store-env/src/index.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package credentialstoreenv is a read-only CredentialStore backed by explicitly named environment variables.
// It resolves only the requested key and never snapshots or enumerates the environment.
package credentialstoreenv

import (
	"errors"
	"os"
	"strings"
	"unicode"
)

// CredentialRef is one credential lookup.
// Store is the CredentialStore implementation name, for example env, os, or aws-default.
// Key is the store-local lookup key. It is configuration, never the secret value.
type CredentialRef struct {
	Store string
	Key   string
}

// CredentialValue is the secret revealed to the selected Endpoint invocation.
// A nil ExpiresAt means the property is absent. This store never sets it.
type CredentialValue struct {
	Secret    string
	ExpiresAt *float64
}

// EnvironmentCredentialStore resolves only explicitly requested environment variables and never snapshots or enumerates env.
type EnvironmentCredentialStore struct {
	environment map[string]string
}

// NewEnvironmentCredentialStore keeps environment by reference.
// A nil environment reads the process environment on each Resolve. That is the omitted constructor argument.
// A non-nil map, including an empty map, is the only source consulted.
func NewEnvironmentCredentialStore(environment map[string]string) *EnvironmentCredentialStore {
	return &EnvironmentCredentialStore{environment: environment}
}

// Resolve returns the secret for an env reference.
// Another store, a missing variable, or an empty value returns nil.
// An empty store name or key panics.
func (s *EnvironmentCredentialStore) Resolve(ref CredentialRef) *CredentialValue {
	verifyCredentialRef(ref)
	if ref.Store != "env" {
		return nil
	}
	secret, ok := s.lookup(ref.Key)
	if !ok || len(secret) == 0 {
		return nil
	}
	return &CredentialValue{Secret: secret}
}

func (s *EnvironmentCredentialStore) lookup(key string) (string, bool) {
	if s.environment == nil {
		return os.LookupEnv(key)
	}
	secret, ok := s.environment[key]
	return secret, ok
}

func verifyCredentialRef(ref CredentialRef) {
	assert(len(jsTrim(ref.Store)) > 0, "CredentialRef store is empty")
	assert(len(jsTrim(ref.Key)) > 0, "CredentialRef key is empty")
}

func assert(condition bool, message string) {
	if !condition {
		panic(errors.New(message))
	}
}

// jsTrim removes the characters String.prototype.trim removes.
// U+FEFF is removed. U+0085 is kept.
func jsTrim(value string) string {
	return strings.TrimFunc(value, isECMAScriptTrimRune)
}

func isECMAScriptTrimRune(r rune) bool {
	switch r {
	case '\u0009', '\u000A', '\u000B', '\u000C', '\u000D', '\uFEFF', '\u2028', '\u2029':
		return true
	default:
		return unicode.Is(unicode.Zs, r)
	}
}
