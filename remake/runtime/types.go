// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime/src/types.ts, execution.ts, and operations.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

// Package runtime is the in-process build scheduler and its host ports.
package runtime

import (
	"io"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// RuntimeBlockedCommand explains why one outstanding command cannot run.
type RuntimeBlockedCommand struct {
	Command string
	Reason  string
	Subject string
}

// RuntimeRunnableCommand is one command whose resources are acquired together.
type RuntimeRunnableCommand struct {
	Command BuildCommandView
	// Resources are acquired atomically before the command can cause a side effect.
	Resources []CapacityResourceClaim
	// CapacityMode is "active" or "asynchronous". Nil means the property is absent.
	CapacityMode *string
}

// BuildCommandView is the outstanding command handed to an executor.
type BuildCommandView = protocol.BuildCommand

// RuntimePreparation classifies one Build's outstanding commands.
type RuntimePreparation struct {
	State    protocol.BuildState
	Runnable []RuntimeRunnableCommand
	Blocked  []RuntimeBlockedCommand
}

// RuntimeExecutionContext is the host context for one command call.
type RuntimeExecutionContext struct {
	Build string
	// ReportProgress observes activity. It never changes scheduling or Core facts.
	ReportProgress func(activity CommandActivity) error
	// RecordExecution stores evidence. Implementations keep logging failures out of Provider outcomes.
	RecordExecution func(event ExecutionLogEvent) error
	// ReleaseOperationCapacity releases only the asynchronous Operation's occupancy.
	ReleaseOperationCapacity func() error
}

// CommandActivity is the current Provider-reported activity.
type CommandActivity struct {
	Endpoint string
	Progress OperationProgress
}

// RuntimeExecutionResult status is completed, pending, or deferred.
type RuntimeExecutionResult struct {
	Status    string
	Event     protocol.CommandResult
	Operation string
	WakeAt    *float64
	Reason    string
}

// RuntimeBuildExecution is the indexed path used while one BuildMachine is alive.
type RuntimeBuildExecution interface {
	View() protocol.BuildState
	Commands() []protocol.BuildCommand
	Record(id string) (protocol.TypedRecord, bool)
}

// RuntimeCommandExecutor classifies and runs commands. Nil optional funcs are absent methods.
type RuntimeCommandExecutor struct {
	Prepare                 func(protocol.BuildState) RuntimePreparation
	PrepareExecution        func(RuntimeBuildExecution) RuntimePreparation
	ExecuteCommand          func(protocol.BuildState, RuntimeRunnableCommand, RuntimeExecutionContext) (RuntimeExecutionResult, error)
	ExecuteExecutionCommand func(RuntimeBuildExecution, RuntimeRunnableCommand, RuntimeExecutionContext) (RuntimeExecutionResult, error)
}

// ResourceStore is the build-local byte store.
type ResourceStore interface {
	Put(bytes []byte, mediaType string) (protocol.BlobRef, error)
	Write(resource protocol.BlobRef, bytes []byte) error
	Get(resource protocol.ResourceId) ([]byte, bool, error)
	Has(resource protocol.ResourceId) (bool, error)
}

// StreamingResourceStore is the optional transfer capability.
type StreamingResourceStore interface {
	ResourceStore
	PutStream(chunks io.Reader, mediaType string) (protocol.BlobRef, error)
	WriteStream(resource protocol.BlobRef, chunks io.Reader) error
	Open(resource protocol.ResourceId) (io.ReadCloser, bool, error)
}

// IsStreamingResourceStore reports whether value exposes putStream, writeStream, and open.
func IsStreamingResourceStore(value ResourceStore) bool {
	_, ok := value.(StreamingResourceStore)
	return ok
}

// BuildSnapshot is one stored Definition plus its accepted Facts.
type BuildSnapshot struct {
	Build      string
	Definition protocol.BuildDefinition
	Facts      []protocol.BuildFact
	State      protocol.BuildState
}

// BuildStore stores one Definition and the Core Facts accepted for it in order.
type BuildStore interface {
	Create(build string, definition protocol.BuildDefinition) (BuildSnapshot, error)
	Read(build string) (*BuildSnapshot, error)
	Append(build string, fact protocol.BuildFact) error
}

// ScheduledBuild is one in-memory state or one durable snapshot.
// A nil State or Snapshot means that alternative is absent.
type ScheduledBuild struct {
	ID       string
	State    *protocol.BuildState
	Snapshot *BuildSnapshot
}

// SchedulerExecutionOutcome is one command attempted by the scheduler.
type SchedulerExecutionOutcome struct {
	Command   string
	Kind      string
	Resources []string
	Status    string
	Operation string
	WakeAt    *float64
	Message   string
}

// ScheduledBuildResult is the terminal view of one submitted Build.
type ScheduledBuildResult struct {
	ID       string
	Status   string
	State    protocol.BuildState
	Outcomes []SchedulerExecutionOutcome
	Blocked  []RuntimeBlockedCommand
}

// BuildSchedulerOptions selects the optional durable authority.
type BuildSchedulerOptions struct {
	BuildStore    BuildStore
	OnStateChange func(build string, state protocol.BuildState) error
}

// OperationReceipt is a non-secret acknowledgement suitable for Results.
type OperationReceipt struct {
	ID  string
	URL *string
}

// OperationProgress is the human-observable projection of one asynchronous Operation.
type OperationProgress struct {
	Phase     string
	Completed *float64
	Total     *float64
	Unit      *string
}

// OperationSnapshot is the provider-neutral record of one asynchronous Operation.
type OperationSnapshot struct {
	ID             string
	Build          string
	Command        string
	Endpoint       string
	Status         string
	CreatedAt      *float64
	AcknowledgedAt *float64
	EndedAt        *float64
	Request        *protocol.FulfillNeedCommand
	Credentials    map[string]CredentialRef
	Pool           *string
	Receipt        *OperationReceipt
	RemoteEnded    bool
	Submission     *string
	Handle         any
	WakeAt         *float64
	Progress       *OperationProgress
	Completion     *protocol.StoredValue
	FailureCode    string
	FailureMessage string
	Cancellation   *OperationCancellation
}

// OperationCancellation is the last acknowledgement of a cancellation request.
type OperationCancellation struct {
	Outcome string
	Message *string
}

// CommandExecutionReceipt is the live handoff for an immediate command.
type CommandExecutionReceipt struct {
	Build    string
	Command  string
	Status   string
	Event    *protocol.CommandResult
	Activity *CommandActivity
}
