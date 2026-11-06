// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime/src/credentials.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
)

// CredentialRef is one credential lookup.
// Store is the CredentialStore implementation name, for example env, os, or aws-default.
// Key is the store-local lookup key. It is configuration, never the secret value.
type CredentialRef struct {
	Store string
	Key   string
}

// CredentialValue is the secret revealed to the selected Endpoint invocation.
// A nil ExpiresAt means the property is absent.
type CredentialValue struct {
	Secret    string
	ExpiresAt *float64
}

// OAuth2Credential is the encoded credential carried in a secret string.
type OAuth2Credential struct {
	Format       string
	AccessToken  string
	RefreshToken *string
	ExpiresAt    *float64
}

type oauthWire struct {
	Format       string   `json:"format"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken *string  `json:"refreshToken,omitempty"`
	ExpiresAt    *float64 `json:"expiresAt,omitempty"`
}

// EncodeOAuth2Credential serializes one OAuth credential.
func EncodeOAuth2Credential(value OAuth2Credential) string {
	assert(len(value.AccessToken) > 0, "OAuth access token is empty")
	if value.RefreshToken != nil {
		assert(len(*value.RefreshToken) > 0, "OAuth refresh token is empty")
	}
	if value.ExpiresAt != nil {
		assert(!math.IsNaN(*value.ExpiresAt) && !math.IsInf(*value.ExpiresAt, 0), "OAuth expiry is invalid")
	}
	raw, err := jsonMarshal(oauthWire{
		Format: "hypit.oauth2-credential@1", AccessToken: value.AccessToken,
		RefreshToken: value.RefreshToken, ExpiresAt: value.ExpiresAt,
	})
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// DecodeOAuth2Credential parses one encoded secret.
// Invalid JSON or an invalid shape returns nil.
func DecodeOAuth2Credential(secret string) *OAuth2Credential {
	var value any
	if err := json.Unmarshal([]byte(secret), &value); err != nil {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	format, _ := object["format"].(string)
	access, accessOK := object["accessToken"].(string)
	if format != "hypit.oauth2-credential@1" || !accessOK || access == "" {
		return nil
	}
	decoded := &OAuth2Credential{Format: format, AccessToken: access}
	if raw, present := object["refreshToken"]; present {
		token, tokenOK := raw.(string)
		if !tokenOK || token == "" {
			return nil
		}
		decoded.RefreshToken = &token
	}
	if raw, present := object["expiresAt"]; present {
		expiry, expiryOK := raw.(float64)
		if !expiryOK || math.IsNaN(expiry) || math.IsInf(expiry, 0) {
			return nil
		}
		decoded.ExpiresAt = &expiry
	}
	return decoded
}

// CredentialAcquisition is the host-facing way to acquire one credential.
type CredentialAcquisition struct {
	Kind                  string
	AuthorizationEndpoint string
	RedirectURI           string
	TokenEndpoint         string
	ClientID              string
	Scopes                []string
	RequestTimeoutMs      float64
}

// CredentialStore resolves one credential.
type CredentialStore interface {
	Resolve(ref CredentialRef) (*CredentialValue, error)
}

// WritableCredentialStore can prove ownership and change a credential.
type WritableCredentialStore interface {
	CredentialStore
	Owns(ref CredentialRef) bool
	Put(ref CredentialRef, value CredentialValue) error
	Delete(ref CredentialRef) (bool, error)
}

// CredentialRef builds one checked reference.
func CredentialRefOf(store, key string) CredentialRef {
	value := CredentialRef{Store: store, Key: key}
	VerifyCredentialRef(value)
	return value
}

// VerifyCredentialRef rejects an empty store or key.
func VerifyCredentialRef(ref CredentialRef) {
	assert(len(jsTrim(ref.Store)) > 0, "CredentialRef store is empty")
	assert(len(jsTrim(ref.Key)) > 0, "CredentialRef key is empty")
}

// IsWritableCredentialStore reports whether value exposes owns, put, and delete.
func IsWritableCredentialStore(value CredentialStore) bool {
	_, ok := value.(WritableCredentialStore)
	return ok
}

// WritableCredentialStoreFor returns the unique writable owner of ref.
func WritableCredentialStoreFor(store CredentialStore, ref CredentialRef) WritableCredentialStore {
	VerifyCredentialRef(ref)
	if composite, ok := store.(*CompositeCredentialStore); ok {
		return composite.Writable(ref)
	}
	writable, ok := store.(WritableCredentialStore)
	if ok && writable.Owns(ref) {
		return writable
	}
	return nil
}

// CompositeCredentialStore composes independently selected stores.
// Every implementation sees the exact CredentialRef and must decline names it does not own.
type CompositeCredentialStore struct {
	stores []CredentialStore
}

// NewCompositeCredentialStore copies the store list.
func NewCompositeCredentialStore(stores []CredentialStore) *CompositeCredentialStore {
	return &CompositeCredentialStore{stores: append([]CredentialStore{}, stores...)}
}

// Resolve returns the single store that answers ref.
func (s *CompositeCredentialStore) Resolve(ref CredentialRef) (*CredentialValue, error) {
	VerifyCredentialRef(ref)
	var resolved []*CredentialValue
	for _, store := range s.stores {
		value, err := store.Resolve(ref)
		if err != nil {
			return nil, err
		}
		if value != nil {
			resolved = append(resolved, value)
		}
	}
	assert(len(resolved) <= 1, "CredentialRef store "+ref.Store+" is implemented more than once")
	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved[0], nil
}

// Writable returns the single writable owner of ref.
func (s *CompositeCredentialStore) Writable(ref CredentialRef) WritableCredentialStore {
	VerifyCredentialRef(ref)
	var candidates []WritableCredentialStore
	for _, store := range s.stores {
		writable, ok := store.(WritableCredentialStore)
		if !ok {
			continue
		}
		if writable.Owns(ref) {
			candidates = append(candidates, writable)
		}
	}
	assert(len(candidates) <= 1, "CredentialRef store "+ref.Store+" has several writable owners")
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

func assert(condition bool, message string) {
	if !condition {
		panic(errors.New(message))
	}
}

func jsonMarshal(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
