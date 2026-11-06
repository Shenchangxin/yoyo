// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/plan.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package kernel

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestCompileBuildPlansTheProducerThatFeedsTheOutput(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{echoModule()}), []protocol.TypedRecord{textRecord("src")})
	graph := echoGraph()
	request := protocol.BuildRequest{Format: "hypit.build-request@1", Targets: []protocol.BuildTarget{{Output: "out"}}}
	plan := CompileBuild(program, graph, request)
	if plan.Format != "hypit.plan@1" || len(plan.Steps) != 1 {
		t.Fatalf("%+v", plan)
	}
	step := plan.Steps[0]
	if step.ID != "echo" || step.Inputs["in"] != "src" || step.Outputs["out"] != "produced" || len(step.Needs) != 0 {
		t.Fatalf("%+v", step)
	}
	if len(plan.Goals) != 1 || plan.Goals[0].Record != "produced" || plan.Goals[0].Type.Name != "Text" {
		t.Fatalf("%+v", plan.Goals)
	}
	if len(plan.OutputBindings) != 1 || plan.OutputBindings[0].Record != "produced" {
		t.Fatalf("%+v", plan.OutputBindings)
	}
	run := PlanBuild(program, graph, protocol.RunGraph{
		Format:  "hypit.run-graph@1",
		Targets: request.Targets,
	})
	if run.Plan.Steps[0].ID != "echo" || len(run.InitialRecords) != 0 || run.Selections[0].Candidate != "cand" {
		t.Fatalf("%+v", run)
	}
}

func TestCompileBuildAdmitsALiteralCandidateAsAnInitialRecord(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{textModule()}), nil)
	graph := protocol.CompiledGraph{
		Format:  "hypit.graph@1",
		Outputs: []protocol.LogicalOutput{{ID: "out", Type: textType(), Primary: "cand"}},
		Candidates: []protocol.Candidate{{
			ID: "cand", Type: textType(),
			Root: protocol.CandidateRoot{Kind: "value", Value: protocol.ProvidedValue{
				ID: "lit", Value: protocol.StoredValue{Kind: "inline", Value: "hello"},
			}},
		}},
	}
	plan := CompileBuild(program, graph, protocol.BuildRequest{
		Format: "hypit.build-request@1", Targets: []protocol.BuildTarget{{Output: "out"}},
	})
	if len(plan.Steps) != 0 || len(plan.Goals) != 1 || plan.Goals[0].Record != "lit" {
		t.Fatalf("%+v", plan)
	}
}

func TestCompileBuildRejectsAProducerCycle(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{echoModule()}), nil)
	ref := protocol.GraphValueRef{Kind: "operation-result"}
	op := func(id, other, record string) protocol.OperationNode {
		input := ref
		input.Operation = other
		return protocol.OperationNode{
			ID: id, Producer: protocol.ProducerRef{Module: exampleRef(), Name: "Echo"},
			Inputs: map[string]protocol.GraphValueRef{"in": input},
			Result: protocol.OperationResult{Kind: "output", Name: "out", Record: record},
		}
	}
	graph := protocol.CompiledGraph{
		Format:  "hypit.graph@1",
		Outputs: []protocol.LogicalOutput{{ID: "out", Type: textType(), Primary: "cand"}},
		Candidates: []protocol.Candidate{{
			ID: "cand", Type: textType(),
			Root: protocol.CandidateRoot{Kind: "operation", Result: protocol.OperationResultRef{Kind: "operation-result", Operation: "op-a"}},
		}},
		Operations: []protocol.OperationNode{op("op-a", "op-b", "rec-a"), op("op-b", "op-a", "rec-b")},
	}
	err := mustSvml(t, func() {
		CompileBuild(program, graph, protocol.BuildRequest{Format: "hypit.build-request@1", Targets: []protocol.BuildTarget{{Output: "out"}}})
	})
	if err.Code != "PLAN_CYCLE" {
		t.Fatal(err.Code, err.Message)
	}
}

func TestVerifyBuildPlanRejectsAnUnreachableStep(t *testing.T) {
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{echoModule()}), []protocol.TypedRecord{textRecord("src")})
	step := func(id, record string) protocol.ProducerStep {
		return protocol.ProducerStep{
			ID: id, Producer: protocol.ProducerRef{Module: exampleRef(), Name: "Echo"},
			Inputs:  map[string]protocol.RecordId{"in": "src"},
			Outputs: map[string]protocol.RecordId{"out": record},
			Needs:   map[string]protocol.NeedBinding{},
		}
	}
	err := mustSvml(t, func() {
		verifyBuildPlan(program, nil, protocol.BuildPlan{
			Format: "hypit.plan@1",
			Steps:  []protocol.ProducerStep{step("echo", "produced"), step("extra", "side")},
			Goals:  []protocol.BuildGoal{{Record: "produced", Type: textType()}},
			OutputBindings: []protocol.BuildOutputBinding{{
				Output: "out", Record: "produced", Type: textType(),
			}},
		})
	})
	if err.Code != "UNREACHABLE_STEP" || err.Subject == nil || *err.Subject != "extra" {
		t.Fatal(err.Code, err.Message)
	}
}

func TestPlannedNeedsFollowsStepOrder(t *testing.T) {
	manifest := textModule()
	manifest.Capabilities = []protocol.CapabilityDeclaration{{Name: "Ask", Returns: textType()}}
	manifest.Producers = []protocol.ProducerDeclaration{{
		Name:  "Asker",
		Needs: []protocol.NeedPortDeclaration{{Name: "ask", Capability: protocol.CapabilityRef{Module: exampleRef(), Name: "Ask"}, Returns: textType()}},
	}}
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{manifest}), nil)
	state := protocol.BuildState{
		Program: program,
		Plan: protocol.BuildPlan{Steps: []protocol.ProducerStep{{
			ID: "ask", Producer: protocol.ProducerRef{Module: exampleRef(), Name: "Asker"},
			Needs: map[string]protocol.NeedBinding{"ask": {ID: "need-1", Result: "answer"}},
		}}},
	}
	found := PlannedNeeds(state)
	if len(found) != 1 || found[0].Need != "need-1" || found[0].Result != "answer" || found[0].Port != "ask" {
		t.Fatalf("%+v", found)
	}
}
