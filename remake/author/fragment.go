// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package author

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// FragmentInputRef names one declared input of a Graph Fragment.
type FragmentInputRef struct {
	Kind string
	Name string
}

// FragmentOperationRef names one local Operation of a Graph Fragment.
type FragmentOperationRef struct {
	Kind      string
	Operation string
}

// FragmentValueRef is a declared input or a local Operation.
// Kind is "fragment-input" or "fragment-operation".
type FragmentValueRef struct {
	Kind      string
	Name      string
	Operation string
}

// FragmentOperationResult is a producer output or a requested Need.
// Kind is "output" or "need".
type FragmentOperationResult struct {
	Kind string
	Name string
}

// FragmentOperation is one local Operation inside a Graph Fragment.
type FragmentOperation struct {
	ID       string
	Producer protocol.ProducerRef
	Inputs   map[string]FragmentValueRef
	Result   FragmentOperationResult
}

// FragmentExport publishes one local Operation result.
type FragmentExport struct {
	Name string
	Type protocol.TypeRef
	Root FragmentValueRef
}

// FragmentInput is one declared input port.
type FragmentInput struct {
	Name string
	Type protocol.TypeRef
}

// GraphFragment is a sealed hypit.fragment@1 value.
type GraphFragment struct {
	Format     string
	ID         string
	Inputs     []FragmentInput
	Operations []FragmentOperation
	Exports    []FragmentExport
}

// FragmentInstanceRequest elaborates one fragment at one author instance.
type FragmentInstanceRequest struct {
	// ID is the stable author instance identity, not a content-deduplication key.
	ID       string
	Fragment string
	Inputs   map[string]protocol.GraphValueRef
}

// ElaboratedFragmentExport is one export after hygienic expansion.
type ElaboratedFragmentExport struct {
	Name string
	Type protocol.TypeRef
	Root protocol.CandidateRoot
}

// ElaboratedFragment is a sealed hypit.fragment-instance@1 value.
type ElaboratedFragment struct {
	Format     string
	ID         string
	Fragment   string
	Instance   string
	Inputs     map[string]protocol.GraphValueRef
	Operations []protocol.OperationNode
	Exports    []ElaboratedFragmentExport
}

// FragmentContribution is graph data emitted by one elaborated instance.
// Satisfactions are explicit Run-Graph satisfactions. Author contributions leave this nil.
type FragmentContribution struct {
	Outputs       []protocol.LogicalOutput
	Candidates    []protocol.Candidate
	Operations    []protocol.OperationNode
	Satisfactions []protocol.Satisfaction
}

// RunFragmentExport names one Run Candidate without an Author Logical Output.
type RunFragmentExport struct {
	Name      string
	Candidate string
	Type      protocol.TypeRef
}

// RunFragmentContribution is a FragmentContribution plus Run export names.
type RunFragmentContribution struct {
	FragmentContribution
	Exports []RunFragmentExport
}

// FragmentError is the coded failure thrown by Fragment elaboration.
type FragmentError struct {
	Code    string
	Message string
	Subject *string
	Name    string
}

func (e *FragmentError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func fragmentError(code, message string, subject []string) *FragmentError {
	err := &FragmentError{Code: code, Message: message, Name: "FragmentError"}
	if len(subject) > 0 {
		value := subject[0]
		err.Subject = &value
	}
	return err
}

func assertFragment(condition bool, code, message string, subject ...string) {
	if condition {
		return
	}
	panic(fragmentError(code, message, subject))
}

func sameType(left, right protocol.TypeRef) bool {
	return left.Module.Name == right.Module.Name &&
		left.Module.Version == right.Module.Version &&
		left.Name == right.Name
}

func typeName(ref protocol.TypeRef) string {
	return ref.Module.Name + "@" + ref.Module.Version + "#" + ref.Name
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func sortLocaleKeys[T any](values map[string]T) []string {
	keys := mapKeys(values)
	sort.SliceStable(keys, func(i, j int) bool {
		return protocol.LocaleCompare(keys[i], keys[j]) < 0
	})
	return keys
}

// sortUTF16 is Array.prototype.sort() with no comparator: UTF-16 code units.
func sortUTF16(values []string) {
	sort.SliceStable(values, func(i, j int) bool {
		return utf16Less(values[i], values[j])
	})
}

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

func exactFragmentKeys[T any](actual map[string]T, expected []string, subject string) {
	left := mapKeys(actual)
	right := append([]string(nil), expected...)
	sortUTF16(left)
	sortUTF16(right)
	if slices.Equal(left, right) {
		return
	}
	assertFragment(false, "FRAGMENT_PORT_BINDING_MISMATCH", fmt.Sprintf("%s binds [%s] but declares [%s]", subject, strings.Join(left, ", "), strings.Join(right, ", ")), subject)
}

func normalizeFragmentRef(ref FragmentValueRef) FragmentValueRef {
	if ref.Kind == "fragment-input" {
		return FragmentValueRef{Kind: "fragment-input", Name: ref.Name}
	}
	assertFragment(ref.Kind == "fragment-operation", "INVALID_FRAGMENT_REFERENCE", "Fragment references must name a declared input or local Operation")
	return FragmentValueRef{Kind: "fragment-operation", Operation: ref.Operation}
}

func normalizeOperation(operation FragmentOperation) FragmentOperation {
	inputs := make(map[string]FragmentValueRef, len(operation.Inputs))
	for _, name := range sortLocaleKeys(operation.Inputs) {
		inputs[name] = normalizeFragmentRef(operation.Inputs[name])
	}
	result := FragmentOperationResult{Kind: "need", Name: operation.Result.Name}
	if operation.Result.Kind == "output" {
		result = FragmentOperationResult{Kind: "output", Name: operation.Result.Name}
	}
	return FragmentOperation{
		ID:       operation.ID,
		Producer: operation.Producer,
		Inputs:   inputs,
		Result:   result,
	}
}

func normalizeExport(item FragmentExport) FragmentExport {
	return FragmentExport{
		Name: item.Name,
		Type: item.Type,
		Root: normalizeFragmentRef(item.Root),
	}
}

func fragmentContent(fragment GraphFragment) GraphFragment {
	inputs := make([]FragmentInput, len(fragment.Inputs))
	for i, input := range fragment.Inputs {
		inputs[i] = FragmentInput{Name: input.Name, Type: input.Type}
	}
	sort.SliceStable(inputs, func(i, j int) bool {
		return protocol.LocaleCompare(inputs[i].Name, inputs[j].Name) < 0
	})
	operations := make([]FragmentOperation, len(fragment.Operations))
	for i, operation := range fragment.Operations {
		operations[i] = normalizeOperation(operation)
	}
	sort.SliceStable(operations, func(i, j int) bool {
		return protocol.LocaleCompare(operations[i].ID, operations[j].ID) < 0
	})
	exports := make([]FragmentExport, len(fragment.Exports))
	for i, item := range fragment.Exports {
		exports[i] = normalizeExport(item)
	}
	sort.SliceStable(exports, func(i, j int) bool {
		return protocol.LocaleCompare(exports[i].Name, exports[j].Name) < 0
	})
	return GraphFragment{
		Format:     "hypit.fragment@1",
		Inputs:     inputs,
		Operations: operations,
		Exports:    exports,
	}
}

// SealGraphFragment normalizes a fragment and assigns its content id.
func SealGraphFragment(fragment GraphFragment) GraphFragment {
	content := fragmentContent(fragment)
	producers := make([]string, len(content.Operations))
	for i, operation := range content.Operations {
		producers[i] = operation.Producer.Module.Name + "@" + operation.Producer.Module.Version + "#" + operation.Producer.Name + ":" + operation.ID
	}
	exports := make([]string, len(content.Exports))
	for i, item := range content.Exports {
		exports[i] = item.Name
	}
	content.ID = "fragment:" + strings.Join(producers, "+") + "=>" + strings.Join(exports, "+")
	return content
}

func resultType(program protocol.LinkedProgram, operation FragmentOperation) protocol.TypeRef {
	producer := kernel.ResolveProducer(program.Closure, operation.Producer)
	if operation.Result.Kind == "output" {
		found := false
		var output protocol.PortDeclaration
		for _, port := range producer.Outputs {
			if port.Name == operation.Result.Name {
				output = port
				found = true
				break
			}
		}
		assertFragment(found && len(producer.Needs) == 0, "FRAGMENT_RESULT_MISMATCH", fmt.Sprintf("%s does not produce output %s", operation.ID, operation.Result.Name), operation.ID)
		return output.Type
	}
	found := false
	var need protocol.NeedPortDeclaration
	for _, port := range producer.Needs {
		if port.Name == operation.Result.Name {
			need = port
			found = true
			break
		}
	}
	assertFragment(found && len(producer.Outputs) == 0, "FRAGMENT_RESULT_MISMATCH", fmt.Sprintf("%s does not request Need %s", operation.ID, operation.Result.Name), operation.ID)
	return need.Returns
}

// VerifyGraphFragment checks producer ports, types, cycles, and reachable exports.
func VerifyGraphFragment(program protocol.LinkedProgram, fragment GraphFragment) {
	assertFragment(fragment.Format == "hypit.fragment@1", "UNSUPPORTED_FRAGMENT", "unsupported Graph Fragment format")
	assertFragment(jsTrim(fragment.ID) != "", "INVALID_FRAGMENT_ID", "Graph Fragment id is empty")
	assertFragment(len(fragment.Exports) > 0, "EMPTY_FRAGMENT_EXPORTS", fmt.Sprintf("%s has no exports", fragment.ID))

	inputs := map[string]protocol.TypeRef{}
	for _, input := range fragment.Inputs {
		assertFragment(input.Name != "", "EMPTY_FRAGMENT_INPUT", fmt.Sprintf("%s has an empty input name", fragment.ID))
		_, duplicate := inputs[input.Name]
		assertFragment(!duplicate, "DUPLICATE_FRAGMENT_INPUT", fmt.Sprintf("%s repeats input %s", fragment.ID, input.Name))
		inputs[input.Name] = input.Type
	}

	operations := map[string]FragmentOperation{}
	for _, operation := range fragment.Operations {
		assertFragment(operation.ID != "", "EMPTY_FRAGMENT_OPERATION", fmt.Sprintf("%s has an empty Operation id", fragment.ID))
		_, duplicate := operations[operation.ID]
		assertFragment(!duplicate, "DUPLICATE_FRAGMENT_OPERATION", fmt.Sprintf("%s repeats %s", fragment.ID, operation.ID))
		operations[operation.ID] = operation
		producer := kernel.ResolveProducer(program.Closure, operation.Producer)
		portNames := make([]string, len(producer.Inputs))
		for i, port := range producer.Inputs {
			portNames[i] = port.Name
		}
		exactFragmentKeys(operation.Inputs, portNames, fragment.ID+"."+operation.ID)
	}

	visiting := map[string]struct{}{}
	visited := map[string]struct{}{}
	var operationType func(string) protocol.TypeRef
	operationType = func(id string) protocol.TypeRef {
		operation, ok := operations[id]
		assertFragment(ok, "UNKNOWN_FRAGMENT_OPERATION", fmt.Sprintf("%s references %s", fragment.ID, id), id)
		if _, seen := visited[id]; seen {
			return resultType(program, operation)
		}
		_, inStack := visiting[id]
		assertFragment(!inStack, "FRAGMENT_CYCLE", fmt.Sprintf("%s cycles through %s", fragment.ID, id), id)
		visiting[id] = struct{}{}
		producer := kernel.ResolveProducer(program.Closure, operation.Producer)
		for _, port := range producer.Inputs {
			ref, bound := operation.Inputs[port.Name]
			assertFragment(bound, "FRAGMENT_PORT_BINDING_MISMATCH", fmt.Sprintf("%s.%s is unbound", operation.ID, port.Name))
			var supplied protocol.TypeRef
			suppliedOK := false
			if ref.Kind == "fragment-input" {
				supplied, suppliedOK = inputs[ref.Name]
			} else {
				supplied = operationType(ref.Operation)
				suppliedOK = true
			}
			label := ref.Operation
			if ref.Kind == "fragment-input" {
				label = ref.Name
			}
			assertFragment(suppliedOK, "UNKNOWN_FRAGMENT_INPUT", fmt.Sprintf("%s references input %s", fragment.ID, label))
			assertFragment(sameType(supplied, port.Type), "FRAGMENT_INPUT_TYPE_MISMATCH", fmt.Sprintf("%s.%s wants %s but receives %s", operation.ID, port.Name, typeName(port.Type), typeName(supplied)), operation.ID)
		}
		delete(visiting, id)
		visited[id] = struct{}{}
		return resultType(program, operation)
	}

	exportNames := map[string]struct{}{}
	reachableOperations := map[string]struct{}{}
	var collectDependencies func(FragmentValueRef)
	collectDependencies = func(ref FragmentValueRef) {
		if ref.Kind == "fragment-input" {
			return
		}
		if _, seen := reachableOperations[ref.Operation]; seen {
			return
		}
		reachableOperations[ref.Operation] = struct{}{}
		operation, ok := operations[ref.Operation]
		assertFragment(ok, "UNKNOWN_FRAGMENT_OPERATION", fmt.Sprintf("%s references %s", fragment.ID, ref.Operation))
		for _, name := range sortLocaleKeys(operation.Inputs) {
			collectDependencies(operation.Inputs[name])
		}
	}

	for _, item := range fragment.Exports {
		assertFragment(item.Name != "", "EMPTY_FRAGMENT_EXPORT", fmt.Sprintf("%s has an empty export name", fragment.ID))
		_, duplicate := exportNames[item.Name]
		assertFragment(!duplicate, "DUPLICATE_FRAGMENT_EXPORT", fmt.Sprintf("%s repeats export %s", fragment.ID, item.Name))
		exportNames[item.Name] = struct{}{}
		assertFragment(item.Root.Kind == "fragment-operation", "INVALID_FRAGMENT_EXPORT_ROOT", fmt.Sprintf("%s.%s must export a local Operation result", fragment.ID, item.Name))
		supplied := operationType(item.Root.Operation)
		assertFragment(sameType(supplied, item.Type), "FRAGMENT_EXPORT_TYPE_MISMATCH", fmt.Sprintf("%s.%s declares %s but returns %s", fragment.ID, item.Name, typeName(item.Type), typeName(supplied)))
		collectDependencies(item.Root)
	}
	for _, operation := range fragment.Operations {
		_, ok := reachableOperations[operation.ID]
		assertFragment(ok, "UNREACHABLE_FRAGMENT_OPERATION", fmt.Sprintf("%s.%s contributes to no export", fragment.ID, operation.ID), operation.ID)
	}
}

func normalizeGraphRef(ref protocol.GraphValueRef) protocol.GraphValueRef {
	if ref.Kind == "record" {
		return protocol.GraphValueRef{Kind: "record", ID: ref.ID}
	}
	if ref.Kind == "logical-output" {
		return protocol.GraphValueRef{Kind: "logical-output", ID: ref.ID}
	}
	assertFragment(ref.Kind == "operation-result", "INVALID_FRAGMENT_BINDING", "Fragment inputs must bind to a Graph Value reference")
	return protocol.GraphValueRef{Kind: "operation-result", Operation: ref.Operation}
}

func hygienicID(kind, instance, local string) string {
	return kind + ":" + encodeURIComponent(instance) + ":" + encodeURIComponent(local)
}

// ElaborateGraphFragment verifies a fragment and assigns hygienic Operation identities.
func ElaborateGraphFragment(program protocol.LinkedProgram, fragment GraphFragment, request FragmentInstanceRequest) ElaboratedFragment {
	VerifyGraphFragment(program, fragment)
	assertFragment(request.ID != "", "EMPTY_FRAGMENT_INSTANCE", "Fragment instance id is empty")
	assertFragment(request.Fragment == fragment.ID, "FRAGMENT_INSTANCE_MISMATCH", fmt.Sprintf("%s locks another Fragment", request.ID))
	inputNames := make([]string, len(fragment.Inputs))
	for i, input := range fragment.Inputs {
		inputNames[i] = input.Name
	}
	exactFragmentKeys(request.Inputs, inputNames, request.ID)
	inputs := make(map[string]protocol.GraphValueRef, len(request.Inputs))
	for _, name := range sortLocaleKeys(request.Inputs) {
		inputs[name] = normalizeGraphRef(request.Inputs[name])
	}
	operationIDs := map[string]string{}
	for _, operation := range fragment.Operations {
		operationIDs[operation.ID] = hygienicID("operation", request.ID, operation.ID)
	}
	mapRef := func(ref FragmentValueRef) protocol.GraphValueRef {
		if ref.Kind == "fragment-input" {
			bound, ok := inputs[ref.Name]
			assertFragment(ok, "UNKNOWN_FRAGMENT_INPUT", fmt.Sprintf("%s does not bind %s", request.ID, ref.Name))
			return bound
		}
		operation, ok := operationIDs[ref.Operation]
		assertFragment(ok, "UNKNOWN_FRAGMENT_OPERATION", fmt.Sprintf("%s references %s", fragment.ID, ref.Operation))
		return protocol.GraphValueRef{Kind: "operation-result", Operation: operation}
	}
	operations := make([]protocol.OperationNode, len(fragment.Operations))
	for i, operation := range fragment.Operations {
		id := operationIDs[operation.ID]
		record := hygienicID("record", request.ID, operation.ID)
		node := protocol.OperationNode{
			ID:       id,
			Producer: operation.Producer,
			Inputs:   map[string]protocol.GraphValueRef{},
		}
		for _, name := range sortLocaleKeys(operation.Inputs) {
			node.Inputs[name] = mapRef(operation.Inputs[name])
		}
		if operation.Result.Kind == "output" {
			node.Result = protocol.OperationResult{Kind: "output", Name: operation.Result.Name, Record: record}
		} else {
			node.Result = protocol.OperationResult{
				Kind:   "need",
				Name:   operation.Result.Name,
				ID:     hygienicID("need", request.ID, operation.ID),
				Record: record,
			}
		}
		operations[i] = node
	}
	exports := make([]ElaboratedFragmentExport, len(fragment.Exports))
	for i, item := range fragment.Exports {
		root := mapRef(item.Root)
		assertFragment(root.Kind == "operation-result", "INVALID_FRAGMENT_EXPORT_ROOT", item.Name)
		exports[i] = ElaboratedFragmentExport{
			Name: item.Name,
			Type: item.Type,
			Root: protocol.CandidateRoot{
				Kind:   "operation",
				Result: protocol.OperationResultRef{Kind: "operation-result", Operation: root.Operation},
			},
		}
	}
	return ElaboratedFragment{
		Format:     "hypit.fragment-instance@1",
		ID:         "fragment-instance:" + request.ID,
		Fragment:   fragment.ID,
		Instance:   request.ID,
		Inputs:     inputs,
		Operations: operations,
		Exports:    exports,
	}
}

func exportMap(instance ElaboratedFragment) map[string]ElaboratedFragmentExport {
	available := make(map[string]ElaboratedFragmentExport, len(instance.Exports))
	for _, item := range instance.Exports {
		available[item.Name] = item
	}
	return available
}

func reachableOperationIDs(instance ElaboratedFragment, selected []ElaboratedFragmentExport) map[string]struct{} {
	operations := make(map[string]protocol.OperationNode, len(instance.Operations))
	for _, operation := range instance.Operations {
		operations[operation.ID] = operation
	}
	reachable := map[string]struct{}{}
	var visit func(string)
	visit = func(id string) {
		if _, seen := reachable[id]; seen {
			return
		}
		reachable[id] = struct{}{}
		operation, ok := operations[id]
		assertFragment(ok, "UNKNOWN_ELABORATED_OPERATION", fmt.Sprintf("%s references %s", instance.Instance, id))
		for _, name := range sortLocaleKeys(operation.Inputs) {
			ref := operation.Inputs[name]
			if ref.Kind == "operation-result" {
				visit(ref.Operation)
			}
		}
	}
	for _, item := range selected {
		assertFragment(item.Root.Kind == "operation", "INVALID_FRAGMENT_EXPORT_ROOT", item.Name)
		visit(item.Root.Result.Operation)
	}
	return reachable
}

func filterOperations(operations []protocol.OperationNode, reachable map[string]struct{}) []protocol.OperationNode {
	filtered := make([]protocol.OperationNode, 0)
	for _, operation := range operations {
		if _, ok := reachable[operation.ID]; ok {
			filtered = append(filtered, operation)
		}
	}
	return filtered
}

func bindExports(instance ElaboratedFragment, bindings map[string]string, requireAll bool) FragmentContribution {
	available := exportMap(instance)
	if requireAll {
		names := make([]string, len(instance.Exports))
		for i, item := range instance.Exports {
			names[i] = item.Name
		}
		exactFragmentKeys(bindings, names, instance.Instance)
	}
	assertFragment(len(bindings) > 0, "EMPTY_FRAGMENT_BINDING", fmt.Sprintf("%s binds no exports", instance.Instance))
	selected := make([]ElaboratedFragmentExport, 0, len(bindings))
	for _, name := range sortLocaleKeys(bindings) {
		item, ok := available[name]
		assertFragment(ok, "UNKNOWN_FRAGMENT_EXPORT", fmt.Sprintf("%s has no export %s", instance.Instance, name))
		selected = append(selected, item)
	}
	candidates := make([]protocol.Candidate, len(selected))
	for i, item := range selected {
		output := bindings[item.Name]
		assertFragment(output != "", "EMPTY_LOGICAL_OUTPUT_ID", fmt.Sprintf("%s.%s output is empty", instance.Instance, item.Name))
		candidates[i] = protocol.Candidate{
			ID:   hygienicID("candidate", instance.Instance, item.Name),
			Type: item.Type,
			Root: item.Root,
		}
	}
	outputs := []protocol.LogicalOutput{}
	var satisfactions []protocol.Satisfaction
	if requireAll {
		outputs = make([]protocol.LogicalOutput, len(selected))
		for i, item := range selected {
			outputs[i] = protocol.LogicalOutput{
				ID:      bindings[item.Name],
				Type:    item.Type,
				Primary: candidates[i].ID,
			}
		}
	} else {
		satisfactions = make([]protocol.Satisfaction, len(selected))
		for i, item := range selected {
			satisfactions[i] = protocol.Satisfaction{
				Output:    bindings[item.Name],
				Candidate: candidates[i].ID,
			}
		}
	}
	reachable := reachableOperationIDs(instance, selected)
	return FragmentContribution{
		Outputs:       outputs,
		Candidates:    candidates,
		Operations:    filterOperations(instance.Operations, reachable),
		Satisfactions: satisfactions,
	}
}

// ExportRunFragment instantiates a Run-Graph Fragment without deciding which Author Logical Outputs it will satisfy.
// Multiple exports share the instance's internal Operations; separate calls always create separate
// instances because identity comes from the already elaborated instance path.
// A nil names slice is the omitted TypeScript argument and exports every name.
func ExportRunFragment(instance ElaboratedFragment, names []string) RunFragmentContribution {
	if names == nil {
		names = make([]string, len(instance.Exports))
		for i, item := range instance.Exports {
			names[i] = item.Name
		}
	}
	assertFragment(len(names) > 0, "EMPTY_FRAGMENT_BINDING", fmt.Sprintf("%s exports no Run values", instance.Instance))
	seen := map[string]struct{}{}
	for _, name := range names {
		_, duplicate := seen[name]
		assertFragment(!duplicate, "DUPLICATE_FRAGMENT_EXPORT", fmt.Sprintf("%s repeats a Run export", instance.Instance))
		seen[name] = struct{}{}
	}
	available := exportMap(instance)
	selected := make([]ElaboratedFragmentExport, len(names))
	for i, name := range names {
		item, ok := available[name]
		assertFragment(ok, "UNKNOWN_FRAGMENT_EXPORT", fmt.Sprintf("%s has no export %s", instance.Instance, name))
		selected[i] = item
	}
	candidates := make([]protocol.Candidate, len(selected))
	for i, item := range selected {
		candidates[i] = protocol.Candidate{
			ID:   hygienicID("candidate", instance.Instance, item.Name),
			Type: item.Type,
			Root: item.Root,
		}
	}
	exports := make([]RunFragmentExport, len(selected))
	for i, item := range selected {
		exports[i] = RunFragmentExport{
			Name:      item.Name,
			Candidate: candidates[i].ID,
			Type:      item.Type,
		}
	}
	reachable := reachableOperationIDs(instance, selected)
	return RunFragmentContribution{
		FragmentContribution: FragmentContribution{
			Outputs:    []protocol.LogicalOutput{},
			Candidates: candidates,
			Operations: filterOperations(instance.Operations, reachable),
		},
		Exports: exports,
	}
}

// BindAuthorFragment binds all Fragment exports as new author-visible Logical Outputs.
func BindAuthorFragment(instance ElaboratedFragment, outputs map[string]string) FragmentContribution {
	return bindExports(instance, outputs, true)
}

// MergeFragmentContributions concatenates already elaborated contributions.
// Final Graph validation remains Core's authority. Satisfactions are not merged.
func MergeFragmentContributions(graph protocol.CompiledGraph, contributions ...FragmentContribution) protocol.CompiledGraph {
	outputs := append([]protocol.LogicalOutput{}, graph.Outputs...)
	candidates := append([]protocol.Candidate{}, graph.Candidates...)
	operations := append([]protocol.OperationNode{}, graph.Operations...)
	for _, item := range contributions {
		outputs = append(outputs, item.Outputs...)
		candidates = append(candidates, item.Candidates...)
		operations = append(operations, item.Operations...)
	}
	return protocol.CompiledGraph{Outputs: outputs, Candidates: candidates, Operations: operations}
}

func jsTrim(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\ufeff', '\u2028', '\u2029':
			return true
		default:
			return unicode.Is(unicode.Zs, r)
		}
	})
}

func encodeURIComponent(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			fmt.Fprintf(&b, "%%%02X", value[i])
			i++
			continue
		}
		if uriUnescaped(r) {
			b.WriteRune(r)
			i += size
			continue
		}
		for _, c := range []byte(value[i : i+size]) {
			fmt.Fprintf(&b, "%%%02X", c)
		}
		i += size
	}
	return b.String()
}

func uriUnescaped(r rune) bool {
	switch r {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
		return true
	default:
		return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
	}
}
