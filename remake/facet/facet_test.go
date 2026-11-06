// Copyright 2026 Hypit.AI
// Translated from @hypit/facet (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package facet

import (
	"strings"
	"testing"
)

func TestCreateFrontendFacetKeepsTheFrontendIdentity(t *testing.T) {
	facet := CreateFrontendFacet("hypit.frontend@1", "source", FrontendImplementation{ID: "text"})
	if facet.ABI != "hypit.frontend@1" || len(facet.Offers) != 1 || facet.Offers[0] != "text" {
		t.Fatalf("%+v", facet)
	}
	got := FrontendsFromFacets("hypit.frontend@1", "source", []Facet{facet})
	if len(got) != 1 {
		t.Fatalf("%v", got)
	}
	if id, ok := frontendID(got[0]); !ok || id != "text" {
		t.Fatalf("%#v", got[0])
	}
}

func TestCreateFrontendFacetRejectsABlankIdentity(t *testing.T) {
	for _, call := range []func(){
		func() { CreateFrontendFacet("  ", "source", FrontendImplementation{ID: "text"}) },
		func() { CreateFrontendFacet("abi", "", FrontendImplementation{ID: "text"}) },
		func() { CreateFrontendFacet("abi", "source", FrontendImplementation{ID: "  "}) },
	} {
		message := recoverMessage(t, call)
		if !strings.Contains(message, "Frontend facet has an invalid identity") {
			t.Fatal(message)
		}
	}
	message := recoverMessage(t, func() { CreateFrontendFacet("abi", "", FrontendImplementation{ID: "text"}) })
	if message != "Source Frontend facet has an invalid identity" {
		t.Fatal(message)
	}
}

func TestFrontendsFromFacetsSkipsOtherOwnersAndRejectsABadFrontend(t *testing.T) {
	good := CreateFrontendFacet("abi", "source", FrontendImplementation{ID: "text"})
	other := CreateFrontendFacet("other", "source", FrontendImplementation{ID: "nope"})
	wrongKind := CreateFrontendFacet("abi", "preview", FrontendImplementation{ID: "view"})
	got := FrontendsFromFacets("abi", "source", []Facet{other, wrongKind, good, {ABI: "abi", Implementation: nil}})
	if len(got) != 1 {
		t.Fatalf("%v", got)
	}
	bad := Facet{ABI: "abi", Implementation: map[string]any{
		"kind":     "source",
		"frontend": map[string]any{"id": 1},
	}}
	message := recoverMessage(t, func() { FrontendsFromFacets("abi", "source", []Facet{bad}) })
	if message != "source Frontend facet has an invalid implementation" {
		t.Fatal(message)
	}
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
