// Copyright 2026 Hypit.AI
// Translated from @hypit/blob (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package blob

import "github.com/Shenchangxin/yoyo/remake/protocol"

var BlobModuleRef = protocol.ModuleRef{Name: "@hypit/blob", Version: "1"}

var BlobTypes = struct {
	Blob protocol.TypeRef
}{
	Blob: protocol.TypeRef{Module: BlobModuleRef, Name: "Blob"},
}

var BlobManifest = protocol.ModuleManifest{
	Format:       protocol.ModulePackageABI,
	Name:         BlobModuleRef.Name,
	Version:      BlobModuleRef.Version,
	Dependencies: []protocol.ModuleDependency{},
	Types:        []protocol.TypeDeclaration{{Name: BlobTypes.Blob.Name}},
	Capabilities: []protocol.CapabilityDeclaration{},
	Producers:    []protocol.ProducerDeclaration{},
}

var BlobDependency = protocol.ModuleDependency{Module: BlobModuleRef}
