// Copyright 2026 Hypit.AI
// Translated from @hypit/credential-store-env/src/activation.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package credentialstoreenv

import (
	"errors"
	"reflect"

	"github.com/Shenchangxin/yoyo/remake/facet"
)

// Package is the hypit.package@1 activation record.
type Package struct {
	Format string
	Facets []facet.Facet
}

// FactoryContext is the context passed to a Runtime adapter.
// A nil Pool means the property is absent.
type FactoryContext struct {
	HostStateRoot string
	DataRoot      string
	Instance      string
	Pool          *string
	Config        any
}

// Opened is one opened store. A nil Close means this adapter has nothing to release.
type Opened struct {
	Value *EnvironmentCredentialStore
	Close func()
}

// CredentialStoreAdapter is the facet implementation.
// The doctor property is absent.
type CredentialStoreAdapter struct {
	Validate func(FactoryContext)
	Open     func(FactoryContext) Opened
}

// HypitPackage is the hypit.package@1 activation exported by @hypit/credential-store-env.
var HypitPackage = Package{
	Format: "hypit.package@1",
	Facets: []facet.Facet{environmentCredentialStoreAdapter},
}

var environmentCredentialStoreAdapter = facet.Facet{
	ABI:    "hypit.runtime-local-credential-store-adapter@1",
	Offers: []string{"@hypit/credential-store-env"},
	Implementation: CredentialStoreAdapter{
		Validate: validateEnvironmentCredentialStore,
		Open:     openEnvironmentCredentialStore,
	},
}

func validateEnvironmentCredentialStore(context FactoryContext) {
	config := runtimeConfigObject(context.Config, "environment CredentialStore")
	runtimeConfigExact(config, []string{}, "environment CredentialStore")
}

func openEnvironmentCredentialStore(FactoryContext) Opened {
	return Opened{Value: NewEnvironmentCredentialStore(nil)}
}

func runtimeConfigObject(value any, subject string) map[string]any {
	object, ok := objectValue(value)
	assert(ok, subject+" config must be an object")
	return object
}

func objectValue(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if object, ok := value.(map[string]any); ok {
		return object, object != nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map || rv.IsNil() || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		out[iter.Key().String()] = iter.Value().Interface()
	}
	return out, true
}

// runtimeConfigExact rejects keys outside allowed.
// With several rejected keys, the reported key is whichever map iteration yields first.
// JavaScript reports the first key in Object.keys order.
func runtimeConfigExact(value map[string]any, allowed []string, subject string) {
	var unknown string
	found := false
	for key := range value {
		if containsString(allowed, key) {
			continue
		}
		unknown = key
		found = true
		break
	}
	if found {
		panic(errors.New(subject + " config does not accept " + unknown))
	}
}

func containsString(values []string, key string) bool {
	for _, value := range values {
		if value == key {
			return true
		}
	}
	return false
}
