// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/link.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package kernel

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestReferenceKeysMatchTheProtocol(t *testing.T) {
	ref := protocol.ModuleRef{Name: "a", Version: "1"}
	if moduleKey(ref) != "a@1" || !sameModule(ref, ref) || sameModule(ref, protocol.ModuleRef{Name: "b", Version: "1"}) {
		t.Fatal("module identity")
	}
	typ := protocol.TypeRef{Module: ref, Name: "T"}
	if typeKey(typ) != "a@1#T" || !sameType(typ, typ) || sameType(typ, protocol.TypeRef{Module: ref, Name: "U"}) {
		t.Fatal("type identity")
	}
	if producerKey(protocol.ProducerRef{Module: ref, Name: "P"}) != "a@1#P" {
		t.Fatal("producer identity")
	}
	if capabilityKey(protocol.CapabilityRef{Module: ref, Name: "C"}) != "a@1#C" {
		t.Fatal("capability identity")
	}
}

func TestCreateResolvedClosureCopiesManifests(t *testing.T) {
	manifest := textModule()
	closure := CreateResolvedClosure([]protocol.ModuleManifest{manifest})
	manifest.Types[0].Name = "Changed"
	manifest.Name = "changed"
	if closure.Modules[0].Manifest.Name != "@hypit/example" || closure.Modules[0].Manifest.Types[0].Name != "Text" {
		t.Fatal("closure retained the caller's manifest")
	}
	if closure.Format != "hypit.closure@1" {
		t.Fatal(closure.Format)
	}
}

func TestLinkAcceptsOneTypedRecord(t *testing.T) {
	closure := CreateResolvedClosure([]protocol.ModuleManifest{emittingModule()})
	record := textRecord("rec-1")
	program := Link(closure, []protocol.TypedRecord{record})
	if len(program.Records) != 1 || program.Records[0].ID != "rec-1" {
		t.Fatal("linked records")
	}
	resolved := ResolveType(closure, textType())
	if resolved.Name != "Text" || resolved.Ref.Name != "Text" {
		t.Fatalf("%+v", resolved)
	}
	producer := ResolveProducer(closure, protocol.ProducerRef{Module: exampleRef(), Name: "Emit"})
	if producer.Name != "Emit" || len(producer.Outputs) != 1 {
		t.Fatalf("%+v", producer)
	}
	sealed := SealRecord(record)
	record.ID = "other"
	if sealed.ID != "rec-1" {
		t.Fatal("sealRecord did not copy the record")
	}
}

func TestVerifyClosureRejects(t *testing.T) {
	text := textType()
	other := protocol.ModuleRef{Name: "@hypit/other", Version: "1"}
	cases := []struct {
		name    string
		closure protocol.ResolvedModuleClosure
		code    string
	}{
		{name: "format", closure: protocol.ResolvedModuleClosure{Format: "nope"}, code: "UNSUPPORTED_CLOSURE"},
		{name: "empty name", closure: closureOf(protocol.ModuleManifest{Format: "hypit.module@1", Version: "1"}), code: "EMPTY_MODULE_NAME"},
		{name: "empty version", closure: closureOf(protocol.ModuleManifest{Format: "hypit.module@1", Name: "@hypit/example"}), code: "EMPTY_MODULE_VERSION"},
		{name: "module format", closure: closureOf(protocol.ModuleManifest{Format: "nope", Name: "@hypit/example", Version: "1"}), code: "UNSUPPORTED_MODULE"},
		{name: "duplicate module", closure: closureOf(textModule(), textModule()), code: "DUPLICATE_MODULE"},
		{name: "empty type", closure: closureOf(withTypes(textModule(), "", "Text")), code: "EMPTY_NAME"},
		{name: "duplicate type", closure: closureOf(withTypes(textModule(), "Text", "Text")), code: "DUPLICATE_EXPORT"},
		{name: "duplicate dependency", closure: closureOf(withDeps(textModule(), other, other)), code: "DUPLICATE_EXPORT"},
		{name: "no result", closure: closureOf(withProducer(textModule(), protocol.ProducerDeclaration{Name: "Emit"})), code: "PRODUCER_RESULT_NORMAL_FORM"},
		{name: "two results", closure: closureOf(withProducer(textModule(), protocol.ProducerDeclaration{
			Name: "Emit", Outputs: []protocol.PortDeclaration{{Name: "out", Type: text}},
			Needs: []protocol.NeedPortDeclaration{{Name: "ask", Capability: protocol.CapabilityRef{Module: exampleRef(), Name: "Ask"}, Returns: text}},
		})), code: "PRODUCER_RESULT_NORMAL_FORM"},
		{name: "missing dependency", closure: closureOf(withDeps(textModule(), other)), code: "MISSING_DEPENDENCY"},
		{name: "undeclared type", closure: closureOf(withProducer(textModule(), outputPort("Emit", protocol.TypeRef{Module: other, Name: "Text"}))), code: "UNDECLARED_TYPE_DEPENDENCY"},
		{name: "unknown type", closure: closureOf(withProducer(textModule(), outputPort("Emit", protocol.TypeRef{Module: exampleRef(), Name: "Missing"}))), code: "UNKNOWN_TYPE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mustSvml(t, func() { VerifyClosure(tc.closure) })
			if err.Code != tc.code {
				t.Fatalf("code %s (%s)", err.Code, err.Message)
			}
		})
	}
}

func TestCapabilityChecks(t *testing.T) {
	text := textType()
	base := textModule()
	base.Capabilities = []protocol.CapabilityDeclaration{{Name: "Ask", Returns: text}}
	good := withProducer(base, protocol.ProducerDeclaration{
		Name:  "Asker",
		Needs: []protocol.NeedPortDeclaration{{Name: "ask", Capability: protocol.CapabilityRef{Module: exampleRef(), Name: "Ask"}, Returns: text}},
	})
	VerifyClosure(closureOf(good))

	mismatch := withProducer(base, protocol.ProducerDeclaration{
		Name: "Asker",
		Needs: []protocol.NeedPortDeclaration{{
			Name: "ask", Capability: protocol.CapabilityRef{Module: exampleRef(), Name: "Ask"},
			Returns: protocol.TypeRef{Module: exampleRef(), Name: "Missing"},
		}},
	})
	// The return type is unknown before the capability return is compared.
	err := mustSvml(t, func() { VerifyClosure(closureOf(withTypes(mismatch, "Text", "Missing"))) })
	if err.Code != "CAPABILITY_RETURN_MISMATCH" {
		t.Fatal(err.Code, err.Message)
	}

	unknown := withProducer(base, protocol.ProducerDeclaration{
		Name:  "Asker",
		Needs: []protocol.NeedPortDeclaration{{Name: "ask", Capability: protocol.CapabilityRef{Module: exampleRef(), Name: "Nope"}, Returns: text}},
	})
	err = mustSvml(t, func() { VerifyClosure(closureOf(unknown)) })
	if err.Code != "UNKNOWN_CAPABILITY" {
		t.Fatal(err.Code, err.Message)
	}

	foreign := protocol.ModuleRef{Name: "@hypit/other", Version: "1"}
	undeclared := withProducer(base, protocol.ProducerDeclaration{
		Name:  "Asker",
		Needs: []protocol.NeedPortDeclaration{{Name: "ask", Capability: protocol.CapabilityRef{Module: foreign, Name: "Ask"}, Returns: text}},
	})
	err = mustSvml(t, func() { VerifyClosure(closureOf(undeclared)) })
	if err.Code != "UNDECLARED_CAPABILITY_DEPENDENCY" {
		t.Fatal(err.Code, err.Message)
	}

	badReturn := base
	badReturn.Capabilities = []protocol.CapabilityDeclaration{{
		Name:    "Ask",
		Returns: protocol.TypeRef{Module: foreign, Name: "Text"},
	}}
	err = mustSvml(t, func() { VerifyClosure(closureOf(badReturn)) })
	if err.Code != "UNDECLARED_TYPE_DEPENDENCY" {
		t.Fatal(err.Code, err.Message)
	}
}

func TestLinkRejectsADuplicateOrUnknownRecord(t *testing.T) {
	closure := CreateResolvedClosure([]protocol.ModuleManifest{textModule()})
	err := mustSvml(t, func() {
		Link(closure, []protocol.TypedRecord{textRecord("rec-1"), textRecord("rec-1")})
	})
	if err.Code != "DUPLICATE_RECORD" || err.Subject == nil || *err.Subject != "rec-1" {
		t.Fatalf("%+v", err)
	}
	err = mustSvml(t, func() {
		Link(closure, []protocol.TypedRecord{{ID: "", Type: textType(), Value: protocol.StoredValue{Kind: "inline", Value: "x"}}})
	})
	if err.Code != "EMPTY_RECORD_ID" {
		t.Fatal(err.Code)
	}
	err = mustSvml(t, func() { ResolveType(closure, protocol.TypeRef{Module: exampleRef(), Name: "Missing"}) })
	if err.Code != "UNKNOWN_TYPE" || err.Subject == nil {
		t.Fatalf("%+v", err)
	}
	err = mustSvml(t, func() {
		ResolveProducer(closure, protocol.ProducerRef{Module: exampleRef(), Name: "Missing"})
	})
	if err.Code != "UNKNOWN_PRODUCER" {
		t.Fatal(err.Code)
	}
}

func exampleRef() protocol.ModuleRef {
	return protocol.ModuleRef{Name: "@hypit/example", Version: "1"}
}

func textType() protocol.TypeRef {
	return protocol.TypeRef{Module: exampleRef(), Name: "Text"}
}

func textModule() protocol.ModuleManifest {
	return protocol.ModuleManifest{
		Format:       "hypit.module@1",
		Name:         "@hypit/example",
		Version:      "1",
		Dependencies: []protocol.ModuleDependency{},
		Types:        []protocol.TypeDeclaration{{Name: "Text"}},
		Capabilities: []protocol.CapabilityDeclaration{},
		Producers:    []protocol.ProducerDeclaration{},
	}
}

func emittingModule() protocol.ModuleManifest {
	return withProducer(textModule(), outputPort("Emit", textType()))
}

func outputPort(name string, typ protocol.TypeRef) protocol.ProducerDeclaration {
	return protocol.ProducerDeclaration{
		Name:    name,
		Inputs:  []protocol.PortDeclaration{},
		Outputs: []protocol.PortDeclaration{{Name: "out", Type: typ}},
		Needs:   []protocol.NeedPortDeclaration{},
	}
}

func withTypes(manifest protocol.ModuleManifest, names ...string) protocol.ModuleManifest {
	manifest.Types = nil
	for _, name := range names {
		manifest.Types = append(manifest.Types, protocol.TypeDeclaration{Name: name})
	}
	return manifest
}

func withDeps(manifest protocol.ModuleManifest, refs ...protocol.ModuleRef) protocol.ModuleManifest {
	manifest.Dependencies = nil
	for _, ref := range refs {
		manifest.Dependencies = append(manifest.Dependencies, protocol.ModuleDependency{Module: ref})
	}
	return manifest
}

func withProducer(manifest protocol.ModuleManifest, producer protocol.ProducerDeclaration) protocol.ModuleManifest {
	manifest.Producers = append([]protocol.ProducerDeclaration{}, manifest.Producers...)
	manifest.Producers = append(manifest.Producers, producer)
	return manifest
}

func closureOf(manifests ...protocol.ModuleManifest) protocol.ResolvedModuleClosure {
	return CreateResolvedClosure(manifests)
}

func textRecord(id string) protocol.TypedRecord {
	return protocol.TypedRecord{ID: id, Type: textType(), Value: protocol.StoredValue{Kind: "inline", Value: "hello"}}
}

func mustSvml(t *testing.T, fn func()) *protocol.SvmlError {
	t.Helper()
	var got *protocol.SvmlError
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(*protocol.SvmlError)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			got = err
		}()
		fn()
	}()
	if got == nil {
		t.Fatal("expected SvmlError")
	}
	return got
}
