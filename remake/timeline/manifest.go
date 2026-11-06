// Copyright 2026 Hypit.AI
// Translated from @hypit/timeline
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package timeline

import "github.com/Shenchangxin/yoyo/remake/protocol"

// ModuleRef is the timeline module identity.
var ModuleRef = protocol.ModuleRef{Name: "@hypit/timeline", Version: "1"}

// Types declared by the timeline module.
var Types = struct {
	Timeline protocol.TypeRef
	Clock    protocol.TypeRef
}{
	Timeline: protocol.TypeRef{Module: ModuleRef, Name: "Timeline"},
	Clock:    protocol.TypeRef{Module: ModuleRef, Name: "Clock"},
}

var positiveIntegerSchema = func() protocol.ValueSchema {
	integer := true
	minimum := 1.0
	return protocol.ValueSchema{Kind: "number", Integer: &integer, Minimum: &minimum}
}()

var stringSchema = func() protocol.ValueSchema {
	min := 1
	return protocol.ValueSchema{Kind: "string", MinLength: &min}
}()

// ClockSchema is the structural schema of a Clock.
var ClockSchema = protocol.ValueSchema{
	Kind: "object",
	Fields: map[string]protocol.ObjectFieldSchema{
		"frameRate": {Schema: protocol.ValueSchema{Kind: "object", Fields: map[string]protocol.ObjectFieldSchema{
			"numerator":   {Schema: positiveIntegerSchema},
			"denominator": {Schema: positiveIntegerSchema},
		}}},
	},
}

// TimelineSchema is the structural schema of a Timeline.
var TimelineSchema = protocol.ValueSchema{
	Kind: "object",
	Fields: map[string]protocol.ObjectFieldSchema{
		"id":         {Schema: stringSchema},
		"frameRate":  ClockSchema.Fields["frameRate"],
		"frameCount": {Schema: positiveIntegerSchema},
	},
}

// Manifest is the timeline module manifest.
var Manifest = protocol.ModuleManifest{
	Format:       protocol.ModulePackageABI,
	Name:         ModuleRef.Name,
	Version:      ModuleRef.Version,
	Dependencies: []protocol.ModuleDependency{},
	Types:        []protocol.TypeDeclaration{{Name: Types.Timeline.Name}, {Name: Types.Clock.Name}},
	Capabilities: []protocol.CapabilityDeclaration{},
	Producers:    []protocol.ProducerDeclaration{},
}

// Dependency is the module dependency other packages declare.
var Dependency = protocol.ModuleDependency{Module: ModuleRef}
