// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/graph.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func graphIndex(graph protocol.CompiledGraph) (outputs map[string]protocol.LogicalOutput, candidates map[string]protocol.Candidate, operations map[string]protocol.OperationNode) {
	outputs = map[string]protocol.LogicalOutput{}
	candidates = map[string]protocol.Candidate{}
	operations = map[string]protocol.OperationNode{}
	for _, item := range graph.Outputs {
		outputs[item.ID] = item
	}
	for _, item := range graph.Candidates {
		candidates[item.ID] = item
	}
	for _, item := range graph.Operations {
		operations[item.ID] = item
	}
	return outputs, candidates, operations
}

func programRecords(program protocol.LinkedProgram) map[string]protocol.TypedRecord {
	records := make(map[string]protocol.TypedRecord, len(program.Records))
	for _, item := range program.Records {
		records[item.ID] = item
	}
	return records
}

func normalizeRef(ref protocol.GraphValueRef) protocol.GraphValueRef {
	if ref.Kind == "record" {
		return protocol.GraphValueRef{Kind: "record", ID: ref.ID}
	}
	if ref.Kind == "logical-output" {
		return protocol.GraphValueRef{Kind: "logical-output", ID: ref.ID}
	}
	return protocol.GraphValueRef{Kind: "operation-result", Operation: ref.Operation}
}

func normalizedStoredValue(value protocol.StoredValue) protocol.StoredValue {
	if value.Kind == "inline" {
		return protocol.StoredValue{Kind: "inline", Value: protocol.Canonicalize(value.Value)}
	}
	return protocol.StoredValue{
		Kind:      "blob",
		Resource:  value.Resource,
		Size:      value.Size,
		MediaType: value.MediaType,
	}
}

func normalizeRoot(root protocol.CandidateRoot) protocol.CandidateRoot {
	if root.Kind == "operation" {
		return protocol.CandidateRoot{
			Kind:   "operation",
			Result: protocol.OperationResultRef{Kind: "operation-result", Operation: root.Result.Operation},
		}
	}
	return protocol.CandidateRoot{
		Kind: "value",
		Value: protocol.ProvidedValue{
			ID:    root.Value.ID,
			Value: normalizedStoredValue(root.Value.Value),
		},
	}
}

func normalizeOutput(output protocol.LogicalOutput) protocol.LogicalOutput {
	return protocol.LogicalOutput{ID: output.ID, Type: output.Type, Primary: output.Primary}
}

func normalizeCandidate(candidate protocol.Candidate) protocol.Candidate {
	return protocol.Candidate{ID: candidate.ID, Type: candidate.Type, Root: normalizeRoot(candidate.Root)}
}

func normalizeResult(result protocol.OperationResult) protocol.OperationResult {
	if result.Kind == "output" {
		return protocol.OperationResult{Kind: "output", Name: result.Name, Record: result.Record}
	}
	return protocol.OperationResult{Kind: "need", Name: result.Name, ID: result.ID, Record: result.Record}
}

func normalizeOperation(operation protocol.OperationNode) protocol.OperationNode {
	names := make([]string, 0, len(operation.Inputs))
	for name := range operation.Inputs {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return protocol.LocaleCompare(names[i], names[j]) < 0
	})
	inputs := make(map[string]protocol.GraphValueRef, len(names))
	for _, name := range names {
		inputs[name] = normalizeRef(operation.Inputs[name])
	}
	return protocol.OperationNode{
		ID:       operation.ID,
		Producer: operation.Producer,
		Inputs:   inputs,
		Result:   normalizeResult(operation.Result),
	}
}

func byLocale[T any](items []T, id func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		return protocol.LocaleCompare(id(items[i]), id(items[j])) < 0
	})
}

// SealCompiledGraph normalizes a graph into hypit.graph@1.
// Outputs, candidates, and operations are ordered by localeCompare of their ids.
func SealCompiledGraph(graph protocol.CompiledGraph) protocol.CompiledGraph {
	outputs := make([]protocol.LogicalOutput, len(graph.Outputs))
	for i, item := range graph.Outputs {
		outputs[i] = normalizeOutput(item)
	}
	byLocale(outputs, func(item protocol.LogicalOutput) string { return item.ID })
	candidates := make([]protocol.Candidate, len(graph.Candidates))
	for i, item := range graph.Candidates {
		candidates[i] = normalizeCandidate(item)
	}
	byLocale(candidates, func(item protocol.Candidate) string { return item.ID })
	operations := make([]protocol.OperationNode, len(graph.Operations))
	for i, item := range graph.Operations {
		operations[i] = normalizeOperation(item)
	}
	byLocale(operations, func(item protocol.OperationNode) string { return item.ID })
	return protocol.CompiledGraph{
		Format:     "hypit.graph@1",
		Outputs:    outputs,
		Candidates: candidates,
		Operations: operations,
	}
}

// SealBuildRequest normalizes a request into hypit.build-request@1.
func SealBuildRequest(request protocol.BuildRequest) protocol.BuildRequest {
	targets := make([]protocol.BuildTarget, len(request.Targets))
	for i, target := range request.Targets {
		targets[i] = protocol.BuildTarget{Output: target.Output}
	}
	byLocale(targets, func(item protocol.BuildTarget) string { return item.Output })
	return protocol.BuildRequest{Format: "hypit.build-request@1", Targets: targets}
}

// ResolveLogicalOutput returns the output named by id.
func ResolveLogicalOutput(graph protocol.CompiledGraph, id string) protocol.LogicalOutput {
	outputs, _, _ := graphIndex(graph)
	output, ok := outputs[id]
	invariant(ok, "UNKNOWN_LOGICAL_OUTPUT", "unknown logical output "+id, id)
	return output
}

// ResolveCandidate returns the candidate named by id.
func ResolveCandidate(graph protocol.CompiledGraph, id string) protocol.Candidate {
	_, candidates, _ := graphIndex(graph)
	candidate, ok := candidates[id]
	invariant(ok, "UNKNOWN_CANDIDATE", "unknown Candidate "+id, id)
	return candidate
}

// ResolveOperation returns the operation named by id.
func ResolveOperation(graph protocol.CompiledGraph, id string) protocol.OperationNode {
	_, _, operations := graphIndex(graph)
	operation, ok := operations[id]
	invariant(ok, "UNKNOWN_OPERATION", "unknown Operation "+id, id)
	return operation
}

// SatisfiedCandidate returns the primary candidate of one logical output.
func SatisfiedCandidate(graph protocol.CompiledGraph, outputID string) protocol.Candidate {
	output := ResolveLogicalOutput(graph, outputID)
	return ResolveCandidate(graph, output.Primary)
}

// OperationResultRecord returns the record id written by an operation result.
func OperationResultRecord(operation protocol.OperationNode) string {
	return operation.Result.Record
}

func operationResultType(program protocol.LinkedProgram, operation protocol.OperationNode) protocol.TypeRef {
	producer := ResolveProducer(program.Closure, operation.Producer)
	if operation.Result.Kind == "output" {
		var output *protocol.PortDeclaration
		for i := range producer.Outputs {
			if producer.Outputs[i].Name == operation.Result.Name {
				output = &producer.Outputs[i]
				break
			}
		}
		invariant(output != nil && len(producer.Needs) == 0, "OPERATION_RESULT_MISMATCH", operation.ID+" does not produce output "+operation.Result.Name, operation.ID)
		return output.Type
	}
	var need *protocol.NeedPortDeclaration
	for i := range producer.Needs {
		if producer.Needs[i].Name == operation.Result.Name {
			need = &producer.Needs[i]
			break
		}
	}
	invariant(need != nil && len(producer.Outputs) == 0, "OPERATION_RESULT_MISMATCH", operation.ID+" does not request Need "+operation.Result.Name, operation.ID)
	return need.Returns
}

func graphValueType(program protocol.LinkedProgram, graph protocol.CompiledGraph, ref protocol.GraphValueRef) protocol.TypeRef {
	if ref.Kind == "record" {
		record, ok := programRecords(program)[ref.ID]
		invariant(ok, "UNKNOWN_GRAPH_INPUT", "unknown authored record "+ref.ID, ref.ID)
		return record.Type
	}
	if ref.Kind == "logical-output" {
		return ResolveLogicalOutput(graph, ref.ID).Type
	}
	return operationResultType(program, ResolveOperation(graph, ref.Operation))
}

func exactKeys(actual map[string]protocol.GraphValueRef, expected []string, subject string) {
	left := make([]string, 0, len(actual))
	for key := range actual {
		left = append(left, key)
	}
	sortUTF16(left)
	right := append([]string(nil), expected...)
	sortUTF16(right)
	if slices.Equal(left, right) {
		return
	}
	invariant(false, "GRAPH_PORT_BINDING_MISMATCH", fmt.Sprintf("%s binds [%s] but declares [%s]", subject, strings.Join(left, ", "), strings.Join(right, ", ")), subject)
}

func sortUTF16(values []string) {
	sort.SliceStable(values, func(i, j int) bool {
		return utf16Less(values[i], values[j])
	})
}

// utf16Less is Array.prototype.sort's default order: UTF-16 code units.
func utf16Less(left, right string) bool {
	a := utf16.Encode([]rune(left))
	b := utf16.Encode([]rune(right))
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func verifyOperation(program protocol.LinkedProgram, graph protocol.CompiledGraph, operation protocol.OperationNode) {
	invariant(operation.ID != "", "EMPTY_OPERATION_ID", "Operation id is empty")
	invariant(operation.Result.Record != "", "EMPTY_RECORD_ID", operation.ID+" result record is empty")
	producer := ResolveProducer(program.Closure, operation.Producer)
	names := make([]string, len(producer.Inputs))
	for i, port := range producer.Inputs {
		names[i] = port.Name
	}
	exactKeys(operation.Inputs, names, operation.ID+".inputs")
	invariant(len(producer.Outputs)+len(producer.Needs) == 1, "PRODUCER_RESULT_NORMAL_FORM", producerKey(operation.Producer)+" must declare exactly one public result")
	operationResultType(program, operation)
	if operation.Result.Kind == "need" {
		invariant(operation.Result.ID != "", "EMPTY_NEED_ID", operation.ID+" Need id is empty")
	}
	for _, declaration := range producer.Inputs {
		supplied := graphValueType(program, graph, operation.Inputs[declaration.Name])
		invariant(sameType(supplied, declaration.Type), "GRAPH_INPUT_TYPE_MISMATCH", fmt.Sprintf("%s.%s wants %s but receives %s", operation.ID, declaration.Name, typeKey(declaration.Type), typeKey(supplied)), operation.ID)
	}
}

func verifyCandidateValue(program protocol.LinkedProgram, graph protocol.CompiledGraph, candidate protocol.Candidate) {
	if candidate.Root.Kind == "operation" {
		supplied := operationResultType(program, ResolveOperation(graph, candidate.Root.Result.Operation))
		invariant(sameType(supplied, candidate.Type), "CANDIDATE_RESULT_TYPE_MISMATCH", fmt.Sprintf("%s declares %s but returns %s", candidate.ID, typeKey(candidate.Type), typeKey(supplied)), candidate.ID)
		return
	}
	VerifyRecordStructure(program.Closure, protocol.TypedRecord{
		ID:    candidate.Root.Value.ID,
		Type:  candidate.Type,
		Value: candidate.Root.Value.Value,
	})
}

func verifySatisfaction(output protocol.LogicalOutput, candidate protocol.Candidate) {
	invariant(sameType(candidate.Type, output.Type), "CANDIDATE_RESULT_TYPE_MISMATCH", fmt.Sprintf("%s supplies %s, not %s", candidate.ID, typeKey(candidate.Type), typeKey(output.Type)), candidate.ID)
}

// VerifyCompiledGraph checks one linked program against one sealed graph.
func VerifyCompiledGraph(program protocol.LinkedProgram, graph protocol.CompiledGraph) {
	invariant(graph.Format == "hypit.graph@1", "UNSUPPORTED_GRAPH", "unsupported compiled graph format")
	outputIDs := map[string]struct{}{}
	candidateIDs := map[string]struct{}{}
	operationIDs := map[string]struct{}{}
	operationRecords := map[string]struct{}{}
	for _, record := range program.Records {
		operationRecords[record.ID] = struct{}{}
	}
	needIDs := map[string]struct{}{}

	for _, output := range graph.Outputs {
		invariant(output.ID != "", "EMPTY_LOGICAL_OUTPUT_ID", "Logical Output id is empty")
		_, duplicate := outputIDs[output.ID]
		invariant(!duplicate, "DUPLICATE_LOGICAL_OUTPUT", "duplicate "+output.ID, output.ID)
		outputIDs[output.ID] = struct{}{}
	}
	for _, operation := range graph.Operations {
		_, duplicate := operationIDs[operation.ID]
		invariant(!duplicate, "DUPLICATE_OPERATION", "duplicate "+operation.ID, operation.ID)
		operationIDs[operation.ID] = struct{}{}
		_, sourced := operationRecords[operation.Result.Record]
		invariant(!sourced, "DUPLICATE_RECORD", "record "+operation.Result.Record+" has multiple sources", operation.Result.Record)
		operationRecords[operation.Result.Record] = struct{}{}
		if operation.Result.Kind == "need" {
			_, duplicateNeed := needIDs[operation.Result.ID]
			invariant(!duplicateNeed, "DUPLICATE_NEED", "duplicate Need "+operation.Result.ID)
			needIDs[operation.Result.ID] = struct{}{}
		}
	}
	for _, operation := range graph.Operations {
		verifyOperation(program, graph, operation)
	}
	for _, candidate := range graph.Candidates {
		invariant(candidate.ID != "", "EMPTY_CANDIDATE_ID", "Candidate id is empty")
		_, duplicate := candidateIDs[candidate.ID]
		invariant(!duplicate, "DUPLICATE_CANDIDATE", "duplicate "+candidate.ID, candidate.ID)
		candidateIDs[candidate.ID] = struct{}{}
		verifyCandidateValue(program, graph, candidate)
	}
	for _, output := range graph.Outputs {
		verifySatisfaction(output, ResolveCandidate(graph, output.Primary))
	}
}

// VerifyBuildRequest checks the graph and then the requested outputs.
func VerifyBuildRequest(program protocol.LinkedProgram, graph protocol.CompiledGraph, request protocol.BuildRequest) {
	VerifyCompiledGraph(program, graph)
	invariant(request.Format == "hypit.build-request@1", "UNSUPPORTED_BUILD_REQUEST", "unsupported BuildRequest format")
	targets := map[string]struct{}{}
	for _, target := range request.Targets {
		ResolveLogicalOutput(graph, target.Output)
		_, duplicate := targets[target.Output]
		invariant(!duplicate, "DUPLICATE_BUILD_TARGET", target.Output+" is targeted twice")
		targets[target.Output] = struct{}{}
	}
}
