// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/test/core.test.ts and greeting-fixture.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package kernel

import (
	"testing"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestCoreRunsAGreetingThroughANeed(t *testing.T) {
	state, command := reachGreetingNeed(t)
	state = Reduce(state, &protocol.CommandResult{
		Kind: "need-fulfilled", Command: command.ID,
		Value: protocol.StoredValue{Kind: "inline", Value: "Hello, Ada!"},
	})
	var assemble protocol.BuildCommand
	found := false
	for _, item := range state.Outstanding {
		if item.Kind == "invoke-producer" {
			assemble = item
			found = true
		}
	}
	if !found {
		t.Fatal("assemble was not scheduled")
	}
	state = Reduce(state, &protocol.CommandResult{
		Kind: "producer-completed", Command: assemble.ID,
		Outputs: map[string]protocol.StoredValue{
			"document": {Kind: "inline", Value: map[string]any{"text": "Hello, Ada!"}},
		},
		Needs: map[string]any{},
	})
	if state.Status != "complete" || len(state.Outstanding) != 0 {
		t.Fatalf("status %s outstanding %d", state.Status, len(state.Outstanding))
	}
	last := state.Records[len(state.Records)-1]
	got := protocol.CanonicalStringify(last.Value.Value)
	if last.Value.Kind != "inline" || got != `{"text":"Hello, Ada!"}` {
		t.Fatalf("%s %s", last.Value.Kind, got)
	}
}

func TestNeedCommandRemainsResolvableAfterFailure(t *testing.T) {
	state, command := reachGreetingNeed(t)
	resolved, ok := ResolveNeedCommand(state, command.ID)
	if !ok || resolved.ID != command.ID || resolved.Need.ID != command.Need.ID {
		t.Fatalf("%+v %v", resolved, ok)
	}
	failed := Reduce(state, &protocol.CommandResult{
		Kind: "command-failed", Command: command.ID, Code: "EXAMPLE", Message: "stopped",
	})
	if failed.Status != "failed" || len(failed.Outstanding) != 0 {
		t.Fatalf("%s %d", failed.Status, len(failed.Outstanding))
	}
	if _, ok := ResolveNeedCommand(failed, command.ID); !ok {
		t.Fatal("need command disappeared")
	}
	if _, ok := ResolveNeedCommand(failed, "absent-command"); ok {
		t.Fatal("absent command resolved")
	}
}

func TestMachineCommitMatchesTheReducer(t *testing.T) {
	initial := greetingBuild(false)
	authored := map[string]struct{}{}
	for _, record := range initial.Program.Records {
		authored[record.ID] = struct{}{}
	}
	var initialRecords []protocol.TypedRecord
	for _, record := range initial.Records {
		if _, ok := authored[record.ID]; !ok {
			initialRecords = append(initialRecords, record)
		}
	}
	definition := DefineBuild(protocol.BuildDefinition{
		Program: initial.Program, InitialRecords: initialRecords, Plan: initial.Plan, Targets: initial.Targets,
	})
	machine := NewBuildMachine(definition, nil)
	prompt := machine.Commands()[0]
	if prompt.Kind != "invoke-producer" {
		t.Fatal(prompt.Kind)
	}
	before := machine.View()
	fact := machine.Evaluate(protocol.CommandResult{
		Kind: "producer-completed", Command: prompt.ID,
		Outputs: map[string]protocol.StoredValue{"prompt": {Kind: "inline", Value: "Greet Ada"}},
		Needs:   map[string]any{},
	})
	if machine.View().Status != before.Status || len(machine.Commands()) != len(before.Outstanding) {
		t.Fatal("evaluate advanced the machine")
	}
	machine.Commit()
	if fact.Kind != "producer-applied" || machine.Commands()[0].Step != "request-text" {
		t.Fatalf("%s %s", fact.Kind, machine.Commands()[0].Step)
	}
	materialized := MaterializeBuild(definition, []protocol.BuildFact{fact})
	if len(materialized.Outstanding) != 1 || materialized.Outstanding[0].ID != machine.Commands()[0].ID {
		t.Fatalf("materialized %v machine %v", materialized.Outstanding, machine.Commands())
	}
	complete := false
	for _, step := range materialized.Steps {
		if step.ID == "make-prompt" && step.Status == "complete" {
			complete = true
		}
	}
	if !complete {
		t.Fatalf("steps %+v", materialized.Steps)
	}
}

func reachGreetingNeed(t *testing.T) (protocol.BuildState, protocol.BuildCommand) {
	t.Helper()
	state := greetingBuild(false)
	if len(state.Outstanding) != 1 || state.Outstanding[0].Kind != "invoke-producer" {
		t.Fatalf("first command %+v", state.Outstanding)
	}
	current := state.Outstanding[0]
	state = Reduce(state, &protocol.CommandResult{
		Kind: "producer-completed", Command: current.ID,
		Outputs: map[string]protocol.StoredValue{"prompt": {Kind: "inline", Value: "Greet Ada"}},
		Needs:   map[string]any{},
	})
	var request protocol.BuildCommand
	found := false
	for _, command := range state.Outstanding {
		if command.Kind == "invoke-producer" {
			request = command
			found = true
		}
	}
	if !found {
		t.Fatal("request-text was not scheduled")
	}
	state = Reduce(state, &protocol.CommandResult{
		Kind: "producer-completed", Command: request.ID,
		Outputs: map[string]protocol.StoredValue{},
		Needs:   map[string]any{"generation": map[string]any{"prompt": "Greet Ada"}},
	})
	for _, item := range state.Outstanding {
		if item.Kind == "fulfill-need" {
			return state, item
		}
	}
	t.Fatal("need was not scheduled")
	return protocol.BuildState{}, protocol.BuildCommand{}
}

func greetingBuild(includeSide bool) protocol.BuildState {
	ref := protocol.ModuleRef{Name: "example.greeting", Version: "0.0.0"}
	typeOf := func(name string) protocol.TypeRef {
		return protocol.TypeRef{Module: ref, Name: name}
	}
	intent, prompt, generated, document := typeOf("GreetingIntent"), typeOf("TextPrompt"), typeOf("GeneratedText"), typeOf("GreetingDocument")
	capability := protocol.CapabilityRef{Module: ref, Name: "generate-greeting-text"}
	manifest := protocol.ModuleManifest{
		Format: "hypit.module@1", Name: ref.Name, Version: ref.Version,
		Types:        []protocol.TypeDeclaration{{Name: intent.Name}, {Name: prompt.Name}, {Name: generated.Name}, {Name: document.Name}},
		Capabilities: []protocol.CapabilityDeclaration{{Name: capability.Name, Returns: generated}},
		Producers: []protocol.ProducerDeclaration{
			{Name: "make-prompt", Inputs: []protocol.PortDeclaration{{Name: "intent", Type: intent}}, Outputs: []protocol.PortDeclaration{{Name: "prompt", Type: prompt}}},
			{Name: "request-text", Inputs: []protocol.PortDeclaration{{Name: "prompt", Type: prompt}}, Needs: []protocol.NeedPortDeclaration{{Name: "generation", Capability: capability, Returns: generated}}},
			{Name: "placeholder-text", Inputs: []protocol.PortDeclaration{{Name: "prompt", Type: prompt}}, Outputs: []protocol.PortDeclaration{{Name: "generated", Type: generated}}},
			{Name: "assemble", Inputs: []protocol.PortDeclaration{{Name: "generated", Type: generated}}, Outputs: []protocol.PortDeclaration{{Name: "document", Type: document}}},
		},
	}
	program := Link(CreateResolvedClosure([]protocol.ModuleManifest{manifest}), []protocol.TypedRecord{{
		ID: "intent:root", Type: intent, Value: protocol.StoredValue{Kind: "inline", Value: map[string]any{"name": "Ada"}},
	}})
	producer := func(name string) protocol.ProducerRef { return protocol.ProducerRef{Module: ref, Name: name} }
	op := func(id, name, port, record string, input protocol.GraphValueRef, kind string) protocol.OperationNode {
		result := protocol.OperationResult{Kind: kind, Name: port, Record: record}
		if kind == "need" {
			result.ID = "need:generation"
		}
		return protocol.OperationNode{ID: id, Producer: producer(name), Inputs: map[string]protocol.GraphValueRef{inputName(name): input}, Result: result}
	}
	outputs := []protocol.LogicalOutput{
		{ID: "prompt", Type: prompt, Primary: "make-prompt"},
		{ID: "generated", Type: generated, Primary: "request-text"},
		{ID: "document", Type: document, Primary: "assemble"},
	}
	candidates := []protocol.Candidate{
		operationCandidate("make-prompt", prompt),
		operationCandidate("request-text", generated),
		operationCandidate("placeholder-text", generated),
		operationCandidate("assemble", document),
	}
	operations := []protocol.OperationNode{
		op("make-prompt", "make-prompt", "prompt", "prompt:root", protocol.GraphValueRef{Kind: "record", ID: "intent:root"}, "output"),
		op("request-text", "request-text", "generation", "generated:root", protocol.GraphValueRef{Kind: "logical-output", ID: "prompt"}, "need"),
		op("placeholder-text", "placeholder-text", "generated", "generated:placeholder", protocol.GraphValueRef{Kind: "logical-output", ID: "prompt"}, "output"),
		op("assemble", "assemble", "document", "document:root", protocol.GraphValueRef{Kind: "logical-output", ID: "generated"}, "output"),
	}
	if includeSide {
		outputs = append(outputs,
			protocol.LogicalOutput{ID: "side-generated", Type: generated, Primary: "side-placeholder"},
			protocol.LogicalOutput{ID: "side-document", Type: document, Primary: "side-assemble"},
		)
		candidates = append(candidates, operationCandidate("side-placeholder", generated), operationCandidate("side-assemble", document))
		operations = append(operations,
			op("side-placeholder", "placeholder-text", "generated", "generated:side", protocol.GraphValueRef{Kind: "logical-output", ID: "prompt"}, "output"),
			op("side-assemble", "assemble", "document", "document:side", protocol.GraphValueRef{Kind: "logical-output", ID: "side-generated"}, "output"),
		)
	}
	graph := SealCompiledGraph(protocol.CompiledGraph{Outputs: outputs, Candidates: candidates, Operations: operations})
	targets := []protocol.BuildTarget{{Output: "document"}}
	if includeSide {
		targets = []protocol.BuildTarget{{Output: "side-document"}, {Output: "document"}}
	}
	return Start(program, graph, SealBuildRequest(protocol.BuildRequest{Targets: targets}))
}

func inputName(producer string) string {
	switch producer {
	case "make-prompt":
		return "intent"
	case "assemble":
		return "generated"
	default:
		return "prompt"
	}
}

func operationCandidate(id string, typ protocol.TypeRef) protocol.Candidate {
	return protocol.Candidate{
		ID: id, Type: typ,
		Root: protocol.CandidateRoot{Kind: "operation", Result: protocol.OperationResultRef{Kind: "operation-result", Operation: id}},
	}
}
