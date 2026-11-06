// Copyright 2026 Hypit.AI
// Translated from @hypit/author (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package author

import (
	"slices"
	"testing"

	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

var laboratory = protocol.ModuleRef{Name: "example.laboratory", Version: "1"}
var sampleType = protocol.TypeRef{Module: laboratory, Name: "Sample"}
var measurementType = protocol.TypeRef{Module: laboratory, Name: "Measurement"}
var reportType = protocol.TypeRef{Module: laboratory, Name: "Report"}
var measureProducer = protocol.ProducerRef{Module: laboratory, Name: "measure"}
var reportProducer = protocol.ProducerRef{Module: laboratory, Name: "write-report"}
var echoProducer = protocol.ProducerRef{Module: laboratory, Name: "echo-sample"}

var manifest = protocol.ModuleManifest{
	Format:       "hypit.module@1",
	Name:         laboratory.Name,
	Version:      laboratory.Version,
	Dependencies: []protocol.ModuleDependency{},
	Types: []protocol.TypeDeclaration{
		{Name: sampleType.Name},
		{Name: measurementType.Name},
		{Name: reportType.Name},
	},
	Capabilities: []protocol.CapabilityDeclaration{},
	Producers: []protocol.ProducerDeclaration{
		{
			Name:    measureProducer.Name,
			Inputs:  []protocol.PortDeclaration{{Name: "sample", Type: sampleType}},
			Outputs: []protocol.PortDeclaration{{Name: "measurement", Type: measurementType}},
			Needs:   []protocol.NeedPortDeclaration{},
		},
		{
			Name:    reportProducer.Name,
			Inputs:  []protocol.PortDeclaration{{Name: "measurement", Type: measurementType}},
			Outputs: []protocol.PortDeclaration{{Name: "report", Type: reportType}},
			Needs:   []protocol.NeedPortDeclaration{},
		},
		{
			Name:    echoProducer.Name,
			Inputs:  []protocol.PortDeclaration{{Name: "sample", Type: sampleType}},
			Outputs: []protocol.PortDeclaration{{Name: "sample", Type: sampleType}},
			Needs:   []protocol.NeedPortDeclaration{},
		},
	},
}

func program() protocol.LinkedProgram {
	closure := kernel.CreateResolvedClosure([]protocol.ModuleManifest{manifest})
	return kernel.Link(closure, []protocol.TypedRecord{kernel.SealRecord(protocol.TypedRecord{
		ID:    "sample:soil",
		Type:  sampleType,
		Value: protocol.StoredValue{Kind: "inline", Value: "soil"},
	})})
}

func singleOperationFragment(inputName string, inputType protocol.TypeRef, producer protocol.ProducerRef, resultName string, resultType protocol.TypeRef) GraphFragment {
	return SealGraphFragment(GraphFragment{
		Inputs: []FragmentInput{{Name: inputName, Type: inputType}},
		Operations: []FragmentOperation{{
			ID:       "produce",
			Producer: producer,
			Inputs: map[string]FragmentValueRef{
				inputName: {Kind: "fragment-input", Name: inputName},
			},
			Result: FragmentOperationResult{Kind: "output", Name: resultName},
		}},
		Exports: []FragmentExport{{
			Name: "result",
			Type: resultType,
			Root: FragmentValueRef{Kind: "fragment-operation", Operation: "produce"},
		}},
	})
}

var measureFragment = singleOperationFragment("sample", sampleType, measureProducer, "measurement", measurementType)
var reportFragment = singleOperationFragment("measurement", measurementType, reportProducer, "report", reportType)
var echoFragment = singleOperationFragment("sample", sampleType, echoProducer, "sample", sampleType)
var fragments = map[string]GraphFragment{
	measureFragment.ID: measureFragment,
	reportFragment.ID:  reportFragment,
	echoFragment.ID:    echoFragment,
}

func resolveLaboratory(id string) (GraphFragment, bool) {
	fragment, ok := fragments[id]
	return fragment, ok
}

func TestAuthorLinkingResolvesForwardComponentReferences(t *testing.T) {
	linked := program()
	components := []AuthorComponent{
		{
			ID:       "final-report",
			Fragment: reportFragment.ID,
			Inputs: map[string]AuthorValueRef{
				"measurement": {Kind: "component-output", Component: "measurement", Output: "result"},
			},
			Outputs: map[string]string{"result": "report:final"},
		},
		{
			ID:       "measurement",
			Fragment: measureFragment.ID,
			Inputs:   map[string]AuthorValueRef{"sample": {Kind: "record", ID: "sample:soil"}},
			Outputs:  map[string]string{"result": "measurement:soil"},
		},
	}

	elaborated := ElaborateAuthorGraph(linked, components, resolveLaboratory)
	if len(elaborated.Outputs) != 2 {
		t.Fatalf("outputs %d", len(elaborated.Outputs))
	}
	gotIDs := []string{elaborated.Outputs[0].ID, elaborated.Outputs[1].ID}
	wantIDs := []string{"measurement:soil", "report:final"}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("%v", gotIDs)
	}
	var reportOperation protocol.OperationNode
	foundReport := false
	for _, operation := range elaborated.Operations {
		if operation.Producer.Name == reportProducer.Name {
			reportOperation = operation
			foundReport = true
			break
		}
	}
	if !foundReport {
		t.Fatal("missing report operation")
	}
	measurement := reportOperation.Inputs["measurement"]
	if measurement.Kind != "logical-output" || measurement.ID != "measurement:soil" || measurement.Operation != "" {
		t.Fatalf("%+v", measurement)
	}

	request := kernel.SealBuildRequest(protocol.BuildRequest{
		Targets: []protocol.BuildTarget{{Output: "report:final"}},
	})
	state := kernel.Start(linked, elaborated, request)
	gotNames := make([]string, len(state.Plan.Steps))
	for i, step := range state.Plan.Steps {
		gotNames[i] = step.Producer.Name
	}
	sortUTF16(gotNames)
	wantNames := []string{measureProducer.Name, reportProducer.Name}
	sortUTF16(wantNames)
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("%v", gotNames)
	}
	var measurementStep, reportStep protocol.ProducerStep
	foundMeasure, foundWrite := false, false
	for _, step := range state.Plan.Steps {
		if step.Producer.Name == measureProducer.Name {
			measurementStep = step
			foundMeasure = true
		}
		if step.Producer.Name == reportProducer.Name {
			reportStep = step
			foundWrite = true
		}
	}
	if !foundMeasure || !foundWrite {
		t.Fatal("missing plan steps")
	}
	if reportStep.Inputs["measurement"] != measurementStep.Outputs["measurement"] {
		t.Fatalf("%s != %s", reportStep.Inputs["measurement"], measurementStep.Outputs["measurement"])
	}
}

func TestAuthorLinkingRejectsCycles(t *testing.T) {
	components := []AuthorComponent{
		{
			ID:       "left",
			Fragment: echoFragment.ID,
			Inputs: map[string]AuthorValueRef{
				"sample": {Kind: "component-output", Component: "right", Output: "result"},
			},
			Outputs: map[string]string{"result": "sample:left"},
		},
		{
			ID:       "right",
			Fragment: echoFragment.ID,
			Inputs: map[string]AuthorValueRef{
				"sample": {Kind: "component-output", Component: "left", Output: "result"},
			},
			Outputs: map[string]string{"result": "sample:right"},
		},
	}
	err := recoverError(t, func() {
		ElaborateAuthorGraph(program(), components, resolveLaboratory)
	})
	graphErr, ok := err.(*AuthorGraphError)
	if !ok || graphErr.Code != "AUTHOR_COMPONENT_CYCLE" {
		t.Fatalf("%T %v", err, err)
	}
}

func TestAuthorLinkingChecksSymbolicInputTypes(t *testing.T) {
	components := []AuthorComponent{{
		ID:       "final-report",
		Fragment: reportFragment.ID,
		Inputs:   map[string]AuthorValueRef{"measurement": {Kind: "record", ID: "sample:soil"}},
		Outputs:  map[string]string{"result": "report:invalid"},
	}}
	err := recoverError(t, func() {
		ElaborateAuthorGraph(program(), components, resolveLaboratory)
	})
	graphErr, ok := err.(*AuthorGraphError)
	if !ok || graphErr.Code != "AUTHOR_INPUT_TYPE_MISMATCH" {
		t.Fatalf("%T %v", err, err)
	}
}

func recoverError(t *testing.T, fn func()) error {
	t.Helper()
	var err error
	panicked := false
	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			panicked = true
			got, ok := rec.(error)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			err = got
		}()
		fn()
	}()
	if !panicked {
		t.Fatal("expected a panic")
	}
	return err
}
