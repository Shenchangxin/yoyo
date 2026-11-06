// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime/src/scheduler.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

type mutableBuild struct {
	id        string
	state     protocol.BuildState
	machine   *kernel.BuildMachine
	outcomes  []SchedulerExecutionOutcome
	blocked   []RuntimeBlockedCommand
	stopped   bool
	failure   *protocol.CommandResult
	attempted map[string]struct{}
}

type activeResult struct {
	key       string
	build     *mutableBuild
	command   RuntimeRunnableCommand
	execution *RuntimeExecutionResult
	err       error
}

type activeCommand struct {
	build   *mutableBuild
	command RuntimeRunnableCommand
	done    chan activeResult
}

// InProcessBuildScheduler is the queue-free, one-process reference Scheduler.
// It admits every declared resource across submitted Builds and never searches a Graph.
type InProcessBuildScheduler struct {
	executor      RuntimeCommandExecutor
	buildStore    BuildStore
	onStateChange func(string, protocol.BuildState) error
}

// NewInProcessBuildScheduler builds a scheduler over executor.
func NewInProcessBuildScheduler(executor RuntimeCommandExecutor, options BuildSchedulerOptions) *InProcessBuildScheduler {
	return &InProcessBuildScheduler{executor: executor, buildStore: options.BuildStore, onStateChange: options.OnStateChange}
}

// Run schedules every request to completion, pause, or failure.
func (s *InProcessBuildScheduler) Run(requests []ScheduledBuild) []ScheduledBuildResult {
	ids := map[string]struct{}{}
	builds := make([]*mutableBuild, 0, len(requests))
	for _, request := range requests {
		if len(jsTrim(request.ID)) == 0 {
			panic(errors.New("scheduled build id must not be empty"))
		}
		if _, exists := ids[request.ID]; exists {
			panic(errors.New("scheduled build " + request.ID + " is duplicated"))
		}
		ids[request.ID] = struct{}{}
		var state protocol.BuildState
		var machine *kernel.BuildMachine
		if s.buildStore == nil {
			if request.State == nil {
				panic(errors.New("scheduled Build " + request.ID + " has no in-memory state"))
			}
			state = cloneState(*request.State)
		} else {
			snapshot := request.Snapshot
			if snapshot == nil {
				panic(errors.New("scheduled Build " + request.ID + " has no durable snapshot"))
			}
			if snapshot.Build != request.ID {
				panic(errors.New("scheduled Build snapshot " + snapshot.Build + " does not belong to " + request.ID))
			}
			state = snapshot.State
			machine = kernel.FromMaterialized(snapshot.Definition, snapshot.State)
		}
		builds = append(builds, &mutableBuild{
			id: request.ID, state: state, machine: machine,
			outcomes: []SchedulerExecutionOutcome{}, blocked: []RuntimeBlockedCommand{},
			attempted: map[string]struct{}{},
		})
	}
	active := map[string]*activeCommand{}
	resourceDefaults := map[string]int64{}
	cursor := 0
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}
		drainActive(active)
		panic(rec)
	}()

	resourceLimit := func(resource CapacityResourceClaim) int64 {
		CapacityUnits(resource)
		proposed := positiveInteger(resource.Limit, "resource "+resource.ID+" limit")
		if previous, ok := resourceDefaults[resource.ID]; ok && previous != proposed {
			panic(errors.New("resource " + resource.ID + " has conflicting default concurrency " + strconv.FormatInt(previous, 10) + " and " + strconv.FormatInt(proposed, 10)))
		}
		resourceDefaults[resource.ID] = proposed
		return proposed
	}
	accept := func(build *mutableBuild, event protocol.CommandResult) {
		if s.buildStore == nil || build.machine == nil {
			build.state = kernel.Reduce(build.state, &event)
			if s.onStateChange != nil {
				if err := s.onStateChange(build.id, build.state); err != nil {
					panic(err)
				}
			}
			return
		}
		fact := build.machine.Evaluate(event)
		if err := s.buildStore.Append(build.id, fact); err != nil {
			panic(err)
		}
		build.machine.Commit()
		build.state = build.machine.View()
		if s.onStateChange != nil {
			if err := s.onStateChange(build.id, build.state); err != nil {
				panic(err)
			}
		}
	}

	for {
		ready := map[string][]RuntimeRunnableCommand{}
		for _, build := range builds {
			if build.stopped || build.state.Status == "complete" || build.state.Status == "failed" {
				ready[build.id] = []RuntimeRunnableCommand{}
				continue
			}
			var prepared RuntimePreparation
			if build.machine != nil && s.executor.PrepareExecution != nil {
				prepared = s.executor.PrepareExecution(build.machine)
			} else {
				if s.executor.Prepare == nil {
					panic(errors.New("executor prepare is not a function"))
				}
				prepared = s.executor.Prepare(build.state)
			}
			build.state = prepared.State
			build.blocked = prepared.Blocked
			runnable := make([]RuntimeRunnableCommand, 0)
			for _, item := range prepared.Runnable {
				if _, running := active[buildCommandKey(build.id, string(item.Command.ID))]; running {
					continue
				}
				if _, seen := build.attempted[string(item.Command.ID)]; seen {
					continue
				}
				runnable = append(runnable, item)
			}
			ready[build.id] = runnable
		}
		counts := map[string]int64{}
		for _, item := range active {
			for _, resource := range item.command.Resources {
				counts[resource.ID] += CapacityUnits(resource)
			}
		}
		for len(builds) > 0 {
			selected := false
			for offset := 0; offset < len(builds); offset++ {
				index := (cursor + offset) % len(builds)
				build := builds[index]
				var command *RuntimeRunnableCommand
				for i := range ready[build.id] {
					candidate := ready[build.id][i]
					fits := true
					for _, resource := range candidate.Resources {
						if counts[resource.ID]+CapacityUnits(resource) > resourceLimit(resource) {
							fits = false
							break
						}
					}
					if fits {
						chosen := candidate
						command = &chosen
						break
					}
				}
				if command == nil {
					continue
				}
				filtered := make([]RuntimeRunnableCommand, 0)
				for _, item := range ready[build.id] {
					if item.Command.ID != command.Command.ID {
						filtered = append(filtered, item)
					}
				}
				ready[build.id] = filtered
				launch(active, build, *command, s.executor)
				for _, resource := range command.Resources {
					counts[resource.ID] += CapacityUnits(resource)
				}
				cursor = (index + 1) % len(builds)
				selected = true
				break
			}
			if !selected {
				break
			}
		}
		if len(active) == 0 {
			break
		}
		settled, key := raceActive(active)
		delete(active, key)
		build := settled.build
		if build.state.Status == "failed" || build.state.Status == "complete" {
			continue
		}
		if settled.execution == nil {
			build.stopped = true
			build.outcomes = append(build.outcomes, outcome(settled.command, "error", "", nil, errorMessage(settled.err)))
			continue
		}
		switch settled.execution.Status {
		case "pending":
			build.attempted[string(settled.command.Command.ID)] = struct{}{}
			build.outcomes = append(build.outcomes, outcome(settled.command, "pending", settled.execution.Operation, settled.execution.WakeAt, ""))
		case "deferred":
			build.attempted[string(settled.command.Command.ID)] = struct{}{}
			build.outcomes = append(build.outcomes, outcome(settled.command, "deferred", "", settled.execution.WakeAt, settled.execution.Reason))
		default:
			event := settled.execution.Event
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						build.stopped = true
						build.outcomes = append(build.outcomes, outcome(settled.command, "error", "", nil, errorMessage(rec)))
					}
				}()
				if event.Kind == "command-failed" {
					build.stopped = true
					if build.failure == nil {
						failed := event
						build.failure = &failed
					}
				} else {
					accept(build, event)
				}
				build.outcomes = append(build.outcomes, outcome(settled.command, "completed", "", nil, ""))
			}()
		}
	}
	for _, build := range builds {
		if build.failure != nil {
			accept(build, *build.failure)
		}
	}
	results := make([]ScheduledBuildResult, len(builds))
	for i, build := range builds {
		status := "paused"
		if build.state.Status == "complete" {
			status = "complete"
		} else if build.state.Status == "failed" {
			status = "failed"
		}
		results[i] = ScheduledBuildResult{
			ID: build.id, Status: status, State: build.state,
			Outcomes: append([]SchedulerExecutionOutcome{}, build.outcomes...),
			Blocked:  append([]RuntimeBlockedCommand{}, build.blocked...),
		}
	}
	return results
}

func launch(active map[string]*activeCommand, build *mutableBuild, command RuntimeRunnableCommand, executor RuntimeCommandExecutor) {
	key := buildCommandKey(build.id, string(command.Command.ID))
	startedFrom := build.state
	done := make(chan activeResult, 1)
	active[key] = &activeCommand{build: build, command: command, done: done}
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				done <- activeResult{key: key, build: build, command: command, err: asError(rec)}
			}
		}()
		var execution RuntimeExecutionResult
		var err error
		context := RuntimeExecutionContext{Build: build.id}
		if build.machine != nil && executor.ExecuteExecutionCommand != nil {
			execution, err = executor.ExecuteExecutionCommand(build.machine, command, context)
		} else if executor.ExecuteCommand != nil {
			execution, err = executor.ExecuteCommand(startedFrom, command, context)
		} else {
			err = errors.New("executor executeCommand is not a function")
		}
		if err != nil {
			done <- activeResult{key: key, build: build, command: command, err: err}
			return
		}
		done <- activeResult{key: key, build: build, command: command, execution: &execution}
	}()
}

func raceActive(active map[string]*activeCommand) (activeResult, string) {
	cases := make([]reflect.SelectCase, 0, len(active))
	keys := make([]string, 0, len(active))
	for key, item := range active {
		keys = append(keys, key)
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(item.done)})
	}
	index, value, _ := reflect.Select(cases)
	return value.Interface().(activeResult), keys[index]
}

func drainActive(active map[string]*activeCommand) {
	for len(active) > 0 {
		_, key := raceActive(active)
		delete(active, key)
	}
}

func outcome(command RuntimeRunnableCommand, status, operation string, wakeAt *float64, message string) SchedulerExecutionOutcome {
	resources := make([]string, len(command.Resources))
	for i, resource := range command.Resources {
		resources[i] = resource.ID
	}
	return SchedulerExecutionOutcome{
		Command: string(command.Command.ID), Kind: command.Command.Kind, Resources: resources,
		Status: status, Operation: operation, WakeAt: wakeAt, Message: message,
	}
}

func buildCommandKey(build, command string) string {
	return build + "\x00" + command
}

func positiveInteger(value int64, name string) int64 {
	if !safeInteger(value) || value < 1 {
		panic(errors.New(name + " must be a positive safe integer"))
	}
	return value
}

func cloneState(state protocol.BuildState) protocol.BuildState {
	state.Records = append([]protocol.TypedRecord{}, state.Records...)
	state.Steps = append([]protocol.StepState{}, state.Steps...)
	state.Needs = append([]protocol.Need{}, state.Needs...)
	state.Outstanding = append([]protocol.BuildCommand{}, state.Outstanding...)
	state.Diagnostics = append([]protocol.BuildDiagnostic{}, state.Diagnostics...)
	state.Targets = append([]protocol.BuildTarget{}, state.Targets...)
	return state
}

func errorMessage(value any) string {
	switch item := value.(type) {
	case error:
		return item.Error()
	case string:
		return item
	default:
		return fmt.Sprint(value)
	}
}

func asError(value any) error {
	if err, ok := value.(error); ok {
		return err
	}
	return errors.New(errorMessage(value))
}
