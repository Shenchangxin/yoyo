// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/reducer.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func commandID(kind, subject string) string {
	return kind + ":" + subject
}

func buildNeedCommand(need protocol.Need) protocol.BuildCommand {
	return protocol.BuildCommand{Kind: "fulfill-need", ID: commandID("need", need.ID), Need: need}
}

func buildProducerCommand(step protocol.ProducerStep) protocol.BuildCommand {
	return protocol.BuildCommand{
		Kind:     "invoke-producer",
		ID:       commandID("producer", step.ID),
		Step:     step.ID,
		Producer: step.Producer,
		Inputs:   step.Inputs,
	}
}

// ResolveNeedCommand returns an issued Need command even after scheduling has stopped.
func ResolveNeedCommand(state protocol.BuildState, id protocol.CommandId) (protocol.BuildCommand, bool) {
	for _, need := range state.Needs {
		command := buildNeedCommand(need)
		if command.ID == id {
			return command, true
		}
	}
	return protocol.BuildCommand{}, false
}

func withoutCommand(state protocol.BuildState, id string) []protocol.BuildCommand {
	out := make([]protocol.BuildCommand, 0, len(state.Outstanding))
	for _, command := range state.Outstanding {
		if command.ID != id {
			out = append(out, command)
		}
	}
	return out
}

func normalizeStoredValue(value protocol.StoredValue) protocol.StoredValue {
	if value.Kind == "inline" {
		return protocol.StoredValue{Kind: "inline", Value: protocol.Canonicalize(value.Value)}
	}
	invariant(value.Size >= 0 && value.Size <= 9007199254740991 && value.Size <= math.MaxInt64, "INVALID_BLOB", "blob size is invalid")
	return value
}

func exactResultKeys(actual []string, expected []string, subject string) {
	left := append([]string(nil), actual...)
	right := append([]string(nil), expected...)
	sortUTF16(left)
	sortUTF16(right)
	if slices.Equal(left, right) {
		return
	}
	invariant(false, "PRODUCER_RESULT_PORT_MISMATCH", fmt.Sprintf("%s returned [%s] but must return [%s]", subject, strings.Join(left, ", "), strings.Join(right, ", ")), subject)
}

func acceptProducerEvent(state protocol.BuildState, command protocol.BuildCommand, event protocol.CommandResult) protocol.BuildState {
	step := producerStep(state.Plan, command.Step)
	var stepState protocol.StepState
	found := false
	for _, item := range state.Steps {
		if item.ID == step.ID {
			stepState = item
			found = true
			break
		}
	}
	invariant(found && stepState.Status == "pending", "STEP_ALREADY_COMPLETE", step.ID+" is already complete", step.ID)
	producer := ResolveProducer(state.Program.Closure, step.Producer)
	exactResultKeys(mapKeys(event.Outputs), portNames(producer.Outputs), step.ID+".outputs")
	exactResultKeys(mapKeys(event.Needs), needNames(producer.Needs), step.ID+".needs")

	var outputs []protocol.TypedRecord
	for _, port := range producer.Outputs {
		id, bound := step.Outputs[port.Name]
		if !bound {
			continue
		}
		raw, present := event.Outputs[port.Name]
		invariant(present, "MISSING_OUTPUT_VALUE", step.ID+"."+port.Name+" returned no value")
		outputs = append(outputs, protocol.TypedRecord{ID: id, Type: port.Type, Value: normalizeStoredValue(raw)})
	}
	var needs []protocol.Need
	for _, port := range producer.Needs {
		binding, bound := step.Needs[port.Name]
		if !bound {
			continue
		}
		constraints, present := event.Needs[port.Name]
		invariant(present, "MISSING_NEED_VALUE", step.ID+"."+port.Name+" returned no constraints")
		needs = append(needs, protocol.Need{
			ID:          binding.ID,
			Capability:  port.Capability,
			Returns:     port.Returns,
			Constraints: protocol.Canonicalize(constraints),
			Result:      binding.Result,
		})
	}
	for _, record := range outputs {
		VerifyRecordStructure(state.Program.Closure, record)
	}
	next := state
	next.Records = append(append([]protocol.TypedRecord{}, state.Records...), outputs...)
	next.Needs = append(append([]protocol.Need{}, state.Needs...), needs...)
	next.Steps = make([]protocol.StepState, len(state.Steps))
	copy(next.Steps, state.Steps)
	for i := range next.Steps {
		if next.Steps[i].ID == step.ID {
			next.Steps[i].Status = "complete"
		}
	}
	next.Outstanding = withoutCommand(state, command.ID)
	return next
}

func acceptNeedEvent(state protocol.BuildState, command protocol.BuildCommand, event protocol.CommandResult) protocol.BuildState {
	var need protocol.Need
	found := false
	for _, item := range state.Needs {
		if item.ID == command.Need.ID {
			need = item
			found = true
			break
		}
	}
	invariant(found, "UNKNOWN_NEED", "unknown need "+command.Need.ID, command.Need.ID)
	for _, record := range state.Records {
		invariant(record.ID != need.Result, "NEED_ALREADY_FULFILLED", need.ID+" is already fulfilled", need.ID)
	}
	record := protocol.TypedRecord{ID: need.Result, Type: need.Returns, Value: normalizeStoredValue(event.Value)}
	VerifyRecordStructure(state.Program.Closure, record)
	next := state
	next.Records = append(append([]protocol.TypedRecord{}, state.Records...), record)
	next.Outstanding = withoutCommand(state, command.ID)
	return next
}

func acceptFailure(state protocol.BuildState, event protocol.CommandResult) protocol.BuildState {
	next := state
	next.Status = "failed"
	next.Outstanding = []protocol.BuildCommand{}
	subject := event.Command
	next.Diagnostics = append(append([]protocol.BuildDiagnostic{}, state.Diagnostics...), protocol.BuildDiagnostic{
		Code: event.Code, Message: event.Message, Subject: &subject,
	})
	return next
}

func applyEvent(state protocol.BuildState, event protocol.CommandResult) protocol.BuildState {
	var command protocol.BuildCommand
	found := false
	for _, item := range state.Outstanding {
		if item.ID == event.Command {
			command = item
			found = true
			break
		}
	}
	invariant(found, "UNKNOWN_COMMAND", "event references "+event.Command, event.Command)
	if event.Kind == "command-failed" {
		return acceptFailure(state, event)
	}
	if command.Kind == "invoke-producer" && event.Kind == "producer-completed" {
		return acceptProducerEvent(state, command, event)
	}
	if command.Kind == "fulfill-need" && event.Kind == "need-fulfilled" {
		return acceptNeedEvent(state, command, event)
	}
	panic(&protocol.SvmlError{
		Code:    "EVENT_COMMAND_MISMATCH",
		Message: event.Kind + " cannot complete " + command.Kind,
		Subject: &command.ID,
	})
}

func goalsComplete(state protocol.BuildState, records map[string]struct{}) bool {
	for _, goal := range state.Plan.Goals {
		if _, ok := records[goal.Record]; !ok {
			return false
		}
	}
	return true
}

func schedule(state protocol.BuildState) protocol.BuildState {
	if state.Status != "active" {
		return state
	}
	records := map[string]struct{}{}
	for _, record := range state.Records {
		records[record.ID] = struct{}{}
	}
	if goalsComplete(state, records) {
		next := state
		next.Status = "complete"
		next.Outstanding = []protocol.BuildCommand{}
		return next
	}
	issued := map[string]struct{}{}
	commands := append([]protocol.BuildCommand{}, state.Outstanding...)
	for _, command := range commands {
		issued[command.ID] = struct{}{}
	}
	for _, need := range state.Needs {
		if _, done := records[need.Result]; done {
			continue
		}
		command := buildNeedCommand(need)
		if _, seen := issued[command.ID]; seen {
			continue
		}
		commands = append(commands, command)
		issued[command.ID] = struct{}{}
	}
	for _, stepState := range state.Steps {
		if stepState.Status != "pending" {
			continue
		}
		step := producerStep(state.Plan, stepState.ID)
		ready := true
		for _, id := range step.Inputs {
			if _, ok := records[id]; !ok {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		command := buildProducerCommand(step)
		if _, seen := issued[command.ID]; seen {
			continue
		}
		commands = append(commands, command)
	}
	sortCommands(commands)
	if len(commands) == 0 {
		next := state
		next.Status = "failed"
		next.Diagnostics = append(append([]protocol.BuildDiagnostic{}, state.Diagnostics...), protocol.BuildDiagnostic{
			Code:    "BUILD_DEADLOCK",
			Message: "no command can make progress toward the requested goals",
		})
		return next
	}
	next := state
	next.Outstanding = commands
	return next
}

func sortCommands(commands []protocol.BuildCommand) {
	for i := 1; i < len(commands); i++ {
		j := i
		for j > 0 && protocol.LocaleCompare(commands[j-1].ID, commands[j].ID) > 0 {
			commands[j-1], commands[j] = commands[j], commands[j-1]
			j--
		}
	}
}

// Start plans an author graph and returns the first scheduled view.
func Start(program protocol.LinkedProgram, graph protocol.CompiledGraph, request protocol.BuildRequest) protocol.BuildState {
	planned := PlanBuild(program, graph, protocol.RunGraph{
		Format:  "hypit.run-graph@1",
		Records: []protocol.TypedRecord{},
		Targets: request.Targets,
	})
	return schedule(initialBuildView(DefineBuild(protocol.BuildDefinition{
		Program:        program,
		InitialRecords: planned.InitialRecords,
		Plan:           planned.Plan,
		Targets:        request.Targets,
	})))
}

// Reduce applies one optional result and schedules the commands that can run now.
func Reduce(state protocol.BuildState, event *protocol.CommandResult) protocol.BuildState {
	next := state
	if event != nil {
		next = applyEvent(state, *event)
	}
	return schedule(next)
}

// DefineBuild seals one already selected execution.
func DefineBuild(input protocol.BuildDefinition) protocol.BuildDefinition {
	verifyBuildPlan(input.Program, input.InitialRecords, input.Plan)
	bindings := map[string]struct{}{}
	for _, binding := range input.Plan.OutputBindings {
		bindings[binding.Output] = struct{}{}
	}
	targets := map[string]struct{}{}
	for _, target := range input.Targets {
		_, ok := bindings[target.Output]
		invariant(ok, "UNKNOWN_BUILD_TARGET", target.Output+" has no Output binding", target.Output)
		_, duplicate := targets[target.Output]
		invariant(!duplicate, "DUPLICATE_BUILD_TARGET", target.Output+" is targeted twice", target.Output)
		targets[target.Output] = struct{}{}
	}
	input.Format = "hypit.build-definition@1"
	return cloneDefinition(input)
}

func initialBuildView(definition protocol.BuildDefinition) protocol.BuildState {
	invariant(definition.Format == "hypit.build-definition@1", "UNSUPPORTED_BUILD_DEFINITION", definition.Format)
	verifyBuildPlan(definition.Program, definition.InitialRecords, definition.Plan)
	records := make([]protocol.TypedRecord, 0, len(definition.Program.Records)+len(definition.InitialRecords))
	records = append(records, definition.Program.Records...)
	records = append(records, definition.InitialRecords...)
	steps := make([]protocol.StepState, len(definition.Plan.Steps))
	for i, step := range definition.Plan.Steps {
		steps[i] = protocol.StepState{ID: step.ID, Status: "pending"}
	}
	return protocol.BuildState{
		Format:      "hypit.build@1",
		Program:     definition.Program,
		Plan:        definition.Plan,
		Targets:     append([]protocol.BuildTarget{}, definition.Targets...),
		Status:      "active",
		Records:     records,
		Steps:       steps,
		Needs:       []protocol.Need{},
		Outstanding: []protocol.BuildCommand{},
		Diagnostics: []protocol.BuildDiagnostic{},
	}
}

// BuildAdmissionLookup supplies the indexes BuildMachine keeps privately.
type BuildAdmissionLookup struct {
	Command    func(protocol.CommandId) (protocol.BuildCommand, bool)
	StepStatus func(string) (string, bool)
	Need       func(string) (protocol.Need, bool)
	Record     func(string) (protocol.TypedRecord, bool)
}

// FactForBuildResult validates one result and derives its durable fact.
func FactForBuildResult(state protocol.BuildState, event protocol.CommandResult, indexed *BuildAdmissionLookup) protocol.BuildFact {
	lookup := indexed
	if lookup == nil {
		lookup = &BuildAdmissionLookup{
			Command: func(id protocol.CommandId) (protocol.BuildCommand, bool) {
				for _, item := range state.Outstanding {
					if item.ID == id {
						return item, true
					}
				}
				return protocol.BuildCommand{}, false
			},
			StepStatus: func(id string) (string, bool) {
				for _, item := range state.Steps {
					if item.ID == id {
						return item.Status, true
					}
				}
				return "", false
			},
			Need: func(id string) (protocol.Need, bool) {
				for _, item := range state.Needs {
					if item.ID == id {
						return item, true
					}
				}
				return protocol.Need{}, false
			},
			Record: func(id string) (protocol.TypedRecord, bool) {
				for _, item := range state.Records {
					if item.ID == id {
						return item, true
					}
				}
				return protocol.TypedRecord{}, false
			},
		}
	}
	command, ok := lookup.Command(event.Command)
	invariant(ok, "UNKNOWN_COMMAND", "event references "+event.Command, event.Command)
	if event.Kind == "command-failed" {
		subject := event.Command
		return protocol.BuildFact{
			Format:  "hypit.build-fact@1",
			Kind:    "command-failed",
			Command: event.Command,
			Diagnostic: protocol.BuildDiagnostic{
				Code: event.Code, Message: event.Message, Subject: &subject,
			},
		}
	}
	if command.Kind == "invoke-producer" && event.Kind == "producer-completed" {
		step := producerStep(state.Plan, command.Step)
		status, statusOK := lookup.StepStatus(step.ID)
		invariant(statusOK && status == "pending", "STEP_ALREADY_COMPLETE", step.ID+" is already complete", step.ID)
		producer := ResolveProducer(state.Program.Closure, step.Producer)
		exactResultKeys(mapKeys(event.Outputs), portNames(producer.Outputs), step.ID+".outputs")
		exactResultKeys(mapKeys(event.Needs), needNames(producer.Needs), step.ID+".needs")
		var records []protocol.TypedRecord
		for _, port := range producer.Outputs {
			id, bound := step.Outputs[port.Name]
			if !bound {
				continue
			}
			raw, present := event.Outputs[port.Name]
			invariant(present, "MISSING_OUTPUT_VALUE", step.ID+"."+port.Name+" returned no value")
			records = append(records, protocol.TypedRecord{ID: id, Type: port.Type, Value: normalizeStoredValue(raw)})
		}
		var needs []protocol.Need
		for _, port := range producer.Needs {
			binding, bound := step.Needs[port.Name]
			if !bound {
				continue
			}
			constraints, present := event.Needs[port.Name]
			invariant(present, "MISSING_NEED_VALUE", step.ID+"."+port.Name+" returned no constraints")
			needs = append(needs, protocol.Need{
				ID: binding.ID, Capability: port.Capability, Returns: port.Returns,
				Constraints: protocol.Canonicalize(constraints), Result: binding.Result,
			})
		}
		for _, record := range records {
			VerifyRecordStructure(state.Program.Closure, record)
		}
		return protocol.BuildFact{
			Format: "hypit.build-fact@1", Kind: "producer-applied", Command: event.Command,
			Step: command.Step, Records: records, Needs: needs,
		}
	}
	if command.Kind == "fulfill-need" && event.Kind == "need-fulfilled" {
		need, needOK := lookup.Need(command.Need.ID)
		invariant(needOK, "UNKNOWN_NEED", "unknown need "+command.Need.ID, command.Need.ID)
		_, exists := lookup.Record(need.Result)
		invariant(!exists, "NEED_ALREADY_FULFILLED", need.ID+" is already fulfilled", need.ID)
		record := protocol.TypedRecord{ID: need.Result, Type: need.Returns, Value: normalizeStoredValue(event.Value)}
		VerifyRecordStructure(state.Program.Closure, record)
		return protocol.BuildFact{
			Format: "hypit.build-fact@1", Kind: "need-applied", Command: event.Command, Need: need.ID, Record: record,
		}
	}
	panic(&protocol.SvmlError{
		Code: "EVENT_COMMAND_MISMATCH", Message: event.Kind + " cannot complete " + command.Kind, Subject: &command.ID,
	})
}

// AdmitBuildResult validates one untrusted result. The next view is provisional until fact is stored.
func AdmitBuildResult(state protocol.BuildState, event protocol.CommandResult) (protocol.BuildFact, protocol.BuildState) {
	fact := FactForBuildResult(state, event, nil)
	return fact, Reduce(state, &event)
}

func verifyFactShape(fact protocol.BuildFact) {
	invariant(fact.Format == "hypit.build-fact@1", "UNSUPPORTED_BUILD_FACT", fact.Command, fact.Command)
}

// MaterializeBuild reconstructs the view from one definition and its accepted facts.
func MaterializeBuild(definition protocol.BuildDefinition, facts []protocol.BuildFact) protocol.BuildState {
	initial := initialBuildView(definition)
	records := append([]protocol.TypedRecord{}, initial.Records...)
	steps := append([]protocol.StepState{}, initial.Steps...)
	var needs []protocol.Need
	var diagnostics []protocol.BuildDiagnostic
	stepIndexes := map[string]int{}
	for index, step := range steps {
		stepIndexes[step.ID] = index
	}
	recordIDs := map[string]struct{}{}
	for _, record := range records {
		recordIDs[record.ID] = struct{}{}
	}
	needIDs := map[string]struct{}{}
	status := "active"
	for _, fact := range facts {
		verifyFactShape(fact)
		invariant(status == "active", "FACT_AFTER_TERMINAL", fact.Command, fact.Command)
		if fact.Kind == "producer-applied" {
			stepIndex, ok := stepIndexes[fact.Step]
			invariant(ok, "UNKNOWN_STEP", fact.Step, fact.Step)
			invariant(steps[stepIndex].Status == "pending", "STEP_ALREADY_COMPLETE", fact.Step, fact.Step)
			planned := producerStep(definition.Plan, fact.Step)
			ready := true
			for _, record := range planned.Inputs {
				if _, present := recordIDs[record]; !present {
					ready = false
					break
				}
			}
			invariant(ready, "FACT_INPUT_NOT_READY", fact.Step, fact.Step)
			for _, record := range fact.Records {
				_, duplicate := recordIDs[record.ID]
				invariant(!duplicate, "DUPLICATE_RECORD", record.ID, record.ID)
				recordIDs[record.ID] = struct{}{}
				records = append(records, record)
			}
			for _, need := range fact.Needs {
				_, duplicate := needIDs[need.ID]
				invariant(!duplicate, "DUPLICATE_NEED", need.ID, need.ID)
				needIDs[need.ID] = struct{}{}
				needs = append(needs, need)
			}
			steps[stepIndex] = protocol.StepState{ID: fact.Step, Status: "complete"}
			continue
		}
		if fact.Kind == "need-applied" {
			_, known := needIDs[fact.Need]
			invariant(known, "UNKNOWN_NEED", fact.Need, fact.Need)
			_, duplicate := recordIDs[fact.Record.ID]
			invariant(!duplicate, "NEED_ALREADY_FULFILLED", fact.Need, fact.Need)
			recordIDs[fact.Record.ID] = struct{}{}
			records = append(records, fact.Record)
			continue
		}
		diagnostics = append(diagnostics, fact.Diagnostic)
		status = "failed"
	}
	if needs == nil {
		needs = []protocol.Need{}
	}
	if diagnostics == nil {
		diagnostics = []protocol.BuildDiagnostic{}
	}
	next := initial
	next.Status = status
	next.Records = records
	next.Steps = steps
	next.Needs = needs
	next.Diagnostics = diagnostics
	return schedule(next)
}

func cloneDefinition(input protocol.BuildDefinition) protocol.BuildDefinition {
	out := input
	out.Program.Records = append([]protocol.TypedRecord{}, input.Program.Records...)
	out.Program.Closure.Modules = append([]protocol.ResolvedModule{}, input.Program.Closure.Modules...)
	out.InitialRecords = append([]protocol.TypedRecord{}, input.InitialRecords...)
	out.Targets = append([]protocol.BuildTarget{}, input.Targets...)
	out.Plan.Goals = append([]protocol.BuildGoal{}, input.Plan.Goals...)
	out.Plan.OutputBindings = append([]protocol.BuildOutputBinding{}, input.Plan.OutputBindings...)
	out.Plan.Steps = make([]protocol.ProducerStep, len(input.Plan.Steps))
	for i, step := range input.Plan.Steps {
		step.Inputs = cloneStringMap(step.Inputs)
		step.Outputs = cloneStringMap(step.Outputs)
		needs := make(map[string]protocol.NeedBinding, len(step.Needs))
		for key, value := range step.Needs {
			needs[key] = value
		}
		step.Needs = needs
		out.Plan.Steps[i] = step
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
