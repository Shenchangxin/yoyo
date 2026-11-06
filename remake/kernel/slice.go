// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/slice.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"errors"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// SlicedExecution is the program, plan, and records reached by one run's targets.
type SlicedExecution struct {
	Program        protocol.LinkedProgram
	Plan           protocol.BuildPlan
	InitialRecords []protocol.TypedRecord
	Targets        []protocol.BuildTarget
	Selections     []BuildCandidateSelection
}

func sliceModuleKey(ref protocol.ModuleRef) string {
	return ref.Name + "\x00" + ref.Version
}

// SliceExecution keeps the candidate selected for each targeted output and the modules that candidate reaches.
func SliceExecution(program protocol.LinkedProgram, authorGraph protocol.CompiledGraph, runGraph protocol.RunGraph) SlicedExecution {
	preliminary := PlanBuild(program, authorGraph, runGraph)
	graph := protocol.CompiledGraph{
		Format:     "hypit.graph@1",
		Outputs:    authorGraph.Outputs,
		Candidates: append(append([]protocol.Candidate{}, authorGraph.Candidates...), runGraph.Candidates...),
		Operations: append(append([]protocol.OperationNode{}, authorGraph.Operations...), runGraph.Operations...),
	}
	outputByID := map[string]protocol.LogicalOutput{}
	for _, item := range authorGraph.Outputs {
		outputByID[item.ID] = item
	}
	candidateByID := map[string]protocol.Candidate{}
	for _, item := range graph.Candidates {
		candidateByID[item.ID] = item
	}
	operationByID := map[string]protocol.OperationNode{}
	for _, item := range graph.Operations {
		operationByID[item.ID] = item
	}
	selectionByOutput := map[string]BuildCandidateSelection{}
	for _, item := range preliminary.Selections {
		selectionByOutput[item.Output] = item
	}
	outputs := map[string]struct{}{}
	candidates := map[string]struct{}{}
	operations := map[string]struct{}{}
	records := map[string]struct{}{}

	var visitRef func(protocol.GraphValueRef)
	var visitOperation func(string)
	var visitCandidate func(string)
	var visitOutput func(string)
	visitRef = func(ref protocol.GraphValueRef) {
		if ref.Kind == "record" {
			records[ref.ID] = struct{}{}
		} else if ref.Kind == "logical-output" {
			visitOutput(ref.ID)
		} else {
			visitOperation(ref.Operation)
		}
	}
	visitOperation = func(id string) {
		if _, seen := operations[id]; seen {
			return
		}
		operation, ok := operationByID[id]
		if !ok {
			panic(errors.New("execution slice refers to absent Operation " + id))
		}
		operations[id] = struct{}{}
		for _, ref := range inputRefs(operation.Inputs) {
			visitRef(ref)
		}
	}
	visitCandidate = func(id string) {
		if _, seen := candidates[id]; seen {
			return
		}
		candidate, ok := candidateByID[id]
		if !ok {
			panic(errors.New("execution slice refers to absent Candidate " + id))
		}
		candidates[id] = struct{}{}
		if candidate.Root.Kind == "operation" {
			visitOperation(candidate.Root.Result.Operation)
		}
	}
	visitOutput = func(id string) {
		if _, seen := outputs[id]; seen {
			return
		}
		output, ok := outputByID[id]
		if !ok {
			panic(errors.New("execution slice refers to absent Logical Output " + id))
		}
		outputs[id] = struct{}{}
		selection, selected := selectionByOutput[id]
		chosen := output.Primary
		if selected {
			chosen = selection.Candidate
		}
		visitCandidate(chosen)
	}
	for _, target := range runGraph.Targets {
		visitOutput(target.Output)
	}

	var selectedOperations []protocol.OperationNode
	for _, item := range graph.Operations {
		if _, ok := operations[item.ID]; ok {
			selectedOperations = append(selectedOperations, item)
		}
	}
	var authored []protocol.TypedRecord
	for _, record := range program.Records {
		if _, ok := records[record.ID]; ok {
			authored = append(authored, record)
		}
	}
	required := map[string]struct{}{}
	modulesByKey := map[string]protocol.ResolvedModule{}
	for _, item := range program.Closure.Modules {
		modulesByKey[sliceModuleKey(protocol.ModuleRef{Name: item.Manifest.Name, Version: item.Manifest.Version})] = item
	}
	var requireModule func(protocol.ModuleRef)
	requireModule = func(ref protocol.ModuleRef) {
		key := sliceModuleKey(ref)
		if _, seen := required[key]; seen {
			return
		}
		resolved, ok := modulesByKey[key]
		if !ok {
			panic(errors.New("execution slice refers to absent module " + ref.Name + "@" + ref.Version))
		}
		required[key] = struct{}{}
		for _, item := range resolved.Manifest.Dependencies {
			requireModule(item.Module)
		}
	}
	for _, item := range selectedOperations {
		requireModule(item.Producer.Module)
	}
	for _, item := range preliminary.Plan.OutputBindings {
		requireModule(item.Type.Module)
	}
	for _, item := range preliminary.InitialRecords {
		requireModule(item.Type.Module)
	}
	for _, item := range authored {
		requireModule(item.Type.Module)
	}
	var manifests []protocol.ModuleManifest
	for _, item := range program.Closure.Modules {
		if _, ok := required[sliceModuleKey(protocol.ModuleRef{Name: item.Manifest.Name, Version: item.Manifest.Version})]; ok {
			manifests = append(manifests, item.Manifest)
		}
	}
	closure := CreateResolvedClosure(manifests)
	return SlicedExecution{
		Program:        Link(closure, authored),
		Plan:           preliminary.Plan,
		InitialRecords: preliminary.InitialRecords,
		Targets:        runGraph.Targets,
		Selections:     preliminary.Selections,
	}
}
