// Copyright 2026 Hypit.AI
// Translated from @hypit/admission (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package admission

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestRegistryRejectsASecondHandlerForOneType(t *testing.T) {
	registry := &TypeValidatorRegistry{}
	ref := textType()
	registry.Register(ref, func(TypeValidatorContext) {})
	err := mustValidation(t, func() { registry.Register(ref, func(TypeValidatorContext) {}) })
	if err.Code != "DUPLICATE_TYPE_VALIDATOR" || err.Subject == nil || *err.Subject != "@hypit/example@1#Text" {
		t.Fatalf("%+v", err)
	}
}

func TestAdmitRecordClonesTheValueBeforeTheHandler(t *testing.T) {
	closure := kernel.CreateResolvedClosure([]protocol.ModuleManifest{textModule()})
	registry := &TypeValidatorRegistry{}
	registry.Register(textType(), func(ctx TypeValidatorContext) {
		ctx.Type.Name = "Changed"
		ctx.Value.Value.(map[string]any)["text"] = "changed"
	})
	record := protocol.TypedRecord{
		ID:    "rec-1",
		Type:  textType(),
		Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{"text": "hello"}},
	}
	got := AdmitRecord(closure, record, registry)
	if got.ID != "rec-1" || record.Type.Name != "Text" {
		t.Fatal("admit changed the record type")
	}
	if record.Value.Value.(map[string]any)["text"] != "hello" {
		t.Fatal("handler mutated the admitted value")
	}
}

func TestRefinementRejectionWrapsTheHandlerFailure(t *testing.T) {
	closure := kernel.CreateResolvedClosure([]protocol.ModuleManifest{textModule()})
	registry := &TypeValidatorRegistry{}
	registry.Register(textType(), func(TypeValidatorContext) { panic(errString("nope")) })
	err := mustValidation(t, func() {
		AdmitRecord(closure, protocol.TypedRecord{
			ID: "rec-1", Type: textType(), Value: protocol.StoredValue{Kind: "inline", Value: "x"},
		}, registry)
	})
	if err.Code != "TYPE_REFINEMENT_REJECTED" || err.Message != "@hypit/example@1#Text rejected the value: nope" {
		t.Fatal(err.Message)
	}
}

func TestUnknownTypeFailsBeforeRefinement(t *testing.T) {
	closure := kernel.CreateResolvedClosure([]protocol.ModuleManifest{textModule()})
	defer func() {
		rec := recover()
		err, ok := rec.(*protocol.SvmlError)
		if !ok || err.Code != "UNKNOWN_TYPE" {
			t.Fatalf("%#v", rec)
		}
	}()
	ValidateValue(closure, protocol.TypeRef{Module: exampleRef(), Name: "Missing"}, protocol.StoredValue{Kind: "inline"}, &TypeValidatorRegistry{})
}

func TestAdmissionFacetKeepsOnlyPresentValidators(t *testing.T) {
	blank := CreateAdmissionPackageFacet(nil)
	if blank.ABI != AdmissionPackageFacetABI || blank.Implementation.Validators != nil {
		t.Fatalf("%+v", blank)
	}
	handler := func(TypeValidatorContext) {}
	facetValue := CreateAdmissionPackageFacet(AdmissionPackage{Validators: []TypeValidatorFacet{{Type: textType(), Handler: handler}}})
	packages := AdmissionPackagesFromFacets([]facet.Facet{
		{ABI: "other"},
		{ABI: AdmissionPackageFacetABI, Implementation: facetValue.Implementation},
	})
	if len(packages) != 1 || len(packages[0].Validators) != 1 {
		t.Fatalf("%+v", packages)
	}
	registry := &TypeValidatorRegistry{}
	RegisterTypeValidatorFacets(registry, packages[0].Validators)
	if _, ok := registry.Resolve(textType()); !ok {
		t.Fatal("validator was not registered")
	}
	CreateRecordAdmitter(registry)(kernel.CreateResolvedClosure([]protocol.ModuleManifest{textModule()}), protocol.TypedRecord{
		ID: "rec-1", Type: textType(), Value: protocol.StoredValue{Kind: "inline", Value: "x"},
	})
}

type errString string

func (e errString) Error() string { return string(e) }

func exampleRef() protocol.ModuleRef {
	return protocol.ModuleRef{Name: "@hypit/example", Version: "1"}
}

func textType() protocol.TypeRef {
	return protocol.TypeRef{Module: exampleRef(), Name: "Text"}
}

func textModule() protocol.ModuleManifest {
	return protocol.ModuleManifest{
		Format: "hypit.module@1", Name: "@hypit/example", Version: "1",
		Types: []protocol.TypeDeclaration{{Name: "Text"}},
	}
}

func mustValidation(t *testing.T, fn func()) *TypeValidationError {
	t.Helper()
	var got *TypeValidationError
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(*TypeValidationError)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			got = err
		}()
		fn()
	}()
	if got == nil {
		t.Fatal("expected TypeValidationError")
	}
	return got
}
