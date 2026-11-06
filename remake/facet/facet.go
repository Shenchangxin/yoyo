// Copyright 2026 Hypit.AI
// Translated from @hypit/facet (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package facet is the inert package-contribution record.
// Only the owner of an ABI reads its implementation.
package facet

import (
	"errors"
	"reflect"
	"strings"
)

// Facet is one package contribution understood only by the owner of its exact ABI.
type Facet struct {
	ABI            string
	Offers         []string
	Implementation any
}

// FrontendImplementation is the identity every contributed frontend carries.
type FrontendImplementation struct {
	ID string
}

// FrontendBody is the implementation object stored by CreateFrontendFacet.
type FrontendBody struct {
	Kind     string
	Frontend any
}

// CreateFrontendFacet builds the inert declaration shared by every contributed Source frontend.
// It throws when the ABI, kind, or frontend id is empty after trimming.
func CreateFrontendFacet(abi, kind string, frontend any) Facet {
	id, idIsString := frontendID(frontend)
	if strings.TrimSpace(abi) == "" || strings.TrimSpace(kind) == "" || !idIsString || strings.TrimSpace(id) == "" {
		label := kind
		if kind == "" {
			label = "Source"
		}
		panic(errors.New(label + " Frontend facet has an invalid identity"))
	}
	return Facet{
		ABI:            abi,
		Offers:         []string{id},
		Implementation: FrontendBody{Kind: kind, Frontend: frontend},
	}
}

// FrontendsFromFacets selects frontends owned by one exact facet ABI and frontend kind.
// A matching facet whose frontend id is not a string throws.
func FrontendsFromFacets(abi, kind string, facets []Facet) []any {
	var values []any
	for _, facet := range facets {
		if facet.ABI != abi || !dataObject(facet.Implementation) {
			continue
		}
		implKind, frontend, ok := implementationFields(facet.Implementation)
		if !ok || implKind != kind {
			continue
		}
		if !dataObject(frontend) {
			panic(errors.New(kind + " Frontend facet has an invalid implementation"))
		}
		if _, idIsString := frontendID(frontend); !idIsString {
			panic(errors.New(kind + " Frontend facet has an invalid implementation"))
		}
		values = append(values, frontend)
	}
	return values
}

func dataObject(value any) bool {
	if value == nil {
		return false
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		return !rv.IsNil() && rv.Type().Key().Kind() == reflect.String
	case reflect.Struct:
		return true
	case reflect.Pointer:
		return !rv.IsNil() && rv.Elem().Kind() == reflect.Struct
	default:
		return false
	}
}

func implementationFields(value any) (kind string, frontend any, ok bool) {
	switch impl := value.(type) {
	case FrontendBody:
		return impl.Kind, impl.Frontend, true
	case map[string]any:
		kind, isString := impl["kind"].(string)
		if !isString {
			return "", nil, false
		}
		return kind, impl["frontend"], true
	default:
		return "", nil, false
	}
}

func frontendID(frontend any) (string, bool) {
	switch value := frontend.(type) {
	case FrontendImplementation:
		return value.ID, true
	case map[string]any:
		id, ok := value["id"].(string)
		return id, ok
	default:
		rv := reflect.ValueOf(frontend)
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			field := rv.FieldByName("ID")
			if field.IsValid() && field.Kind() == reflect.String && field.CanInterface() {
				return field.String(), true
			}
		}
		return "", false
	}
}
