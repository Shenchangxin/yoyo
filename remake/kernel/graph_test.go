// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/graph.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package kernel

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestSealOrdersIDsAndCanonicalizesValues(t *testing.T) {
	sealed := SealCompiledGraph(protocol.CompiledGraph{
		Outputs: []protocol.LogicalOutput{
			{ID: "A", Type: textType(), Primary: "cand"},
			{ID: "a", Type: textType(), Primary: "cand"},
		},
		Candidates: []protocol.Candidate{
			{
				ID: "B", Type: textType(),
				Root: protocol.CandidateRoot{Kind: "value", Value: protocol.ProvidedValue{
					ID: "lit", Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{"b": 1, "a": 2}},
				}},
			},
			{
				ID: "a", Type: textType(),
				Root: protocol.CandidateRoot{Kind: "value", Value: protocol.ProvidedValue{
					ID: "blob-lit", Value: protocol.StoredValue{Kind: "blob", Resource: "res_a", Size: 1, MediaType: "text/plain"},
				}},
			},
		},
		Operations: []protocol.OperationNode{
			{ID: "b", Result: protocol.OperationResult{Kind: "output", Name: "out", Record: "r"}},
			{ID: "a", Inputs: map[string]protocol.GraphValueRef{"in": {Kind: "record", ID: "src", Operation: "ignored"}}, Result: protocol.OperationResult{Kind: "output", Name: "out", Record: "s"}},
		},
	})
	if sealed.Format != "hypit.graph@1" || sealed.Outputs[0].ID != "a" || sealed.Outputs[1].ID != "A" {
		t.Fatalf("outputs %#v", sealed.Outputs)
	}
	if sealed.Candidates[0].ID != "a" || sealed.Candidates[1].ID != "B" {
		t.Fatalf("candidates %s %s", sealed.Candidates[0].ID, sealed.Candidates[1].ID)
	}
	if protocol.CanonicalStringify(sealed.Candidates[1].Root.Value.Value.Value) != `{"a":2,"b":1}` {
		t.Fatal(protocol.CanonicalStringify(sealed.Candidates[1].Root.Value.Value.Value))
	}
	blob := sealed.Candidates[0].Root.Value.Value
	if blob.Kind != "blob" || blob.Resource != "res_a" || blob.MediaType != "text/plain" {
		t.Fatalf("%+v", blob)
	}
	if sealed.Operations[0].ID != "a" || sealed.Operations[0].Inputs["in"].Operation != "" {
		t.Fatal("operation ref was not normalized")
	}
	request := SealBuildRequest(protocol.BuildRequest{Targets: []protocol.BuildTarget{{Output: "A"}, {Output: "a"}}})
	if request.Format != "hypit.build-request@1" || request.Targets[0].Output != "a" || request.Targets[1].Output != "A" {
		t.Fatalf("%+v", request)
	}
}

func TestCompiledGraphAcceptsARecordInputAndAnOperationCandidate(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{echoModule()}), []protocol.TypedRecord{textRecord("src")})
	graph := echoGraph()
	VerifyCompiledGraph(program, graph)
	if SatisfiedCandidate(graph, "out").ID != "cand" {
		t.Fatal("primary candidate")
	}
	if OperationResultRecord(graph.Operations[0]) != "produced" {
		t.Fatal("result record")
	}
	VerifyBuildRequest(program, graph, protocol.BuildRequest{Format: "hypit.build-request@1", Targets: []protocol.BuildTarget{{Output: "out"}}})
}

func TestCompiledGraphRejects(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{echoModule()}), []protocol.TypedRecord{textRecord("src")})
	graph := echoGraph()
	cases := []struct {
		name string
		code string
		run  func()
	}{
		{name: "format", code: "UNSUPPORTED_GRAPH", run: func() {
			bad := graph
			bad.Format = "nope"
			VerifyCompiledGraph(program, bad)
		}},
		{name: "duplicate output", code: "DUPLICATE_LOGICAL_OUTPUT", run: func() {
			bad := graph
			bad.Outputs = append(bad.Outputs, bad.Outputs[0])
			VerifyCompiledGraph(program, bad)
		}},
		{name: "port mismatch", code: "GRAPH_PORT_BINDING_MISMATCH", run: func() {
			bad := graph
			bad.Operations = []protocol.OperationNode{bad.Operations[0]}
			bad.Operations[0].Inputs = map[string]protocol.GraphValueRef{}
			VerifyCompiledGraph(program, bad)
		}},
		{name: "unknown record", code: "UNKNOWN_GRAPH_INPUT", run: func() {
			bad := graph
			bad.Operations = []protocol.OperationNode{bad.Operations[0]}
			bad.Operations[0].Inputs = map[string]protocol.GraphValueRef{"in": {Kind: "record", ID: "missing"}}
			VerifyCompiledGraph(program, bad)
		}},
		{name: "result name", code: "OPERATION_RESULT_MISMATCH", run: func() {
			bad := graph
			bad.Operations = []protocol.OperationNode{bad.Operations[0]}
			bad.Operations[0].Result.Name = "missing"
			VerifyCompiledGraph(program, bad)
		}},
		{name: "candidate type", code: "CANDIDATE_RESULT_TYPE_MISMATCH", run: func() {
			bad := graph
			bad.Candidates = []protocol.Candidate{bad.Candidates[0]}
			bad.Candidates[0].Type = protocol.TypeRef{Module: exampleRef(), Name: "Missing"}
			VerifyCompiledGraph(program, bad)
		}},
		{name: "duplicate target", code: "DUPLICATE_BUILD_TARGET", run: func() {
			VerifyBuildRequest(program, graph, protocol.BuildRequest{
				Format:  "hypit.build-request@1",
				Targets: []protocol.BuildTarget{{Output: "out"}, {Output: "out"}},
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mustSvml(t, tc.run)
			if err.Code != tc.code {
				t.Fatalf("%s (%s)", err.Code, err.Message)
			}
		})
	}
}

func echoModule() protocol.ModuleManifest {
	manifest := textModule()
	manifest.Producers = []protocol.ProducerDeclaration{{
		Name:    "Echo",
		Inputs:  []protocol.PortDeclaration{{Name: "in", Type: textType()}},
		Outputs: []protocol.PortDeclaration{{Name: "out", Type: textType()}},
		Needs:   []protocol.NeedPortDeclaration{},
	}}
	return manifest
}

func echoGraph() protocol.CompiledGraph {
	return protocol.CompiledGraph{
		Format: "hypit.graph@1",
		Outputs: []protocol.LogicalOutput{{
			ID: "out", Type: textType(), Primary: "cand",
		}},
		Candidates: []protocol.Candidate{{
			ID: "cand", Type: textType(),
			Root: protocol.CandidateRoot{
				Kind:   "operation",
				Result: protocol.OperationResultRef{Kind: "operation-result", Operation: "echo"},
			},
		}},
		Operations: []protocol.OperationNode{{
			ID:       "echo",
			Producer: protocol.ProducerRef{Module: exampleRef(), Name: "Echo"},
			Inputs:   map[string]protocol.GraphValueRef{"in": {Kind: "record", ID: "src"}},
			Result:   protocol.OperationResult{Kind: "output", Name: "out", Record: "produced"},
		}},
	}
}
