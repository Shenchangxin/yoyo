// Copyright 2026 Hypit.AI
// Translated from @hypit/timeline/src/component.ts and activation.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package timeline

import (
	"errors"

	"github.com/Shenchangxin/yoyo/remake/admission"
	"github.com/Shenchangxin/yoyo/remake/producer"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// Component is the producer and admission contribution of the timeline package.
var Component = struct {
	Producers  []producer.ProducerFacet
	Validators []admission.TypeValidatorFacet
}{
	Producers: []producer.ProducerFacet{},
	Validators: []admission.TypeValidatorFacet{{
		Type: Types.Timeline,
		Handler: func(ctx admission.TypeValidatorContext) {
			AssertTimelineIdentity(track(ctx.Value))
		},
	}},
}

// Package is the hypit.package@1 activation.
var Package = struct {
	Format  string
	Modules []protocol.ResolvedModule
	Facets  []any
}{
	Format:  "hypit.package@1",
	Modules: nil, // filled in init so Manifest is ready
}

func init() {
	Package.Modules = []protocol.ResolvedModule{{Manifest: Manifest}}
	producerFacet := producer.CreateProducerPackageFacet(producer.ProducerPackage{Producers: Component.Producers})
	admissionFacet := admission.CreateAdmissionPackageFacet(admission.AdmissionPackage{Validators: Component.Validators})
	Package.Facets = []any{producerFacet, admissionFacet}
}

func track(value protocol.StoredValue) Timeline {
	if value.Kind != "inline" {
		panic(errors.New("Timeline must be inline."))
	}
	if timeline, ok := value.Value.(Timeline); ok {
		return timeline
	}
	return timelineFromCanonical(value.Value)
}

func timelineFromCanonical(value any) Timeline {
	object, _ := value.(map[string]any)
	rate, _ := object["frameRate"].(map[string]any)
	return Timeline{
		ID:         stringField(object["id"]),
		FrameCount: intField(object["frameCount"]),
		FrameRate: FrameRate{
			Numerator:   intField(rate["numerator"]),
			Denominator: intField(rate["denominator"]),
		},
	}
}

func stringField(value any) string {
	text, _ := value.(string)
	return text
}

func intField(value any) int64 {
	switch n := value.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		if n != float64(int64(n)) {
			return 0
		}
		return int64(n)
	default:
		return 0
	}
}
