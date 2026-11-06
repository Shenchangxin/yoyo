// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

// Undefined is the JSON-absent value. Canonicalize rejects it inside a value.
type Undefined struct{}

// CanonicalValue is null, bool, number, string, array, or object.
// Numbers are float64. Objects are map[string]any. Arrays are []any.
// Null is nil.

type BlobRef struct {
	Kind      string     `json:"kind"`
	Resource  ResourceId `json:"resource"`
	Size      int64      `json:"size"`
	MediaType string     `json:"mediaType"`
}

type StoredValue struct {
	Kind      string     `json:"kind"`
	Value     any        `json:"value,omitempty"`
	Resource  ResourceId `json:"resource,omitempty"`
	Size      int64      `json:"size,omitempty"`
	MediaType string     `json:"mediaType,omitempty"`
}

type ValueSchema struct {
	Kind         string                       `json:"kind"`
	Integer      *bool                        `json:"integer,omitempty"`
	Minimum      *float64                     `json:"minimum,omitempty"`
	Maximum      *float64                     `json:"maximum,omitempty"`
	Enum         []string                     `json:"enum,omitempty"`
	MinLength    *int                         `json:"minLength,omitempty"`
	MaxLength    *int                         `json:"maxLength,omitempty"`
	Format       string                       `json:"format,omitempty"`
	Value        any                          `json:"value,omitempty"`
	Items        *ValueSchema                 `json:"items,omitempty"`
	MinItems     *int                         `json:"minItems,omitempty"`
	MaxItems     *int                         `json:"maxItems,omitempty"`
	Fields       map[string]ObjectFieldSchema `json:"fields,omitempty"`
	AllowUnknown *bool                        `json:"allowUnknown,omitempty"`
	Variants     []ValueSchema                `json:"variants,omitempty"`
	MediaTypes   []string                     `json:"mediaTypes,omitempty"`
	MaxBytes     *int64                       `json:"maxBytes,omitempty"`
}

type ObjectFieldSchema struct {
	Schema   ValueSchema `json:"schema"`
	Optional *bool       `json:"optional,omitempty"`
}

// BlobRefObjectSchema is the structural schema for a BlobRef nested inside an inline value.
func BlobRefObjectSchema(mediaTypes []string) ValueSchema {
	media := ValueSchema{Kind: "string", MinLength: intPtr(1)}
	if mediaTypes != nil {
		media = ValueSchema{Kind: "string", Enum: mediaTypes}
	}
	integer := true
	min := 0.0
	return ValueSchema{
		Kind: "object",
		Fields: map[string]ObjectFieldSchema{
			"kind":      {Schema: ValueSchema{Kind: "literal", Value: "blob"}},
			"resource":  {Schema: ValueSchema{Kind: "string", MinLength: intPtr(5)}},
			"size":      {Schema: ValueSchema{Kind: "number", Integer: &integer, Minimum: &min}},
			"mediaType": {Schema: media},
		},
	}
}

func intPtr(n int) *int { return &n }
