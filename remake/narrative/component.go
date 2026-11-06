// Copyright 2026 Hypit.AI
// Translated from @hypit/narrative/src/component.ts and activation.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package narrative

import (
	"errors"

	"github.com/Shenchangxin/yoyo/remake/admission"
	"github.com/Shenchangxin/yoyo/remake/producer"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// Component validates narrative values admitted into a build.
var Component = struct {
	Validators []admission.TypeValidatorFacet
}{Validators: []admission.TypeValidatorFacet{
	{Type: Types.Narrative, Handler: func(ctx admission.TypeValidatorContext) { AssertIdentity(inlineNarrative(ctx.Value, "Narrative")) }},
	{Type: Types.SegmentRef, Handler: func(ctx admission.TypeValidatorContext) {
		AssertSegmentRefIdentity(inlineSegment(ctx.Value, "NarrativeSegmentRef"))
	}},
	{Type: Types.Selection, Handler: func(ctx admission.TypeValidatorContext) {
		AssertSelectionRefIdentity(inlineSelection(ctx.Value, "NarrativeSelection"))
	}},
	{Type: Types.Moment, Handler: func(ctx admission.TypeValidatorContext) {
		AssertMomentRefIdentity(inlineMoment(ctx.Value, "NarrativeMoment"))
	}},
}}

// Package is the hypit.package@1 activation.
var Package struct {
	Format  string
	Modules []protocol.ResolvedModule
	Facets  []any
}

func init() {
	Package.Format = "hypit.package@1"
	Package.Modules = []protocol.ResolvedModule{{Manifest: Manifest}}
	Package.Facets = []any{
		producer.CreateProducerPackageFacet(producer.ProducerPackage{}),
		admission.CreateAdmissionPackageFacet(admission.AdmissionPackage{Validators: Component.Validators}),
	}
}

func mustInline(value protocol.StoredValue, subject string) any {
	if value.Kind != "inline" {
		panic(errors.New(subject + " must be inline."))
	}
	return value.Value
}

func inlineNarrative(value protocol.StoredValue, subject string) Narrative {
	raw := mustInline(value, subject)
	if narrative, ok := raw.(Narrative); ok {
		return narrative
	}
	panic(errors.New(subject + " must be inline."))
}

func inlineSegment(value protocol.StoredValue, subject string) SegmentRef {
	raw := mustInline(value, subject)
	if ref, ok := raw.(SegmentRef); ok {
		return ref
	}
	panic(errors.New(subject + " must be inline."))
}

func inlineSelection(value protocol.StoredValue, subject string) SelectionRef {
	raw := mustInline(value, subject)
	if ref, ok := raw.(SelectionRef); ok {
		return ref
	}
	panic(errors.New(subject + " must be inline."))
}

func inlineMoment(value protocol.StoredValue, subject string) MomentRef {
	raw := mustInline(value, subject)
	if ref, ok := raw.(MomentRef); ok {
		return ref
	}
	panic(errors.New(subject + " must be inline."))
}
