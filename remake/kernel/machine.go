// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel/src/machine.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import (
	"errors"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// BuildMachine is the non-serializable incremental Core machine.
// Definition plus Facts are the stored form. Indexes belong to this execution.
type BuildMachine struct {
	Definition protocol.BuildDefinition

	state         protocol.BuildState
	proposal      *protocol.BuildFact
	records       map[string]protocol.TypedRecord
	needs         map[string]protocol.Need
	commands      map[string]protocol.BuildCommand
	stepIndexes   map[string]int
	stepStatuses  map[string]string
	missingInputs map[string]int
	dependents    map[string][]string
	missingGoals  map[string]struct{}
}

// FromMaterialized adopts a view already materialized from this definition's facts.
func FromMaterialized(definition protocol.BuildDefinition, state protocol.BuildState) *BuildMachine {
	return newBuildMachine(definition, nil, &state)
}

// NewBuildMachine replays facts onto a definition.
func NewBuildMachine(definition protocol.BuildDefinition, facts []protocol.BuildFact) *BuildMachine {
	return newBuildMachine(definition, facts, nil)
}

func newBuildMachine(definition protocol.BuildDefinition, facts []protocol.BuildFact, materialized *protocol.BuildState) *BuildMachine {
	machine := &BuildMachine{
		Definition:    cloneDefinition(definition),
		records:       map[string]protocol.TypedRecord{},
		needs:         map[string]protocol.Need{},
		commands:      map[string]protocol.BuildCommand{},
		stepIndexes:   map[string]int{},
		stepStatuses:  map[string]string{},
		missingInputs: map[string]int{},
		dependents:    map[string][]string{},
		missingGoals:  map[string]struct{}{},
	}
	if materialized != nil {
		machine.state = *materialized
	} else {
		machine.state = MaterializeBuild(machine.Definition, facts)
	}
	invariant(machine.state.Format == "hypit.build@1", "UNSUPPORTED_BUILD", machine.state.Format)
	for _, record := range machine.state.Records {
		machine.records[record.ID] = record
	}
	for _, need := range machine.state.Needs {
		machine.needs[need.ID] = need
	}
	for _, command := range machine.state.Outstanding {
		machine.commands[command.ID] = command
	}
	for index, stepState := range machine.state.Steps {
		machine.stepIndexes[stepState.ID] = index
		machine.stepStatuses[stepState.ID] = stepState.Status
		if stepState.Status != "pending" {
			continue
		}
		step := producerStep(machine.Definition.Plan, stepState.ID)
		missing := map[string]struct{}{}
		for _, record := range step.Inputs {
			if _, ok := machine.records[record]; !ok {
				missing[record] = struct{}{}
			}
		}
		machine.missingInputs[step.ID] = len(missing)
		for record := range missing {
			machine.dependents[record] = append(machine.dependents[record], step.ID)
		}
	}
	for _, goal := range machine.Definition.Plan.Goals {
		if _, ok := machine.records[goal.Record]; !ok {
			machine.missingGoals[goal.Record] = struct{}{}
		}
	}
	return machine
}

func (m *BuildMachine) Status() string { return m.state.Status }

func (m *BuildMachine) Commands() []protocol.BuildCommand {
	return append([]protocol.BuildCommand{}, m.state.Outstanding...)
}

func (m *BuildMachine) Record(id string) (protocol.TypedRecord, bool) {
	record, ok := m.records[id]
	return record, ok
}

// View is the materialized read view. It is not the durable store representation.
func (m *BuildMachine) View() protocol.BuildState { return m.state }

// Evaluate validates one result without mutating the machine.
func (m *BuildMachine) Evaluate(result protocol.CommandResult) protocol.BuildFact {
	if m.proposal != nil {
		panic(errors.New("Build result " + m.proposal.Command + " has not been durably applied"))
	}
	fact := FactForBuildResult(m.state, result, &BuildAdmissionLookup{
		Command: func(id protocol.CommandId) (protocol.BuildCommand, bool) {
			command, ok := m.commands[id]
			return command, ok
		},
		StepStatus: func(id string) (string, bool) {
			status, ok := m.stepStatuses[id]
			return status, ok
		},
		Need: func(id string) (protocol.Need, bool) {
			need, ok := m.needs[id]
			return need, ok
		},
		Record: func(id string) (protocol.TypedRecord, bool) {
			record, ok := m.records[id]
			return record, ok
		},
	})
	m.proposal = &fact
	return fact
}

// Commit advances memory after the proposed fact has been stored.
func (m *BuildMachine) Commit() {
	if m.proposal == nil {
		panic(errors.New("Build machine has no evaluated result to commit"))
	}
	fact := *m.proposal
	m.proposal = nil
	_, ok := m.commands[fact.Command]
	invariant(ok, "UNKNOWN_COMMAND", "Fact references "+fact.Command, fact.Command)
	delete(m.commands, fact.Command)

	records := m.state.Records
	needs := m.state.Needs
	steps := m.state.Steps
	diagnostics := m.state.Diagnostics
	status := m.state.Status
	var added []protocol.BuildCommand

	if fact.Kind == "command-failed" {
		status = "failed"
		diagnostics = append(append([]protocol.BuildDiagnostic{}, diagnostics...), fact.Diagnostic)
		m.commands = map[string]protocol.BuildCommand{}
	} else {
		var addedRecords []protocol.TypedRecord
		if fact.Kind == "producer-applied" {
			addedRecords = fact.Records
			stepIndex, found := m.stepIndexes[fact.Step]
			invariant(found, "UNKNOWN_STEP", fact.Step, fact.Step)
			m.stepStatuses[fact.Step] = "complete"
			updated := append([]protocol.StepState{}, steps...)
			updated[stepIndex] = protocol.StepState{ID: fact.Step, Status: "complete"}
			steps = updated
			for _, need := range fact.Needs {
				m.needs[need.ID] = need
				next := buildNeedCommand(need)
				m.commands[next.ID] = next
				added = append(added, next)
			}
			if len(fact.Needs) > 0 {
				needs = append(append([]protocol.Need{}, needs...), fact.Needs...)
			}
		} else {
			addedRecords = []protocol.TypedRecord{fact.Record}
		}
		for _, record := range addedRecords {
			_, duplicate := m.records[record.ID]
			invariant(!duplicate, "DUPLICATE_RECORD", record.ID, record.ID)
			m.records[record.ID] = record
			delete(m.missingGoals, record.ID)
			for _, stepID := range m.dependents[record.ID] {
				remaining := m.missingInputs[stepID] - 1
				invariant(remaining >= 0, "STEP_INPUT_UNDERFLOW", stepID, stepID)
				m.missingInputs[stepID] = remaining
				if remaining == 0 && m.stepStatuses[stepID] == "pending" {
					next := buildProducerCommand(producerStep(m.Definition.Plan, stepID))
					m.commands[next.ID] = next
					added = append(added, next)
				}
			}
			delete(m.dependents, record.ID)
		}
		if len(addedRecords) > 0 {
			records = append(append([]protocol.TypedRecord{}, records...), addedRecords...)
		}
		if len(m.missingGoals) == 0 {
			status = "complete"
			m.commands = map[string]protocol.BuildCommand{}
		} else if len(m.commands) == 0 {
			status = "failed"
			diagnostics = append(append([]protocol.BuildDiagnostic{}, diagnostics...), protocol.BuildDiagnostic{
				Code:    "BUILD_DEADLOCK",
				Message: "no command can make progress toward the requested goals",
			})
		}
	}
	outstanding := []protocol.BuildCommand{}
	if status == "active" {
		outstanding = mergeCommands(m.state.Outstanding, fact.Command, added)
	}
	m.state = protocol.BuildState{
		Format: m.state.Format, Program: m.state.Program, Plan: m.state.Plan, Targets: m.state.Targets,
		Status: status, Records: records, Steps: steps, Needs: needs, Outstanding: outstanding, Diagnostics: diagnostics,
	}
}

func mergeCommands(current []protocol.BuildCommand, removed string, added []protocol.BuildCommand) []protocol.BuildCommand {
	retained := make([]protocol.BuildCommand, 0, len(current))
	for _, command := range current {
		if command.ID != removed {
			retained = append(retained, command)
		}
	}
	additions := append([]protocol.BuildCommand{}, added...)
	sortCommands(additions)
	merged := make([]protocol.BuildCommand, 0, len(retained)+len(additions))
	left, right := 0, 0
	for left < len(retained) || right < len(additions) {
		if right == len(additions) || (left < len(retained) && protocol.LocaleCompare(retained[left].ID, additions[right].ID) < 0) {
			merged = append(merged, retained[left])
			left++
			continue
		}
		merged = append(merged, additions[right])
		right++
	}
	return merged
}
