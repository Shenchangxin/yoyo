// Copyright 2026 Hypit.AI
// Translated from @hypit/narrative/src/schema.ts and manifest.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package narrative

import "github.com/Shenchangxin/yoyo/remake/protocol"

var ModuleRef = protocol.ModuleRef{Name: "@hypit/narrative", Version: "1"}

var Types = struct {
	Narrative  protocol.TypeRef
	SegmentRef protocol.TypeRef
	Selection  protocol.TypeRef
	Moment     protocol.TypeRef
}{
	Narrative:  protocol.TypeRef{Module: ModuleRef, Name: "Narrative"},
	SegmentRef: protocol.TypeRef{Module: ModuleRef, Name: "NarrativeSegmentRef"},
	Selection:  protocol.TypeRef{Module: ModuleRef, Name: "NarrativeSelection"},
	Moment:     protocol.TypeRef{Module: ModuleRef, Name: "NarrativeMoment"},
}

var Manifest = protocol.ModuleManifest{
	Format: protocol.ModulePackageABI, Name: ModuleRef.Name, Version: ModuleRef.Version,
	Dependencies: []protocol.ModuleDependency{},
	Types: []protocol.TypeDeclaration{
		{Name: Types.Narrative.Name}, {Name: Types.SegmentRef.Name},
		{Name: Types.Selection.Name}, {Name: Types.Moment.Name},
	},
	Capabilities: []protocol.CapabilityDeclaration{},
	Producers:    []protocol.ProducerDeclaration{},
}

var Dependency = protocol.ModuleDependency{Module: ModuleRef}

func textSchema() protocol.ValueSchema {
	n := 1
	return protocol.ValueSchema{Kind: "string", MinLength: &n}
}

func indexSchema() protocol.ValueSchema {
	integer := true
	minimum := 0.0
	return protocol.ValueSchema{Kind: "number", Integer: &integer, Minimum: &minimum}
}

func objectSchema(fields map[string]protocol.ObjectFieldSchema) protocol.ValueSchema {
	return protocol.ValueSchema{Kind: "object", Fields: fields}
}

func field(schema protocol.ValueSchema) protocol.ObjectFieldSchema {
	return protocol.ObjectFieldSchema{Schema: schema}
}

func optional(schema protocol.ValueSchema) protocol.ObjectFieldSchema {
	yes := true
	return protocol.ObjectFieldSchema{Schema: schema, Optional: &yes}
}

func arraySchema(items protocol.ValueSchema, minItems *int) protocol.ValueSchema {
	return protocol.ValueSchema{Kind: "array", Items: &items, MinItems: minItems}
}

// SegmentRefSchema, Schema, MomentSchema, and SelectionSchema are the structural schemas.
var (
	semanticAnchor = protocol.ValueSchema{Kind: "oneOf", Variants: []protocol.ValueSchema{
		objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": {Schema: textSchema()}, "kind": {Schema: protocol.ValueSchema{Kind: "string", Enum: []string{"segment-start", "segment-end"}}},
			"segmentId": {Schema: textSchema()},
		}),
		objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": {Schema: textSchema()}, "kind": {Schema: protocol.ValueSchema{Kind: "string", Enum: []string{"token-start", "token-end"}}},
			"segmentId": {Schema: textSchema()}, "tokenId": {Schema: textSchema()},
		}),
	}}
	SegmentRefSchema = objectSchema(map[string]protocol.ObjectFieldSchema{
		"narrativeId": field(textSchema()), "kind": field(protocol.ValueSchema{Kind: "literal", Value: "segment"}),
		"id": field(textSchema()), "tokenStart": field(indexSchema()), "tokenEndExclusive": field(indexSchema()),
	})
	Schema = objectSchema(map[string]protocol.ObjectFieldSchema{
		"id": field(textSchema()),
		"segments": field(arraySchema(objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": field(textSchema()), "startAnchorId": field(textSchema()), "endAnchorId": field(textSchema()),
			"tokenStart": field(indexSchema()), "tokenEndExclusive": field(indexSchema()),
		}), intPtr(1))),
		"tokens": field(arraySchema(objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": field(textSchema()), "segmentId": field(textSchema()),
			"startAnchorId": field(textSchema()), "endAnchorId": field(textSchema()),
			"text": field(textSchema()), "normalized": field(textSchema()),
		}), nil)),
		"turns": field(arraySchema(objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": field(textSchema()), "segmentId": field(textSchema()), "role": optional(textSchema()),
			"tokenStart": field(indexSchema()), "tokenEndExclusive": field(indexSchema()),
		}), nil)),
		"selections": field(arraySchema(objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": field(textSchema()), "startAnchorId": field(textSchema()), "endAnchorId": field(textSchema()),
		}), nil)),
		"moments": field(arraySchema(objectSchema(map[string]protocol.ObjectFieldSchema{
			"id": field(textSchema()), "anchorId": field(textSchema()),
		}), nil)),
		"anchors": field(arraySchema(semanticAnchor, intPtr(2))),
	})
	MomentSchema = objectSchema(map[string]protocol.ObjectFieldSchema{
		"narrativeId": field(textSchema()), "id": field(textSchema()), "anchorId": field(textSchema()),
	})
	SelectionSchema = objectSchema(map[string]protocol.ObjectFieldSchema{
		"narrativeId": field(textSchema()), "id": field(textSchema()),
		"startAnchorId": field(textSchema()), "endAnchorId": field(textSchema()),
	})
)

func intPtr(n int) *int { return &n }
