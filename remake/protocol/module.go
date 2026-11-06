// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

// ModulePackageABI is the logical package address for immutable semantic Modules.
const ModulePackageABI = "hypit.module@1"

type ModuleDependency struct {
	Module ModuleRef `json:"module"`
}

type TypeDeclaration struct {
	Name string `json:"name"`
}

type PortDeclaration struct {
	Name string  `json:"name"`
	Type TypeRef `json:"type"`
}

type NeedPortDeclaration struct {
	Name       string        `json:"name"`
	Capability CapabilityRef `json:"capability"`
	Returns    TypeRef       `json:"returns"`
}

type CapabilityDeclaration struct {
	Name    string  `json:"name"`
	Returns TypeRef `json:"returns"`
}

type ProducerDeclaration struct {
	Name    string                `json:"name"`
	Inputs  []PortDeclaration     `json:"inputs"`
	Outputs []PortDeclaration     `json:"outputs"`
	Needs   []NeedPortDeclaration `json:"needs"`
}

type ModuleManifest struct {
	Format       string                  `json:"format"`
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	Dependencies []ModuleDependency      `json:"dependencies"`
	Types        []TypeDeclaration       `json:"types"`
	Capabilities []CapabilityDeclaration `json:"capabilities"`
	Producers    []ProducerDeclaration   `json:"producers"`
}

type ResolvedModule struct {
	Manifest ModuleManifest `json:"manifest"`
}

type ResolvedModuleClosure struct {
	Format  string           `json:"format"`
	Modules []ResolvedModule `json:"modules"`
}

type ResolvedTypeDeclaration struct {
	Name string  `json:"name"`
	Ref  TypeRef `json:"ref"`
}

type ResolvedProducerDeclaration struct {
	ProducerDeclaration
	Ref ProducerRef `json:"ref"`
}

type ResolvedCapabilityDeclaration struct {
	CapabilityDeclaration
	Ref CapabilityRef `json:"ref"`
}
