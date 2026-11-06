// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package author

import (
	"regexp"
	"testing"

	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// Upstream fragment.test.ts builds this closure from videoContractManifests plus
// @hypit/narrative-temporal, @hypit/composition, and @hypit/spatial. Those packages
// are outside this module's allowed imports, so the fixture keeps the module and
// type identities the test actually resolves.

var narrativeTemporalRef = protocol.ModuleRef{Name: "@hypit/narrative-temporal", Version: "1"}
var compositionRef = protocol.ModuleRef{Name: "@hypit/composition", Version: "1"}
var spatialRef = protocol.ModuleRef{Name: "@hypit/spatial", Version: "1"}
var narrativeAlignment = protocol.TypeRef{Module: narrativeTemporalRef, Name: "NarrativeAlignment"}
var visualTrack = protocol.TypeRef{Module: compositionRef, Name: "VisualTrack"}
var audioTrack = protocol.TypeRef{Module: compositionRef, Name: "AudioTrack"}
var canvasType = protocol.TypeRef{Module: spatialRef, Name: "Canvas"}

var testModule = protocol.ModuleRef{Name: "example.fragment-speech", Version: "0.0.0"}
var requestType = protocol.TypeRef{Module: testModule, Name: "Request"}
var generateProducer = protocol.ProducerRef{Module: testModule, Name: "generate"}
var projectVisualProducer = protocol.ProducerRef{Module: testModule, Name: "project-visual"}
var projectAudioProducer = protocol.ProducerRef{Module: testModule, Name: "project-audio"}

func speechManifests() []protocol.ModuleManifest {
	return []protocol.ModuleManifest{
		{
			Format: "hypit.module@1", Name: narrativeTemporalRef.Name, Version: narrativeTemporalRef.Version,
			Dependencies: []protocol.ModuleDependency{},
			Types:        []protocol.TypeDeclaration{{Name: narrativeAlignment.Name}},
			Capabilities: []protocol.CapabilityDeclaration{},
			Producers:    []protocol.ProducerDeclaration{},
		},
		{
			Format: "hypit.module@1", Name: compositionRef.Name, Version: compositionRef.Version,
			Dependencies: []protocol.ModuleDependency{},
			Types: []protocol.TypeDeclaration{
				{Name: visualTrack.Name},
				{Name: audioTrack.Name},
			},
			Capabilities: []protocol.CapabilityDeclaration{},
			Producers:    []protocol.ProducerDeclaration{},
		},
		{
			Format: "hypit.module@1", Name: spatialRef.Name, Version: spatialRef.Version,
			Dependencies: []protocol.ModuleDependency{},
			Types:        []protocol.TypeDeclaration{{Name: canvasType.Name}},
			Capabilities: []protocol.CapabilityDeclaration{},
			Producers:    []protocol.ProducerDeclaration{},
		},
		{
			Format:  "hypit.module@1",
			Name:    testModule.Name,
			Version: testModule.Version,
			Dependencies: []protocol.ModuleDependency{
				{Module: narrativeTemporalRef},
				{Module: compositionRef},
			},
			Types:        []protocol.TypeDeclaration{{Name: requestType.Name}},
			Capabilities: []protocol.CapabilityDeclaration{},
			Producers: []protocol.ProducerDeclaration{{
				Name: generateProducer.Name,
				Inputs: []protocol.PortDeclaration{
					{Name: "request", Type: requestType},
					{Name: "style", Type: requestType},
				},
				Outputs: []protocol.PortDeclaration{{Name: "take", Type: narrativeAlignment}},
				Needs:   []protocol.NeedPortDeclaration{},
			}, {
				Name:    projectVisualProducer.Name,
				Inputs:  []protocol.PortDeclaration{{Name: "take", Type: narrativeAlignment}},
				Outputs: []protocol.PortDeclaration{{Name: "visual", Type: visualTrack}},
				Needs:   []protocol.NeedPortDeclaration{},
			}, {
				Name:    projectAudioProducer.Name,
				Inputs:  []protocol.PortDeclaration{{Name: "take", Type: narrativeAlignment}},
				Outputs: []protocol.PortDeclaration{{Name: "audio", Type: audioTrack}},
				Needs:   []protocol.NeedPortDeclaration{},
			}},
		},
	}
}

func speechProgram() protocol.LinkedProgram {
	closure := kernel.CreateResolvedClosure(speechManifests())
	return kernel.Link(closure, []protocol.TypedRecord{
		kernel.SealRecord(protocol.TypedRecord{
			ID:    "request:root",
			Type:  requestType,
			Value: protocol.StoredValue{Kind: "inline", Value: "Say hello."},
		}),
		kernel.SealRecord(protocol.TypedRecord{
			ID:    "style:root",
			Type:  requestType,
			Value: protocol.StoredValue{Kind: "inline", Value: "Direct to camera."},
		}),
		kernel.SealRecord(protocol.TypedRecord{
			ID:   "canvas:root",
			Type: canvasType,
			Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{
				"widthPx": float64(1080), "heightPx": float64(1920),
			}},
		}),
	})
}

func speechFragment() GraphFragment {
	input := func(name string) FragmentValueRef {
		return FragmentValueRef{Kind: "fragment-input", Name: name}
	}
	operation := func(id string) FragmentValueRef {
		return FragmentValueRef{Kind: "fragment-operation", Operation: id}
	}
	return SealGraphFragment(GraphFragment{
		Inputs: []FragmentInput{
			{Name: "request", Type: requestType},
			{Name: "style", Type: requestType},
			{Name: "canvas", Type: canvasType},
		},
		Operations: []FragmentOperation{
			{
				ID:       "generate",
				Producer: generateProducer,
				Inputs:   map[string]FragmentValueRef{"request": input("request"), "style": input("style")},
				Result:   FragmentOperationResult{Kind: "output", Name: "take"},
			},
			{
				ID:       "visual",
				Producer: projectVisualProducer,
				Inputs:   map[string]FragmentValueRef{"take": operation("generate")},
				Result:   FragmentOperationResult{Kind: "output", Name: "visual"},
			},
			{
				ID:       "audio",
				Producer: projectAudioProducer,
				Inputs:   map[string]FragmentValueRef{"take": operation("generate")},
				Result:   FragmentOperationResult{Kind: "output", Name: "audio"},
			},
		},
		Exports: []FragmentExport{
			{Name: "take", Type: narrativeAlignment, Root: operation("generate")},
			{Name: "visual", Type: visualTrack, Root: operation("visual")},
			{Name: "audio", Type: audioTrack, Root: operation("audio")},
		},
	})
}

func speechInstance(programValue protocol.LinkedProgram, id string) ElaboratedFragment {
	fragment := speechFragment()
	return ElaborateGraphFragment(programValue, fragment, FragmentInstanceRequest{
		ID:       id,
		Fragment: fragment.ID,
		Inputs: map[string]protocol.GraphValueRef{
			"request": {Kind: "record", ID: "request:root"},
			"style":   {Kind: "record", ID: "style:root"},
			"canvas":  {Kind: "record", ID: "canvas:root"},
		},
	})
}

func compiledGraph(contributions ...FragmentContribution) protocol.CompiledGraph {
	merged := MergeFragmentContributions(protocol.CompiledGraph{
		Outputs:    []protocol.LogicalOutput{},
		Candidates: []protocol.Candidate{},
		Operations: []protocol.OperationNode{},
	}, contributions...)
	return kernel.SealCompiledGraph(merged)
}

func TestOneFragmentInstanceSharesItsGenerationOperation(t *testing.T) {
	linked := speechProgram()
	elaborated := speechInstance(linked, "opening")
	contribution := BindAuthorFragment(elaborated, map[string]string{
		"take":   "opening.take",
		"audio":  "opening.audio",
		"visual": "opening.visual",
	})
	compiled := compiledGraph(contribution)
	request := kernel.SealBuildRequest(protocol.BuildRequest{
		Targets: []protocol.BuildTarget{
			{Output: "opening.audio"},
			{Output: "opening.visual"},
		},
	})
	state := kernel.Start(linked, compiled, request)
	generations := 0
	for _, step := range state.Plan.Steps {
		if step.Producer.Name == generateProducer.Name {
			generations++
		}
	}
	if generations != 1 {
		t.Fatalf("generate steps %d", generations)
	}
	if len(state.Plan.Steps) != 3 {
		t.Fatalf("steps %d", len(state.Plan.Steps))
	}
	takeRecord := ""
	foundTake := false
	for _, item := range contribution.Operations {
		if item.Producer.Name == "generate" {
			takeRecord = item.Result.Record
			foundTake = true
			break
		}
	}
	if !foundTake {
		t.Fatal("missing generate operation")
	}
	var audioStep, visualStep protocol.ProducerStep
	foundAudio, foundVisual := false, false
	for _, step := range state.Plan.Steps {
		if step.Producer.Name == "project-audio" {
			audioStep = step
			foundAudio = true
		}
		if step.Producer.Name == "project-visual" {
			visualStep = step
			foundVisual = true
		}
	}
	if !foundAudio || !foundVisual {
		t.Fatal("missing projection steps")
	}
	if audioStep.Inputs["take"] != takeRecord {
		t.Fatalf("audio %s != %s", audioStep.Inputs["take"], takeRecord)
	}
	if visualStep.Inputs["take"] != takeRecord {
		t.Fatalf("visual %s != %s", visualStep.Inputs["take"], takeRecord)
	}
}

func TestDistinctFragmentInstancesNeverContentDedupe(t *testing.T) {
	linked := speechProgram()
	opening := speechInstance(linked, "opening")
	closing := speechInstance(linked, "closing")
	if opening.ID == closing.ID {
		t.Fatal("instances were deduplicated")
	}
	ids := map[string]struct{}{}
	for _, operation := range opening.Operations {
		ids[operation.ID] = struct{}{}
	}
	for _, operation := range closing.Operations {
		ids[operation.ID] = struct{}{}
	}
	if len(ids) != 6 {
		t.Fatalf("operation ids %d", len(ids))
	}

	compiled := compiledGraph(
		BindAuthorFragment(opening, map[string]string{
			"take":   "opening.take",
			"audio":  "opening.audio",
			"visual": "opening.visual",
		}),
		BindAuthorFragment(closing, map[string]string{
			"take":   "closing.take",
			"audio":  "closing.audio",
			"visual": "closing.visual",
		}),
	)
	state := kernel.Start(linked, compiled, kernel.SealBuildRequest(protocol.BuildRequest{
		Targets: []protocol.BuildTarget{
			{Output: "opening.audio"},
			{Output: "closing.audio"},
		},
	}))
	generations := 0
	for _, step := range state.Plan.Steps {
		if step.Producer.Name == generateProducer.Name {
			generations++
		}
	}
	if generations != 2 {
		t.Fatalf("generate steps %d", generations)
	}
}

func TestFragmentReferencesCannotEscapeThroughARawGraphReference(t *testing.T) {
	valid := speechFragment()
	operations := make([]FragmentOperation, len(valid.Operations))
	for i, item := range valid.Operations {
		if item.ID == "generate" {
			inputs := make(map[string]FragmentValueRef, len(item.Inputs))
			for name, ref := range item.Inputs {
				inputs[name] = ref
			}
			// Upstream writes {kind:"record", id:"secret:ambient"}. The kind check throws before id is read.
			inputs["request"] = FragmentValueRef{Kind: "record"}
			item.Inputs = inputs
		}
		operations[i] = item
	}
	err := recoverError(t, func() {
		SealGraphFragment(GraphFragment{
			Inputs:     valid.Inputs,
			Operations: operations,
			Exports:    valid.Exports,
		})
	})
	if !regexp.MustCompile(`must name a declared input or local Operation`).MatchString(err.Error()) {
		t.Fatal(err)
	}
}
