// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/plan.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"fmt"
	"slices"
	"sort"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// PlannedNeed is one external request of a finite build plan.
type PlannedNeed struct {
	Step       protocol.StepId
	Port       string
	Need       protocol.NeedId
	Result     protocol.RecordId
	Capability protocol.CapabilityRef
	Returns    protocol.TypeRef
}

// BuildCandidateSelection records which candidate supplied one logical output.
type BuildCandidateSelection struct {
	Output    string
	Candidate string
	Record    string
}

// PlannedExecution is the plan, the candidate selections, and the records the run must admit first.
type PlannedExecution struct {
	Plan           protocol.BuildPlan
	Selections     []BuildCandidateSelection
	InitialRecords []protocol.TypedRecord
}

type resolvedSource struct {
	record protocol.RecordId
	typ    protocol.TypeRef
}

type producedRecord struct {
	typ  protocol.TypeRef
	step string
	has  bool
}

type demandedOp struct {
	operation protocol.OperationNode
	inputs    map[string]protocol.RecordId
}

func exactStepKeys(actual []string, expected []string, subject string) {
	left := append([]string(nil), actual...)
	right := append([]string(nil), expected...)
	sortUTF16(left)
	sortUTF16(right)
	if slices.Equal(left, right) {
		return
	}
	invariant(false, "PORT_BINDING_MISMATCH", fmt.Sprintf("%s binds [%s] but declares [%s]", subject, joinComma(left), joinComma(right)), subject)
}

func subsetStepKeys(actual []string, declared []string, subject string) {
	allowed := map[string]struct{}{}
	for _, name := range declared {
		allowed[name] = struct{}{}
	}
	names := append([]string(nil), actual...)
	sort.SliceStable(names, func(i, j int) bool {
		return protocol.LocaleCompare(names[i], names[j]) < 0
	})
	for _, name := range names {
		_, ok := allowed[name]
		invariant(ok, "PORT_BINDING_MISMATCH", subject+" binds undeclared port "+name, subject)
	}
}

func joinComma(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out
}

func mapKeys[T any](values map[string]T) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	return names
}

// PlannedNeeds lists every external requirement of the plan, in step order.
func PlannedNeeds(state protocol.BuildState) []PlannedNeed {
	found := []PlannedNeed{}
	for _, step := range state.Plan.Steps {
		producer := ResolveProducer(state.Program.Closure, step.Producer)
		for _, need := range producer.Needs {
			binding, ok := step.Needs[need.Name]
			if !ok {
				continue
			}
			found = append(found, PlannedNeed{
				Step:       step.ID,
				Port:       need.Name,
				Need:       binding.ID,
				Result:     binding.Result,
				Capability: need.Capability,
				Returns:    need.Returns,
			})
		}
	}
	return append([]PlannedNeed(nil), found...)
}

func planContent(steps []protocol.ProducerStep, goals []protocol.BuildGoal, bindings []protocol.BuildOutputBinding) protocol.BuildPlan {
	return protocol.BuildPlan{Format: "hypit.plan@1", Steps: steps, Goals: goals, OutputBindings: bindings}
}

func compilePlanning(
	program protocol.LinkedProgram,
	graph protocol.CompiledGraph,
	request protocol.BuildRequest,
	satisfactions []protocol.Satisfaction,
	runRecords []protocol.TypedRecord,
) PlannedExecution {
	verification := protocol.LinkedProgram{
		Closure: program.Closure,
		Records: append(append([]protocol.TypedRecord{}, program.Records...), runRecords...),
	}
	VerifyBuildRequest(verification, graph, request)

	selectedCandidates := map[string]string{}
	for _, satisfaction := range satisfactions {
		output := ResolveLogicalOutput(graph, satisfaction.Output)
		candidate := ResolveCandidate(graph, satisfaction.Candidate)
		_, duplicate := selectedCandidates[output.ID]
		invariant(!duplicate, "DUPLICATE_RUN_SATISFACTION", output.ID+" is satisfied more than once", output.ID)
		invariant(sameType(candidate.Type, output.Type), "CANDIDATE_RESULT_TYPE_MISMATCH", fmt.Sprintf("%s supplies %s, not %s", candidate.ID, typeKey(candidate.Type), typeKey(output.Type)), candidate.ID)
		selectedCandidates[output.ID] = candidate.ID
	}

	authored := map[string]protocol.TypedRecord{}
	for _, record := range program.Records {
		authored[record.ID] = record
	}
	availableRun := map[string]protocol.TypedRecord{}
	for _, record := range runRecords {
		_, conflict := authored[record.ID]
		invariant(!conflict, "RUN_RECORD_CONFLICT", record.ID+" conflicts with authored input", record.ID)
		_, duplicate := availableRun[record.ID]
		invariant(!duplicate, "DUPLICATE_RUN_RECORD", record.ID+" is declared twice", record.ID)
		availableRun[record.ID] = record
		authored[record.ID] = record
	}
	demandedRun := map[string]struct{}{}
	demandedOperations := map[string]protocol.OperationNode{}
	resolvedOutputs := map[string]resolvedSource{}
	selections := map[string]BuildCandidateSelection{}

	operationSource := func(id string) resolvedSource {
		operation := ResolveOperation(graph, id)
		producer := ResolveProducer(program.Closure, operation.Producer)
		var typ protocol.TypeRef
		found := false
		if operation.Result.Kind == "output" {
			for _, port := range producer.Outputs {
				if port.Name == operation.Result.Name {
					typ = port.Type
					found = true
					break
				}
			}
		} else {
			for _, port := range producer.Needs {
				if port.Name == operation.Result.Name {
					typ = port.Returns
					found = true
					break
				}
			}
		}
		invariant(found, "OPERATION_RESULT_MISMATCH", id+" result is not declared", id)
		return resolvedSource{record: OperationResultRecord(operation), typ: typ}
	}

	pending := make([]protocol.GraphValueRef, len(request.Targets))
	for i, target := range request.Targets {
		pending[i] = protocol.GraphValueRef{Kind: "logical-output", ID: target.Output}
	}
	for len(pending) > 0 {
		ref := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if ref.Kind == "record" {
			_, ok := authored[ref.ID]
			invariant(ok, "UNKNOWN_RECORD", "unknown authored record "+ref.ID, ref.ID)
			if _, run := availableRun[ref.ID]; run {
				demandedRun[ref.ID] = struct{}{}
			}
			continue
		}
		if ref.Kind == "operation-result" {
			if _, seen := demandedOperations[ref.Operation]; seen {
				continue
			}
			operation := ResolveOperation(graph, ref.Operation)
			demandedOperations[operation.ID] = operation
			pending = append(pending, inputRefs(operation.Inputs)...)
			continue
		}
		if _, seen := resolvedOutputs[ref.ID]; seen {
			continue
		}
		output := ResolveLogicalOutput(graph, ref.ID)
		candidateID, selected := selectedCandidates[ref.ID]
		if !selected {
			candidateID = output.Primary
		}
		candidate := ResolveCandidate(graph, candidateID)
		var resolved resolvedSource
		if candidate.Root.Kind == "value" {
			record := protocol.TypedRecord{ID: candidate.Root.Value.ID, Type: candidate.Type, Value: candidate.Root.Value.Value}
			_, conflict := authored[record.ID]
			invariant(!conflict, "PROVIDED_RECORD_CONFLICT", record.ID+" conflicts with authored input")
			resolved = resolvedSource{record: record.ID, typ: record.Type}
		} else {
			resolved = operationSource(candidate.Root.Result.Operation)
			pending = append(pending, protocol.GraphValueRef{Kind: candidate.Root.Result.Kind, Operation: candidate.Root.Result.Operation})
		}
		invariant(sameType(resolved.typ, output.Type), "CANDIDATE_RESULT_TYPE_MISMATCH", fmt.Sprintf("%s returns %s, not %s", candidate.ID, typeKey(resolved.typ), typeKey(output.Type)), candidate.ID)
		resolvedOutputs[output.ID] = resolved
		selections[output.ID] = BuildCandidateSelection{Output: output.ID, Candidate: candidate.ID, Record: resolved.record}
	}

	resolveRef := func(ref protocol.GraphValueRef) resolvedSource {
		if ref.Kind == "record" {
			record, ok := authored[ref.ID]
			invariant(ok, "UNKNOWN_RECORD", "unknown authored record "+ref.ID, ref.ID)
			return resolvedSource{record: record.ID, typ: record.Type}
		}
		if ref.Kind == "logical-output" {
			resolved, ok := resolvedOutputs[ref.ID]
			invariant(ok, "UNKNOWN_LOGICAL_OUTPUT", "unresolved logical output "+ref.ID, ref.ID)
			return resolved
		}
		_, demanded := demandedOperations[ref.Operation]
		invariant(demanded, "UNKNOWN_OPERATION", "undemanded Operation "+ref.Operation, ref.Operation)
		return operationSource(ref.Operation)
	}

	demanded := make([]demandedOp, 0, len(demandedOperations))
	for _, operation := range demandedOperations {
		inputs := map[string]protocol.RecordId{}
		for name, ref := range operation.Inputs {
			inputs[name] = resolveRef(ref).record
		}
		demanded = append(demanded, demandedOp{operation: operation, inputs: inputs})
	}
	sort.SliceStable(demanded, func(i, j int) bool {
		return protocol.LocaleCompare(demanded[i].operation.ID, demanded[j].operation.ID) < 0
	})
	steps := make([]protocol.ProducerStep, len(demanded))
	for i, item := range demanded {
		operation := item.operation
		step := protocol.ProducerStep{
			ID:       operation.ID,
			Producer: operation.Producer,
			Inputs:   item.inputs,
			Outputs:  map[string]protocol.RecordId{},
			Needs:    map[string]protocol.NeedBinding{},
		}
		if operation.Result.Kind == "output" {
			step.Outputs[operation.Result.Name] = operation.Result.Record
		}
		if operation.Result.Kind == "need" {
			step.Needs[operation.Result.Name] = protocol.NeedBinding{ID: operation.Result.ID, Result: operation.Result.Record}
		}
		steps[i] = step
	}

	goals := make([]protocol.BuildGoal, len(request.Targets))
	for i, target := range request.Targets {
		source := resolvedOutputs[target.Output]
		goals[i] = protocol.BuildGoal{Record: source.record, Type: source.typ}
	}
	byLocale(goals, func(item protocol.BuildGoal) string { return item.Record })
	sortedSelections := make([]BuildCandidateSelection, 0, len(selections))
	for _, item := range selections {
		sortedSelections = append(sortedSelections, item)
	}
	byLocale(sortedSelections, func(item BuildCandidateSelection) string { return item.Output })
	bindings := make([]protocol.BuildOutputBinding, len(sortedSelections))
	for i, selection := range sortedSelections {
		bindings[i] = protocol.BuildOutputBinding{
			Output: selection.Output,
			Record: selection.Record,
			Type:   ResolveLogicalOutput(graph, selection.Output).Type,
		}
	}
	initialByID := map[string]protocol.TypedRecord{}
	for _, record := range SelectedProvidedRecords(program, graph, sortedSelections) {
		initialByID[record.ID] = record
	}
	for id := range demandedRun {
		_, conflict := initialByID[id]
		invariant(!conflict, "RUN_RECORD_CONFLICT", id+" has multiple sources", id)
		initialByID[id] = availableRun[id]
	}
	initial := make([]protocol.TypedRecord, 0, len(initialByID))
	for _, record := range initialByID {
		initial = append(initial, record)
	}
	byLocale(initial, func(item protocol.TypedRecord) string { return item.ID })
	plan := planContent(steps, goals, bindings)
	verifyBuildPlan(program, initial, plan)
	return PlannedExecution{Plan: plan, Selections: sortedSelections, InitialRecords: initial}
}

// PlanBuild plans one author graph together with one run graph.
func PlanBuild(program protocol.LinkedProgram, author protocol.CompiledGraph, run protocol.RunGraph) PlannedExecution {
	invariant(run.Format == "hypit.run-graph@1", "UNSUPPORTED_RUN_GRAPH", "unsupported Run Graph")
	graph := protocol.CompiledGraph{
		Format:     "hypit.graph@1",
		Outputs:    author.Outputs,
		Candidates: append(append([]protocol.Candidate{}, author.Candidates...), run.Candidates...),
		Operations: append(append([]protocol.OperationNode{}, author.Operations...), run.Operations...),
	}
	request := protocol.BuildRequest{Format: "hypit.build-request@1", Targets: run.Targets}
	return compilePlanning(program, graph, request, run.Satisfactions, run.Records)
}

// CompileBuild plans an author graph without a separate run graph.
func CompileBuild(program protocol.LinkedProgram, graph protocol.CompiledGraph, request protocol.BuildRequest) protocol.BuildPlan {
	return compilePlanning(program, graph, request, nil, nil).Plan
}

func verifyBuildPlan(program protocol.LinkedProgram, initialRecords []protocol.TypedRecord, plan protocol.BuildPlan) {
	invariant(plan.Format == "hypit.plan@1", "UNSUPPORTED_PLAN", "unsupported build plan format")
	records := map[protocol.RecordId]producedRecord{}
	for _, record := range program.Records {
		records[record.ID] = producedRecord{typ: record.Type, has: true}
	}
	for _, record := range initialRecords {
		_, duplicate := records[record.ID]
		invariant(!duplicate, "DUPLICATE_RECORD", "record "+record.ID+" has multiple sources", record.ID)
		records[record.ID] = producedRecord{typ: record.Type, has: true}
	}
	stepIDs := map[string]struct{}{}
	needIDs := map[string]struct{}{}
	for _, step := range plan.Steps {
		_, duplicate := stepIDs[step.ID]
		invariant(!duplicate, "DUPLICATE_STEP", "duplicate step "+step.ID, step.ID)
		stepIDs[step.ID] = struct{}{}
		producer := ResolveProducer(program.Closure, step.Producer)
		exactStepKeys(mapKeys(step.Inputs), portNames(producer.Inputs), step.ID+".inputs")
		subsetStepKeys(mapKeys(step.Outputs), portNames(producer.Outputs), step.ID+".outputs")
		subsetStepKeys(mapKeys(step.Needs), needNames(producer.Needs), step.ID+".needs")
		invariant(len(step.Outputs)+len(step.Needs) == 1, "STEP_RESULT_NORMAL_FORM", step.ID+" must expose one atomic result", step.ID)
		for _, port := range producer.Outputs {
			id, ok := step.Outputs[port.Name]
			if !ok {
				continue
			}
			_, duplicateRecord := records[id]
			invariant(!duplicateRecord, "DUPLICATE_RECORD", "record "+id+" has multiple producers", id)
			records[id] = producedRecord{typ: port.Type, step: step.ID, has: true}
		}
		for _, port := range producer.Needs {
			binding, ok := step.Needs[port.Name]
			if !ok {
				continue
			}
			_, duplicateNeed := needIDs[binding.ID]
			invariant(!duplicateNeed, "DUPLICATE_NEED", "need "+binding.ID+" has multiple producers")
			needIDs[binding.ID] = struct{}{}
			_, duplicateRecord := records[binding.Result]
			invariant(!duplicateRecord, "DUPLICATE_RECORD", binding.Result+" has multiple producers")
			records[binding.Result] = producedRecord{typ: port.Returns, step: step.ID, has: true}
		}
	}
	for _, step := range plan.Steps {
		producer := ResolveProducer(program.Closure, step.Producer)
		for _, port := range producer.Inputs {
			id := step.Inputs[port.Name]
			supplied, ok := records[id]
			invariant(ok, "UNKNOWN_RECORD", fmt.Sprintf("%s.%s references %s", step.ID, port.Name, id), id)
			invariant(sameType(supplied.typ, port.Type), "INPUT_TYPE_MISMATCH", fmt.Sprintf("%s.%s wants %s but %s is %s", step.ID, port.Name, typeKey(port.Type), id, typeKey(supplied.typ)), id)
		}
	}
	outputIDs := map[string]struct{}{}
	for _, binding := range plan.OutputBindings {
		_, duplicate := outputIDs[binding.Output]
		invariant(!duplicate, "DUPLICATE_OUTPUT_BINDING", binding.Output+" is bound more than once")
		outputIDs[binding.Output] = struct{}{}
		record, ok := records[binding.Record]
		invariant(ok, "UNKNOWN_OUTPUT_RECORD", binding.Output+" binds "+binding.Record)
		invariant(sameType(record.typ, binding.Type), "OUTPUT_BINDING_TYPE_MISMATCH", binding.Output)
	}
	for _, goal := range plan.Goals {
		supplied, ok := records[goal.Record]
		invariant(ok, "UNKNOWN_GOAL", "goal references "+goal.Record, goal.Record)
		invariant(sameType(supplied.typ, goal.Type), "GOAL_TYPE_MISMATCH", goal.Record)
	}
	assertAcyclic(plan, records)
	assertAllStepsReachGoal(plan, records)
}

// SelectedProvidedRecords materializes the literal values selected by the run.
func SelectedProvidedRecords(program protocol.LinkedProgram, graph protocol.CompiledGraph, selections []BuildCandidateSelection) []protocol.TypedRecord {
	authored := map[string]struct{}{}
	for _, record := range program.Records {
		authored[record.ID] = struct{}{}
	}
	records := map[string]protocol.TypedRecord{}
	candidates := map[string]string{}
	for _, selection := range selections {
		candidate := ResolveCandidate(graph, selection.Candidate)
		if candidate.Root.Kind != "value" {
			continue
		}
		record := protocol.TypedRecord{ID: candidate.Root.Value.ID, Type: candidate.Type, Value: candidate.Root.Value.Value}
		invariant(selection.Record == record.ID, "SELECTION_RECORD_MISMATCH", selection.Output+" does not select "+record.ID)
		_, conflict := authored[record.ID]
		invariant(!conflict, "PROVIDED_RECORD_CONFLICT", record.ID+" conflicts with authored input", record.ID)
		previous, seen := candidates[record.ID]
		if !seen {
			previous = "an authored Record"
		}
		invariant(!seen || previous == candidate.ID, "PROVIDED_RECORD_CONFLICT", fmt.Sprintf("%s is supplied by both %s and %s", record.ID, previous, candidate.ID), record.ID)
		candidates[record.ID] = candidate.ID
		records[record.ID] = record
	}
	out := make([]protocol.TypedRecord, 0, len(records))
	for _, record := range records {
		out = append(out, record)
	}
	byLocale(out, func(item protocol.TypedRecord) string { return item.ID })
	return out
}

func assertAcyclic(plan protocol.BuildPlan, records map[protocol.RecordId]producedRecord) {
	remaining := map[string]int{}
	dependents := map[string][]string{}
	for _, step := range plan.Steps {
		remaining[step.ID] = 0
		dependents[step.ID] = []string{}
	}
	for _, step := range plan.Steps {
		dependencies := map[string]struct{}{}
		for _, input := range step.Inputs {
			if producer := records[input].step; producer != "" {
				dependencies[producer] = struct{}{}
			}
		}
		remaining[step.ID] = len(dependencies)
		for dependency := range dependencies {
			dependents[dependency] = append(dependents[dependency], step.ID)
		}
	}
	ready := []string{}
	for _, step := range plan.Steps {
		if remaining[step.ID] == 0 {
			ready = append(ready, step.ID)
		}
	}
	visited := 0
	for cursor := 0; cursor < len(ready); cursor++ {
		id := ready[cursor]
		visited++
		for _, dependent := range dependents[id] {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	invariant(visited == len(plan.Steps), "PLAN_CYCLE", "build plan contains a producer cycle")
}

func assertAllStepsReachGoal(plan protocol.BuildPlan, records map[protocol.RecordId]producedRecord) {
	steps := map[string]protocol.ProducerStep{}
	for _, step := range plan.Steps {
		steps[step.ID] = step
	}
	required := map[string]struct{}{}
	queue := []string{}
	for _, goal := range plan.Goals {
		if step := records[goal.Record].step; step != "" {
			queue = append(queue, step)
		}
	}
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if _, seen := required[id]; seen {
			continue
		}
		required[id] = struct{}{}
		step, ok := steps[id]
		invariant(ok, "UNKNOWN_STEP", "unknown step "+id, id)
		for _, input := range step.Inputs {
			if dependency := records[input].step; dependency != "" {
				queue = append(queue, dependency)
			}
		}
	}
	for _, step := range plan.Steps {
		_, ok := required[step.ID]
		invariant(ok, "UNREACHABLE_STEP", step.ID+" does not contribute to any goal", step.ID)
	}
}

func producerStep(plan protocol.BuildPlan, id string) protocol.ProducerStep {
	for _, step := range plan.Steps {
		if step.ID == id {
			return step
		}
	}
	invariant(false, "UNKNOWN_STEP", "unknown step "+id, id)
	return protocol.ProducerStep{}
}

func inputRefs(inputs map[string]protocol.GraphValueRef) []protocol.GraphValueRef {
	names := mapKeys(inputs)
	sort.SliceStable(names, func(i, j int) bool {
		return protocol.LocaleCompare(names[i], names[j]) < 0
	})
	out := make([]protocol.GraphValueRef, len(names))
	for i, name := range names {
		out[i] = inputs[name]
	}
	return out
}

func portNames(ports []protocol.PortDeclaration) []string {
	names := make([]string, len(ports))
	for i, port := range ports {
		names[i] = port.Name
	}
	return names
}

func needNames(ports []protocol.NeedPortDeclaration) []string {
	names := make([]string, len(ports))
	for i, port := range ports {
		names[i] = port.Name
	}
	return names
}
