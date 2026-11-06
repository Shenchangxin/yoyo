// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

type SourceRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// TypedRecord is one typed value in the graph.
// Semantic validation happens at the Host admission boundary.
type TypedRecord struct {
	ID    RecordId    `json:"id"`
	Type  TypeRef     `json:"type"`
	Value StoredValue `json:"value"`
}

type Need struct {
	ID          NeedId        `json:"id"`
	Capability  CapabilityRef `json:"capability"`
	Returns     TypeRef       `json:"returns"`
	Constraints any           `json:"constraints"`
	Result      RecordId      `json:"result"`
}

type NeedBinding struct {
	ID     NeedId   `json:"id"`
	Result RecordId `json:"result"`
}

type RecordRef struct {
	Kind string   `json:"kind"`
	ID   RecordId `json:"id"`
}

type LogicalOutputRef struct {
	Kind string          `json:"kind"`
	ID   LogicalOutputId `json:"id"`
}

type OperationResultRef struct {
	Kind      string      `json:"kind"`
	Operation OperationId `json:"operation"`
}

// GraphValueRef is one input of an operation.
// Kind is "record", "logical-output", or "operation-result".
type GraphValueRef struct {
	Kind      string `json:"kind"`
	ID        string `json:"id,omitempty"`
	Operation string `json:"operation,omitempty"`
}

type LogicalOutput struct {
	ID      LogicalOutputId `json:"id"`
	Type    TypeRef         `json:"type"`
	Primary CandidateId     `json:"primary"`
}

// OperationResult is either a producer output or a requested Need.
// Kind is "output" or "need". ID is the Need id when Kind is "need".
type OperationResult struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	Record RecordId `json:"record"`
	ID     NeedId   `json:"id,omitempty"`
}

type OperationNode struct {
	ID       OperationId              `json:"id"`
	Producer ProducerRef              `json:"producer"`
	Inputs   map[string]GraphValueRef `json:"inputs"`
	Result   OperationResult          `json:"result"`
}

type ProvidedValue struct {
	ID    RecordId    `json:"id"`
	Value StoredValue `json:"value"`
}

// CandidateRoot is a literal value or the result of one operation.
// Kind is "value" or "operation".
type CandidateRoot struct {
	Kind   string             `json:"kind"`
	Value  ProvidedValue      `json:"value,omitempty"`
	Result OperationResultRef `json:"result,omitempty"`
}

type Candidate struct {
	ID   CandidateId   `json:"id"`
	Type TypeRef       `json:"type"`
	Root CandidateRoot `json:"root"`
}

type CompiledGraph struct {
	Format     string          `json:"format"`
	Outputs    []LogicalOutput `json:"outputs"`
	Candidates []Candidate     `json:"candidates"`
	Operations []OperationNode `json:"operations"`
}

type BuildTarget struct {
	Output LogicalOutputId `json:"output"`
}

type Satisfaction struct {
	Output    LogicalOutputId `json:"output"`
	Candidate CandidateId     `json:"candidate"`
}

type BuildRequest struct {
	Format  string        `json:"format"`
	Targets []BuildTarget `json:"targets"`
}

type BuildOutputBinding struct {
	Output LogicalOutputId `json:"output"`
	Record RecordId        `json:"record"`
	Type   TypeRef         `json:"type"`
}

type ProducerStep struct {
	ID       StepId                 `json:"id"`
	Producer ProducerRef            `json:"producer"`
	Inputs   map[string]RecordId    `json:"inputs"`
	Outputs  map[string]RecordId    `json:"outputs"`
	Needs    map[string]NeedBinding `json:"needs"`
}

type BuildGoal struct {
	Record RecordId `json:"record"`
	Type   TypeRef  `json:"type"`
}

type BuildPlan struct {
	Format         string               `json:"format"`
	Steps          []ProducerStep       `json:"steps"`
	Goals          []BuildGoal          `json:"goals"`
	OutputBindings []BuildOutputBinding `json:"outputBindings"`
}

// RunGraph is the complete execution intent compiled from one .svrun source.
type RunGraph struct {
	Format        string          `json:"format"`
	Records       []TypedRecord   `json:"records"`
	Candidates    []Candidate     `json:"candidates"`
	Operations    []OperationNode `json:"operations"`
	Satisfactions []Satisfaction  `json:"satisfactions"`
	Targets       []BuildTarget   `json:"targets"`
}

type LinkedProgram struct {
	Closure ResolvedModuleClosure `json:"closure"`
	Records []TypedRecord         `json:"records"`
}

type StepState struct {
	ID     StepId `json:"id"`
	Status string `json:"status"`
}

type InvokeProducerCommand struct {
	Kind     string              `json:"kind"`
	ID       CommandId           `json:"id"`
	Step     StepId              `json:"step"`
	Producer ProducerRef         `json:"producer"`
	Inputs   map[string]RecordId `json:"inputs"`
}

type FulfillNeedCommand struct {
	Kind string    `json:"kind"`
	ID   CommandId `json:"id"`
	Need Need      `json:"need"`
}

type ProducerCompletedEvent struct {
	Kind    string         `json:"kind"`
	Command CommandId      `json:"command"`
	Outputs map[string]any `json:"outputs"`
	Needs   map[string]any `json:"needs"`
}

type NeedFulfilledEvent struct {
	Kind    string      `json:"kind"`
	Command CommandId   `json:"command"`
	Value   StoredValue `json:"value"`
}

type CommandFailedEvent struct {
	Kind    string    `json:"kind"`
	Command CommandId `json:"command"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
}

// BuildCommand is an invoke-producer or fulfill-need command.
// Kind selects which fields are set.
type BuildCommand struct {
	Kind     string              `json:"kind"`
	ID       CommandId           `json:"id"`
	Step     StepId              `json:"step,omitempty"`
	Producer ProducerRef         `json:"producer,omitempty"`
	Inputs   map[string]RecordId `json:"inputs,omitempty"`
	Need     Need                `json:"need,omitempty"`
}

// CommandResult is one execution result offered to Core.
// Kind is producer-completed, need-fulfilled, or command-failed.
type CommandResult struct {
	Kind    string                 `json:"kind"`
	Command CommandId              `json:"command"`
	Outputs map[string]StoredValue `json:"outputs,omitempty"`
	Needs   map[string]any         `json:"needs,omitempty"`
	Value   StoredValue            `json:"value,omitempty"`
	Code    string                 `json:"code,omitempty"`
	Message string                 `json:"message,omitempty"`
}

// BuildFact is one durable fact. Kind is producer-applied, need-applied, or command-failed.
type BuildFact struct {
	Format     string          `json:"format"`
	Kind       string          `json:"kind"`
	Command    CommandId       `json:"command"`
	Step       StepId          `json:"step,omitempty"`
	Records    []TypedRecord   `json:"records,omitempty"`
	Needs      []Need          `json:"needs,omitempty"`
	Need       NeedId          `json:"need,omitempty"`
	Record     TypedRecord     `json:"record,omitempty"`
	Diagnostic BuildDiagnostic `json:"diagnostic,omitempty"`
}

type BuildDiagnostic struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Subject *string `json:"subject,omitempty"`
}

// BuildDefinition is the immutable finite program selected by one Author Graph plus one Run Graph.
type BuildDefinition struct {
	Format         string        `json:"format"`
	Program        LinkedProgram `json:"program"`
	InitialRecords []TypedRecord `json:"initialRecords"`
	Plan           BuildPlan     `json:"plan"`
	Targets        []BuildTarget `json:"targets"`
}

type ProducerAppliedFact struct {
	Format  string        `json:"format"`
	Command CommandId     `json:"command"`
	Kind    string        `json:"kind"`
	Step    StepId        `json:"step"`
	Records []TypedRecord `json:"records"`
	Needs   []Need        `json:"needs"`
}

type NeedAppliedFact struct {
	Format  string      `json:"format"`
	Command CommandId   `json:"command"`
	Kind    string      `json:"kind"`
	Need    NeedId      `json:"need"`
	Record  TypedRecord `json:"record"`
}

type CommandFailedFact struct {
	Format     string          `json:"format"`
	Command    CommandId       `json:"command"`
	Kind       string          `json:"kind"`
	Diagnostic BuildDiagnostic `json:"diagnostic"`
}

// BuildState is the materialized read view.
// Durable stores persist Definition plus Facts, never this object.
type BuildState struct {
	Format      string            `json:"format"`
	Program     LinkedProgram     `json:"program"`
	Plan        BuildPlan         `json:"plan"`
	Targets     []BuildTarget     `json:"targets"`
	Status      string            `json:"status"`
	Records     []TypedRecord     `json:"records"`
	Steps       []StepState       `json:"steps"`
	Needs       []Need            `json:"needs"`
	Outstanding []BuildCommand    `json:"outstanding"`
	Diagnostics []BuildDiagnostic `json:"diagnostics"`
}
