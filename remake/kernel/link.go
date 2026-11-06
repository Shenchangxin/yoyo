// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/link.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"slices"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func manifestRef(manifest protocol.ModuleManifest) protocol.ModuleRef {
	return protocol.ModuleRef{Name: manifest.Name, Version: manifest.Version}
}

// closureIndex is recomputed on each lookup. The TypeScript WeakMap is an
// identity cache; Go values have no object identity, and rebuilding the maps
// returns the same declarations for an unchanged closure.
func closureIndex(closure protocol.ResolvedModuleClosure) (types map[string]protocol.ResolvedTypeDeclaration, producers map[string]protocol.ResolvedProducerDeclaration) {
	types = map[string]protocol.ResolvedTypeDeclaration{}
	producers = map[string]protocol.ResolvedProducerDeclaration{}
	for _, module := range closure.Modules {
		ref := manifestRef(module.Manifest)
		for _, declaration := range module.Manifest.Types {
			typeRef := protocol.TypeRef{Module: ref, Name: declaration.Name}
			types[typeKey(typeRef)] = protocol.ResolvedTypeDeclaration{Name: declaration.Name, Ref: typeRef}
		}
		for _, declaration := range module.Manifest.Producers {
			producer := protocol.ProducerRef{Module: ref, Name: declaration.Name}
			resolved := protocol.ResolvedProducerDeclaration{ProducerDeclaration: cloneProducer(declaration), Ref: producer}
			producers[producerKey(producer)] = resolved
		}
	}
	return types, producers
}

// CreateResolvedClosure copies every manifest. Later edits to the caller's
// slices do not change the closure.
func CreateResolvedClosure(manifests []protocol.ModuleManifest) protocol.ResolvedModuleClosure {
	modules := make([]protocol.ResolvedModule, len(manifests))
	for i, manifest := range manifests {
		modules[i] = protocol.ResolvedModule{Manifest: cloneManifest(manifest)}
	}
	return protocol.ResolvedModuleClosure{Format: "hypit.closure@1", Modules: modules}
}

func cloneManifest(manifest protocol.ModuleManifest) protocol.ModuleManifest {
	out := manifest
	out.Dependencies = cloneDeps(manifest.Dependencies)
	out.Types = slices.Clone(manifest.Types)
	out.Capabilities = slices.Clone(manifest.Capabilities)
	out.Producers = make([]protocol.ProducerDeclaration, len(manifest.Producers))
	for i, producer := range manifest.Producers {
		out.Producers[i] = cloneProducer(producer)
	}
	return out
}

func cloneDeps(values []protocol.ModuleDependency) []protocol.ModuleDependency {
	if values == nil {
		return nil
	}
	return slices.Clone(values)
}

func cloneProducer(producer protocol.ProducerDeclaration) protocol.ProducerDeclaration {
	out := producer
	out.Inputs = slices.Clone(producer.Inputs)
	out.Outputs = slices.Clone(producer.Outputs)
	out.Needs = slices.Clone(producer.Needs)
	return out
}

func ensureUniqueNames(names []string, kind, owner string) {
	seen := map[string]struct{}{}
	for _, name := range names {
		invariant(name != "", "EMPTY_NAME", kind+" name in "+owner+" is empty", owner)
		_, duplicate := seen[name]
		invariant(!duplicate, "DUPLICATE_EXPORT", owner+" declares duplicate "+kind+" "+name, name)
		seen[name] = struct{}{}
	}
}

func namesOf(count int, at func(int) string) []string {
	out := make([]string, count)
	for i := range out {
		out[i] = at(i)
	}
	return out
}

// VerifyClosure checks the module closure the linker is willing to execute.
func VerifyClosure(closure protocol.ResolvedModuleClosure) {
	invariant(closure.Format == "hypit.closure@1", "UNSUPPORTED_CLOSURE", "unsupported closure format")

	modules := map[string]protocol.ResolvedModule{}
	declaredTypes := map[string]struct{}{}
	declaredCapabilities := map[string]protocol.ResolvedCapabilityDeclaration{}
	for _, module := range closure.Modules {
		ref := manifestRef(module.Manifest)
		key := moduleKey(ref)
		invariant(ref.Name != "", "EMPTY_MODULE_NAME", "module name is empty")
		invariant(ref.Version != "", "EMPTY_MODULE_VERSION", ref.Name+" version is empty")
		invariant(module.Manifest.Format == "hypit.module@1", "UNSUPPORTED_MODULE", key+" format is unsupported")
		_, duplicate := modules[key]
		invariant(!duplicate, "DUPLICATE_MODULE", "duplicate module "+key, key)
		ensureUniqueNames(namesOf(len(module.Manifest.Types), func(i int) string { return module.Manifest.Types[i].Name }), "type", key)
		ensureUniqueNames(namesOf(len(module.Manifest.Capabilities), func(i int) string { return module.Manifest.Capabilities[i].Name }), "capability", key)
		ensureUniqueNames(namesOf(len(module.Manifest.Producers), func(i int) string { return module.Manifest.Producers[i].Name }), "producer", key)
		ensureUniqueNames(namesOf(len(module.Manifest.Dependencies), func(i int) string { return moduleKey(module.Manifest.Dependencies[i].Module) }), "dependency", key)
		for _, producer := range module.Manifest.Producers {
			owner := key + "#" + producer.Name
			ensureUniqueNames(namesOf(len(producer.Inputs), func(i int) string { return producer.Inputs[i].Name }), "input port", owner)
			ensureUniqueNames(namesOf(len(producer.Outputs), func(i int) string { return producer.Outputs[i].Name }), "output port", owner)
			ensureUniqueNames(namesOf(len(producer.Needs), func(i int) string { return producer.Needs[i].Name }), "need port", owner)
			invariant(len(producer.Outputs)+len(producer.Needs) == 1, "PRODUCER_RESULT_NORMAL_FORM", owner+" must declare exactly one public result")
		}
		for _, item := range module.Manifest.Types {
			declaredTypes[typeKey(protocol.TypeRef{Module: ref, Name: item.Name})] = struct{}{}
		}
		for _, capability := range module.Manifest.Capabilities {
			capabilityRef := protocol.CapabilityRef{Module: ref, Name: capability.Name}
			declaredCapabilities[capabilityKey(capabilityRef)] = protocol.ResolvedCapabilityDeclaration{
				CapabilityDeclaration: capability,
				Ref:                   capabilityRef,
			}
		}
		modules[key] = module
	}

	for _, module := range closure.Modules {
		for _, dependency := range module.Manifest.Dependencies {
			_, resolved := modules[moduleKey(dependency.Module)]
			invariant(resolved, "MISSING_DEPENDENCY", moduleKey(manifestRef(module.Manifest))+" requires "+moduleKey(dependency.Module))
		}
	}

	for _, module := range closure.Modules {
		allowed := map[string]struct{}{moduleKey(manifestRef(module.Manifest)): {}}
		for _, dependency := range module.Manifest.Dependencies {
			allowed[moduleKey(dependency.Module)] = struct{}{}
		}
		for _, producer := range module.Manifest.Producers {
			for _, port := range append(append([]protocol.PortDeclaration{}, producer.Inputs...), producer.Outputs...) {
				invariant(hasKey(allowed, moduleKey(port.Type.Module)), "UNDECLARED_TYPE_DEPENDENCY", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references "+typeKey(port.Type)+" without a dependency")
				_, known := declaredTypes[typeKey(port.Type)]
				invariant(known, "UNKNOWN_TYPE", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references unknown type "+typeKey(port.Type))
			}
			for _, port := range producer.Needs {
				invariant(hasKey(allowed, moduleKey(port.Returns.Module)), "UNDECLARED_TYPE_DEPENDENCY", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references "+typeKey(port.Returns)+" without a dependency")
				_, known := declaredTypes[typeKey(port.Returns)]
				invariant(known, "UNKNOWN_TYPE", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references unknown type "+typeKey(port.Returns))
				invariant(hasKey(allowed, moduleKey(port.Capability.Module)), "UNDECLARED_CAPABILITY_DEPENDENCY", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references "+capabilityKey(port.Capability)+" without a dependency")
				capability, found := declaredCapabilities[capabilityKey(port.Capability)]
				invariant(found, "UNKNOWN_CAPABILITY", moduleKey(manifestRef(module.Manifest))+"#"+producer.Name+" references unknown capability "+capabilityKey(port.Capability))
				invariant(typeKey(capability.Returns) == typeKey(port.Returns), "CAPABILITY_RETURN_MISMATCH", capabilityKey(port.Capability)+" returns "+typeKey(capability.Returns)+", not "+typeKey(port.Returns))
			}
		}
		for _, capability := range module.Manifest.Capabilities {
			ref := protocol.CapabilityRef{Module: manifestRef(module.Manifest), Name: capability.Name}
			invariant(hasKey(allowed, moduleKey(capability.Returns.Module)), "UNDECLARED_TYPE_DEPENDENCY", capabilityKey(ref)+" returns "+typeKey(capability.Returns)+" without a dependency")
			_, known := declaredTypes[typeKey(capability.Returns)]
			invariant(known, "UNKNOWN_TYPE", capabilityKey(ref)+" returns unknown type "+typeKey(capability.Returns))
		}
	}
}

func hasKey(values map[string]struct{}, key string) bool {
	_, ok := values[key]
	return ok
}

// ResolveType returns the type declaration named by ref.
func ResolveType(closure protocol.ResolvedModuleClosure, ref protocol.TypeRef) protocol.ResolvedTypeDeclaration {
	types, _ := closureIndex(closure)
	declaration, ok := types[typeKey(ref)]
	invariant(ok, "UNKNOWN_TYPE", "unknown type "+typeKey(ref), typeKey(ref))
	return declaration
}

// ResolveProducer returns the producer declaration named by ref.
func ResolveProducer(closure protocol.ResolvedModuleClosure, ref protocol.ProducerRef) protocol.ResolvedProducerDeclaration {
	_, producers := closureIndex(closure)
	declaration, ok := producers[producerKey(ref)]
	invariant(ok, "UNKNOWN_PRODUCER", "unknown producer "+producerKey(ref), producerKey(ref))
	return declaration
}

// SealRecord returns a shallow copy of record.
func SealRecord(record protocol.TypedRecord) protocol.TypedRecord {
	return record
}

// VerifyRecordStructure checks that record names a type in the closure.
func VerifyRecordStructure(closure protocol.ResolvedModuleClosure, record protocol.TypedRecord) protocol.ResolvedTypeDeclaration {
	invariant(record.ID != "", "EMPTY_RECORD_ID", "record id is empty")
	return ResolveType(closure, record.Type)
}

// Link copies the authored records onto the closure and checks the program.
func Link(closure protocol.ResolvedModuleClosure, authoredRecords []protocol.TypedRecord) protocol.LinkedProgram {
	program := protocol.LinkedProgram{
		Closure: closure,
		Records: append([]protocol.TypedRecord(nil), authoredRecords...),
	}
	verifyLinkedProgram(program)
	return program
}

func verifyLinkedProgram(program protocol.LinkedProgram) {
	VerifyClosure(program.Closure)
	ids := map[string]struct{}{}
	for _, record := range program.Records {
		_, duplicate := ids[record.ID]
		invariant(!duplicate, "DUPLICATE_RECORD", "duplicate record "+record.ID, record.ID)
		ids[record.ID] = struct{}{}
		VerifyRecordStructure(program.Closure, record)
	}
}
