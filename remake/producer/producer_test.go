// Copyright 2026 Hypit.AI
// Translated from @hypit/producer/src/index.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package producer

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestPlannedNeedInputsKeepsMissingEdges(t *testing.T) {
	state := protocol.BuildState{
		Records: []protocol.TypedRecord{{ID: "ready"}},
		Plan: protocol.BuildPlan{Steps: []protocol.ProducerStep{
			{ID: "earlier", Outputs: map[string]string{"out": "later"}},
			{ID: "ask", Inputs: map[string]string{"prompt": "ready", "image": "later"}},
		}},
	}
	got := PlannedNeedInputs(state, "ask", map[string]string{"image": "image"})
	if len(got) != 1 || got[0].Input != "image" || got[0].Record != "later" || got[0].SourceStep == nil || *got[0].SourceStep != "earlier" {
		t.Fatalf("%+v", got)
	}
	if got[0].Role == nil || *got[0].Role != "image" {
		t.Fatal("role")
	}
	if len(PlannedNeedInputs(state, "missing", nil)) != 0 {
		t.Fatal("unknown step returned inputs")
	}
}

func TestProducerFacetRoundTrip(t *testing.T) {
	handler := func(ProducerHandlerContext) ProducerHandlerResult {
		return ProducerHandlerResult{}
	}
	ref := protocol.ProducerRef{Module: protocol.ModuleRef{Name: "example", Version: "1"}, Name: "make"}
	created := CreateProducerPackageFacet(ProducerPackage{Producers: []ProducerFacet{{Producer: ref, Handler: handler}}})
	if created.ABI != ProducerPackageFacetABI || created.Implementation.PlannedNeeds != nil {
		t.Fatalf("%+v", created)
	}
	blank := CreateProducerPackageFacet(nil)
	if blank.Implementation.Producers != nil || blank.ABI != ProducerPackageFacetABI {
		t.Fatal("non-object gained producers")
	}
	packages := ProducerPackagesFromFacets([]facet.Facet{
		{ABI: "other"},
		{ABI: ProducerPackageFacetABI, Implementation: created.Implementation},
	})
	if len(packages) != 1 || len(packages[0].Producers) != 1 {
		t.Fatalf("%+v", packages)
	}
	registry := &recordingRegistry{}
	RegisterProducerFacets(registry, packages[0].Producers)
	if len(registry.refs) != 1 || registry.refs[0] != ref {
		t.Fatal("handler was not registered")
	}
}

type recordingRegistry struct {
	refs []protocol.ProducerRef
}

func (r *recordingRegistry) RegisterProducer(ref protocol.ProducerRef, _ ProducerHandler) {
	r.refs = append(r.refs, ref)
}
