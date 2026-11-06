// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package author is parser-independent author linking and static Graph Fragment elaboration outside SVML Core.
package author

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// AuthorRecordRef is a reference to an authored fact that already exists in the Linked Program.
type AuthorRecordRef struct {
	Kind string
	ID   string
}

// AuthorComponentOutputRef is a symbolic reference to one public export of another component instance.
type AuthorComponentOutputRef struct {
	Kind      string
	Component string
	Output    string
}

// AuthorValueRef is an authored record or a component output.
// Kind is "record" or "component-output".
type AuthorValueRef struct {
	Kind      string
	ID        string
	Component string
	Output    string
}

// AuthorComponent is one author-visible component call.
// Outputs binds every public Fragment export to a Logical Output.
type AuthorComponent struct {
	ID       string
	Fragment string
	Inputs   map[string]AuthorValueRef
	Outputs  map[string]string
}

// AuthorOutputBinding is one public export predeclared before inputs are resolved.
type AuthorOutputBinding struct {
	Component string
	Output    string
	ID        string
	Type      protocol.TypeRef
}

// GraphFragmentResolver resolves a selected Graph Fragment without executing package code.
// The boolean is false when the fragment is unavailable.
type GraphFragmentResolver func(id string) (GraphFragment, bool)

// AuthorGraphError is the coded failure thrown by author linking.
type AuthorGraphError struct {
	Code    string
	Message string
	Subject *string
	Name    string
}

func (e *AuthorGraphError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func authorGraphError(code, message string, subject []string) *AuthorGraphError {
	err := &AuthorGraphError{Code: code, Message: message, Name: "AuthorGraphError"}
	if len(subject) > 0 {
		value := subject[0]
		err.Subject = &value
	}
	return err
}

func assertAuthor(condition bool, code, message string, subject ...string) {
	if condition {
		return
	}
	panic(authorGraphError(code, message, subject))
}

func exactAuthorKeys[T any](actual map[string]T, expected []string, subject string) {
	left := mapKeys(actual)
	right := append([]string(nil), expected...)
	sortUTF16(left)
	sortUTF16(right)
	if slices.Equal(left, right) {
		return
	}
	assertAuthor(false, "AUTHOR_PORT_BINDING_MISMATCH", fmt.Sprintf("%s binds [%s] but declares [%s]", subject, strings.Join(left, ", "), strings.Join(right, ", ")), subject)
}

func normalizeRef(ref AuthorValueRef) AuthorValueRef {
	if ref.Kind == "record" {
		return AuthorValueRef{Kind: "record", ID: ref.ID}
	}
	assertAuthor(ref.Kind == "component-output", "INVALID_AUTHOR_REFERENCE", "Author inputs must reference a record or a component output")
	return AuthorValueRef{Kind: "component-output", Component: ref.Component, Output: ref.Output}
}

func normalizeComponent(component AuthorComponent) AuthorComponent {
	inputs := make(map[string]AuthorValueRef, len(component.Inputs))
	for _, name := range sortLocaleKeys(component.Inputs) {
		inputs[name] = normalizeRef(component.Inputs[name])
	}
	outputs := make(map[string]string, len(component.Outputs))
	for _, name := range sortLocaleKeys(component.Outputs) {
		outputs[name] = component.Outputs[name]
	}
	return AuthorComponent{
		ID:       component.ID,
		Fragment: component.Fragment,
		Inputs:   inputs,
		Outputs:  outputs,
	}
}

type collectedComponent struct {
	declaration AuthorComponent
	fragment    GraphFragment
}

type collectedAuthorGraph struct {
	components map[string]collectedComponent
	outputs    map[string]AuthorOutputBinding
	ordered    []collectedComponent
}

func outputKey(component, output string) string {
	return component + "\u0000" + output
}

// collectAuthorGraph is phase one: collect every component and predeclare every public output before resolving any input.
func collectAuthorGraph(program protocol.LinkedProgram, declarations []AuthorComponent, resolveFragment GraphFragmentResolver) collectedAuthorGraph {
	components := map[string]collectedComponent{}
	outputs := map[string]AuthorOutputBinding{}
	outputIDs := map[string]struct{}{}
	inserted := make([]collectedComponent, 0, len(declarations))

	for _, rawDeclaration := range declarations {
		declaration := normalizeComponent(rawDeclaration)
		assertAuthor(declaration.ID != "", "EMPTY_AUTHOR_COMPONENT_ID", "Author component id is empty")
		_, duplicate := components[declaration.ID]
		assertAuthor(!duplicate, "DUPLICATE_AUTHOR_COMPONENT", fmt.Sprintf("Author Graph repeats component %s", declaration.ID), declaration.ID)
		assertAuthor(jsTrim(declaration.Fragment) != "", "INVALID_FRAGMENT_ID", fmt.Sprintf("%s Fragment id is empty", declaration.ID))
		fragment, ok := resolveFragment(declaration.Fragment)
		assertAuthor(ok, "UNKNOWN_GRAPH_FRAGMENT", fmt.Sprintf("%s references unavailable Fragment %s", declaration.ID, declaration.Fragment), declaration.ID)
		assertAuthor(fragment.ID == declaration.Fragment, "FRAGMENT_RESOLUTION_MISMATCH", fmt.Sprintf("%s resolved a different Fragment", declaration.ID), declaration.ID)
		VerifyGraphFragment(program, fragment)
		inputNames := make([]string, len(fragment.Inputs))
		for i, input := range fragment.Inputs {
			inputNames[i] = input.Name
		}
		exactAuthorKeys(declaration.Inputs, inputNames, declaration.ID)
		exportNames := make([]string, len(fragment.Exports))
		for i, item := range fragment.Exports {
			exportNames[i] = item.Name
		}
		exactAuthorKeys(declaration.Outputs, exportNames, declaration.ID)
		collected := collectedComponent{declaration: declaration, fragment: fragment}
		components[declaration.ID] = collected
		inserted = append(inserted, collected)

		for _, item := range fragment.Exports {
			id := declaration.Outputs[item.Name]
			assertAuthor(id != "", "EMPTY_LOGICAL_OUTPUT_ID", fmt.Sprintf("%s.%s output is empty", declaration.ID, item.Name))
			_, duplicateID := outputIDs[id]
			assertAuthor(!duplicateID, "DUPLICATE_LOGICAL_OUTPUT_ID", fmt.Sprintf("Author Graph binds Logical Output %s more than once", id), id)
			outputIDs[id] = struct{}{}
			outputs[outputKey(declaration.ID, item.Name)] = AuthorOutputBinding{
				Component: declaration.ID,
				Output:    item.Name,
				ID:        id,
				Type:      item.Type,
			}
		}
	}
	ordered := append([]collectedComponent(nil), inserted...)
	slices.SortStableFunc(ordered, func(left, right collectedComponent) int {
		return protocol.LocaleCompare(left.declaration.ID, right.declaration.ID)
	})
	return collectedAuthorGraph{components: components, outputs: outputs, ordered: ordered}
}

// ElaborateAuthorGraph is phase two: resolve symbolic references, reject cycles and type errors, then elaborate ordinary Core data.
func ElaborateAuthorGraph(program protocol.LinkedProgram, components []AuthorComponent, resolveFragment GraphFragmentResolver) protocol.CompiledGraph {
	collected := collectAuthorGraph(program, components, resolveFragment)
	records := map[string]protocol.TypedRecord{}
	for _, record := range program.Records {
		records[record.ID] = record
	}
	visiting := []string{}
	visitingAt := map[string]int{}
	visited := map[string]struct{}{}
	ordered := make([]collectedComponent, 0, len(collected.ordered))

	var visit func(collectedComponent)
	visit = func(component collectedComponent) {
		id := component.declaration.ID
		if _, seen := visited[id]; seen {
			return
		}
		cycleStart, seen := visitingAt[id]
		if !seen {
			cycleStart = -1
		}
		if cycleStart != -1 {
			path := append(append([]string{}, visiting[cycleStart:]...), id)
			assertAuthor(false, "AUTHOR_COMPONENT_CYCLE", "Author components cycle through "+strings.Join(path, " -> "), id)
		}
		visitingAt[id] = len(visiting)
		visiting = append(visiting, id)
		for _, name := range sortLocaleKeys(component.declaration.Inputs) {
			ref := component.declaration.Inputs[name]
			if ref.Kind != "component-output" {
				continue
			}
			dependency, ok := collected.components[ref.Component]
			assertAuthor(ok, "UNKNOWN_AUTHOR_COMPONENT", fmt.Sprintf("%s references unknown component %s", id, ref.Component), id)
			visit(dependency)
		}
		visiting = visiting[:len(visiting)-1]
		delete(visitingAt, id)
		visited[id] = struct{}{}
		ordered = append(ordered, component)
	}
	for _, component := range collected.ordered {
		visit(component)
	}

	contributions := make([]FragmentContribution, 0, len(ordered))
	for _, component := range ordered {
		inputs := map[string]protocol.GraphValueRef{}
		for _, input := range component.fragment.Inputs {
			ref := component.declaration.Inputs[input.Name]
			if ref.Kind == "record" {
				record, ok := records[ref.ID]
				assertAuthor(ok, "UNKNOWN_AUTHOR_RECORD", fmt.Sprintf("%s.%s references unknown record %s", component.declaration.ID, input.Name, ref.ID), component.declaration.ID)
				assertAuthor(sameType(record.Type, input.Type), "AUTHOR_INPUT_TYPE_MISMATCH", fmt.Sprintf("%s.%s wants %s but receives %s", component.declaration.ID, input.Name, typeName(input.Type), typeName(record.Type)), component.declaration.ID)
				inputs[input.Name] = protocol.GraphValueRef{Kind: "record", ID: record.ID}
				continue
			}
			binding, ok := collected.outputs[outputKey(ref.Component, ref.Output)]
			assertAuthor(ok, "UNKNOWN_AUTHOR_OUTPUT", fmt.Sprintf("%s.%s references unknown output %s.%s", component.declaration.ID, input.Name, ref.Component, ref.Output), component.declaration.ID)
			assertAuthor(sameType(binding.Type, input.Type), "AUTHOR_INPUT_TYPE_MISMATCH", fmt.Sprintf("%s.%s wants %s but receives %s", component.declaration.ID, input.Name, typeName(input.Type), typeName(binding.Type)), component.declaration.ID)
			inputs[input.Name] = protocol.GraphValueRef{Kind: "logical-output", ID: binding.ID}
		}
		instance := ElaborateGraphFragment(program, component.fragment, FragmentInstanceRequest{
			ID:       "author:" + component.declaration.ID,
			Fragment: component.fragment.ID,
			Inputs:   inputs,
		})
		contributions = append(contributions, BindAuthorFragment(instance, component.declaration.Outputs))
	}

	merged := MergeFragmentContributions(protocol.CompiledGraph{
		Outputs:    []protocol.LogicalOutput{},
		Candidates: []protocol.Candidate{},
		Operations: []protocol.OperationNode{},
	}, contributions...)
	graph := kernel.SealCompiledGraph(merged)
	kernel.VerifyCompiledGraph(program, graph)
	return graph
}
