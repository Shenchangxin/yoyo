// Copyright 2026 Hypit.AI
// Translated from @hypit/producer/src/index.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package producer is the deterministic handler surface owned by a package.
// Artifact bytes, credentials, network, and stores stay behind an explicit Need.
package producer

import (
	"sort"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// ProducerPackageFacetABI is the facet ABI for producer-owned implementations.
const ProducerPackageFacetABI = "hypit.producer-package@1"

// ProducerHandlerResult is the outputs and need constraints a handler returns.
type ProducerHandlerResult struct {
	Outputs map[string]protocol.StoredValue
	Needs   map[string]any
}

// ProducerHandlerContext is the pure compute context of one invocation.
type ProducerHandlerContext struct {
	Command  protocol.InvokeProducerCommand
	Producer protocol.ProducerRef
	Inputs   map[string]protocol.TypedRecord
}

// ProducerHandler computes one producer result. It throws to reject the invocation.
type ProducerHandler func(ProducerHandlerContext) ProducerHandlerResult

// ProducerRegistrar accepts one handler per producer.
type ProducerRegistrar interface {
	RegisterProducer(protocol.ProducerRef, ProducerHandler)
}

// ProducerFacet is one enumerable handler owned by its package.
type ProducerFacet struct {
	Producer protocol.ProducerRef
	Handler  ProducerHandler
}

// PlannedNeedInput is one input whose value exists only after an upstream step.
// A nil SourceStep or Role means that field was absent.
type PlannedNeedInput struct {
	Input      string
	Record     string
	SourceStep *string
	Role       *string
}

// PlannedNeedSpecification is the support-relevant request known before external work starts.
type PlannedNeedSpecification struct {
	Constraints   any
	PendingInputs []PlannedNeedInput
}

// PlannedNeedPresentation is a package-owned reading of one request.
type PlannedNeedPresentation struct {
	Fields     map[string][]any
	References map[string]int
}

// PlannedNeedFacet is the planning knowledge for one external request.
type PlannedNeedFacet struct {
	Producer   protocol.ProducerRef
	Port       string
	Capability protocol.CapabilityRef
	Plan       func(state protocol.BuildState, step, port string) (PlannedNeedSpecification, bool)
	Present    func(PlannedNeedSpecification) PlannedNeedPresentation
}

// ProducerPackage holds the handlers and planning knowledge a package contributes.
// A nil slice means the field was absent.
type ProducerPackage struct {
	Producers    []ProducerFacet
	PlannedNeeds []PlannedNeedFacet
}

// ProducerPackageFacet is a facet whose ABI is ProducerPackageFacetABI.
type ProducerPackageFacet struct {
	ABI            string
	Implementation ProducerPackage
}

// PlannedNeedInputs lists missing direct inputs of one planned Need.
func PlannedNeedInputs(state protocol.BuildState, stepID string, roles map[string]string) []PlannedNeedInput {
	var step protocol.ProducerStep
	found := false
	for _, item := range state.Plan.Steps {
		if item.ID == stepID {
			step = item
			found = true
			break
		}
	}
	if !found {
		return []PlannedNeedInput{}
	}
	available := map[string]struct{}{}
	for _, record := range state.Records {
		available[record.ID] = struct{}{}
	}
	producedBy := map[string]string{}
	for _, item := range state.Plan.Steps {
		for _, record := range outputRecords(item.Outputs) {
			producedBy[record] = item.ID
		}
	}
	names := make([]string, 0, len(step.Inputs))
	for name := range step.Inputs {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return protocol.LocaleCompare(names[i], names[j]) < 0
	})
	var pending []PlannedNeedInput
	for _, input := range names {
		record := step.Inputs[input]
		if _, ok := available[record]; ok {
			continue
		}
		item := PlannedNeedInput{Input: input, Record: record}
		if source, ok := producedBy[record]; ok {
			item.SourceStep = &source
		}
		if role, ok := roles[input]; ok {
			item.Role = &role
		}
		pending = append(pending, item)
	}
	if pending == nil {
		return []PlannedNeedInput{}
	}
	return pending
}

// RegisterProducerFacets registers each facet on the registry.
func RegisterProducerFacets(registry ProducerRegistrar, facets []ProducerFacet) {
	for _, item := range facets {
		registry.RegisterProducer(item.Producer, item.Handler)
	}
}

// CreateProducerPackageFacet keeps producers and planned needs only when the value carries them.
func CreateProducerPackageFacet(value any) ProducerPackageFacet {
	item, ok := value.(ProducerPackage)
	if !ok {
		return ProducerPackageFacet{ABI: ProducerPackageFacetABI}
	}
	return ProducerPackageFacet{ABI: ProducerPackageFacetABI, Implementation: ProducerPackage{
		Producers: item.Producers, PlannedNeeds: item.PlannedNeeds,
	}}
}

// ProducerPackagesFromFacets returns implementations whose ABI is the producer package ABI.
func ProducerPackagesFromFacets(facets []facet.Facet) []ProducerPackage {
	var out []ProducerPackage
	for _, item := range facets {
		if item.ABI != ProducerPackageFacetABI {
			continue
		}
		impl, ok := item.Implementation.(ProducerPackage)
		if !ok {
			continue
		}
		out = append(out, impl)
	}
	return out
}

func outputRecords(outputs map[string]string) []string {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return protocol.LocaleCompare(names[i], names[j]) < 0
	})
	records := make([]string, len(names))
	for i, name := range names {
		records[i] = outputs[name]
	}
	return records
}
