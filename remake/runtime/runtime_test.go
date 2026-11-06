// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package runtime

import (
	"io"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/remake/protocol"
)

func TestCapacityUnitsRejectsAnEmptyOrOversizedClaim(t *testing.T) {
	if CapacityUnits(CapacityResourceClaim{ID: "pool", Limit: 2}) != 1 {
		t.Fatal("default units")
	}
	two := int64(2)
	if CapacityUnits(CapacityResourceClaim{ID: "pool", Limit: 4, Units: &two}) != 2 {
		t.Fatal("explicit units")
	}
	if message := recoverMessage(func() { CapacityUnits(CapacityResourceClaim{ID: " ", Limit: 1}) }); message != "Capacity   requires a positive integer limit and units" {
		t.Fatal(message)
	}
	if message := recoverMessage(func() { CapacityUnits(CapacityResourceClaim{ID: "pool", Limit: 1, Units: &two}) }); message != "Capacity pool request needs 2 units but its limit is 1" {
		t.Fatal(message)
	}
	period := int64(0)
	if message := recoverMessage(func() {
		CapacityUnits(CapacityResourceClaim{ID: "pool", Limit: 1, PeriodMs: &period})
	}); message != "Resource pool periodMs must be a positive integer" {
		t.Fatal(message)
	}
}

func TestOAuthCredentialRoundTripAndCredentialRefs(t *testing.T) {
	expiry := 12.5
	refresh := "refresh"
	encoded := EncodeOAuth2Credential(OAuth2Credential{AccessToken: "token", RefreshToken: &refresh, ExpiresAt: &expiry})
	if encoded != `{"format":"hypit.oauth2-credential@1","accessToken":"token","refreshToken":"refresh","expiresAt":12.5}` {
		t.Fatal(encoded)
	}
	decoded := DecodeOAuth2Credential(encoded)
	if decoded == nil || decoded.AccessToken != "token" || decoded.RefreshToken == nil || *decoded.RefreshToken != "refresh" || decoded.ExpiresAt == nil || *decoded.ExpiresAt != 12.5 {
		t.Fatalf("%+v", decoded)
	}
	if DecodeOAuth2Credential("{") != nil || DecodeOAuth2Credential("[]") != nil || DecodeOAuth2Credential(`{"format":"hypit.oauth2-credential@1","accessToken":""}`) != nil {
		t.Fatal("invalid secret was accepted")
	}
	infinite := math.Inf(1)
	if message := recoverMessage(func() { EncodeOAuth2Credential(OAuth2Credential{AccessToken: "token", ExpiresAt: &infinite}) }); message != "OAuth expiry is invalid" {
		t.Fatal(message)
	}
	if message := recoverMessage(func() { VerifyCredentialRef(CredentialRef{Store: " \t", Key: "k"}) }); message != "CredentialRef store is empty" {
		t.Fatal(message)
	}
	if CredentialRefOf("env", "\u0085").Key != "\u0085" {
		t.Fatal("U+0085 was trimmed")
	}
}

func TestCompositeStoreKeepsASingleOwner(t *testing.T) {
	secret := &CredentialValue{Secret: "value"}
	first := memoryStore{name: "env", secret: secret}
	second := memoryStore{name: "os"}
	store := NewCompositeCredentialStore([]CredentialStore{first, &second})
	got, err := store.Resolve(CredentialRefOf("env", "KEY"))
	if err != nil || got == nil || got.Secret != "value" {
		t.Fatalf("%v %+v", err, got)
	}
	if IsWritableCredentialStore(store) {
		t.Fatal("composite was treated as writable")
	}
	if WritableCredentialStoreFor(store, CredentialRefOf("env", "KEY")) == nil {
		t.Fatal("missing writable owner")
	}
	other := memoryStore{name: "env", secret: secret}
	ambiguous := NewCompositeCredentialStore([]CredentialStore{first, &other})
	if message := recoverMessage(func() { _, _ = ambiguous.Resolve(CredentialRefOf("env", "KEY")) }); message != "CredentialRef store env is implemented more than once" {
		t.Fatal(message)
	}
}

func TestExecutionLogKeepsTheTailAndASplitRune(t *testing.T) {
	line := func(command string) string {
		return `{"format":"hypit.execution-log@1","endpoint":"local","kind":"phase","phase":"render","time":1,"command":"` + command + `"}` + "\n"
	}
	view := ReadExecutionLog([][]byte{
		[]byte(line("a") + line("b")),
		[]byte("\n" + line("c")),
		[]byte(line("d")),
	}, 2)
	if view.Total != 4 || len(view.Records) != 2 || view.Records[0].Command != "c" || view.Records[1].Command != "d" {
		t.Fatalf("%+v", view)
	}
	split := ReadExecutionLog([][]byte{[]byte("{\"format\":\"hypit.execution-log@1\",\"kind\":\"started\",\"command\":\""), {0xE7}, {0xB4, 0xA0}, []byte("\"}\n")}, 1)
	if split.Total != 1 || split.Records[0].Command != "素" {
		t.Fatalf("%+v", split)
	}
	if message := recoverMessage(func() { ReadExecutionLog(nil, 0) }); message != "Log line count must be a positive integer" {
		t.Fatal(message)
	}
}

func TestSchedulerValidatesRequestsAndAdmitsOneResourceAtATime(t *testing.T) {
	scheduler := NewInProcessBuildScheduler(RuntimeCommandExecutor{}, BuildSchedulerOptions{})
	if message := recoverMessage(func() { scheduler.Run([]ScheduledBuild{{ID: "  "}}) }); message != "scheduled build id must not be empty" {
		t.Fatal(message)
	}
	state := protocol.BuildState{Status: "running"}
	if message := recoverMessage(func() {
		scheduler.Run([]ScheduledBuild{{ID: "same", State: &state}, {ID: "same", State: &state}})
	}); message != "scheduled build same is duplicated" {
		t.Fatal(message)
	}
	var active, maxActive int
	var mu sync.Mutex
	limited := NewInProcessBuildScheduler(RuntimeCommandExecutor{
		Prepare: func(current protocol.BuildState) RuntimePreparation {
			runnable := make([]RuntimeRunnableCommand, len(current.Outstanding))
			for i, command := range current.Outstanding {
				runnable[i] = RuntimeRunnableCommand{Command: command, Resources: []CapacityResourceClaim{{ID: "pool", Limit: 1}}}
			}
			return RuntimePreparation{State: current, Runnable: runnable}
		},
		ExecuteCommand: func(protocol.BuildState, RuntimeRunnableCommand, RuntimeExecutionContext) (RuntimeExecutionResult, error) {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			active--
			mu.Unlock()
			return RuntimeExecutionResult{Status: "deferred", Reason: "later"}, nil
		},
	}, BuildSchedulerOptions{})
	left := protocol.BuildState{Status: "running", Outstanding: []protocol.BuildCommand{{Kind: "invoke-producer", ID: "left"}}}
	right := protocol.BuildState{Status: "running", Outstanding: []protocol.BuildCommand{{Kind: "invoke-producer", ID: "right"}}}
	results := limited.Run([]ScheduledBuild{{ID: "left", State: &left}, {ID: "right", State: &right}})
	if len(results) != 2 || results[0].Status != "paused" || results[0].Outcomes[0].Status != "deferred" || maxActive != 1 {
		t.Fatalf("%+v max %d", results, maxActive)
	}
	conflict := NewInProcessBuildScheduler(RuntimeCommandExecutor{
		Prepare: func(current protocol.BuildState) RuntimePreparation {
			limit := int64(1)
			if current.Outstanding[0].ID == "right" {
				limit = 2
			}
			return RuntimePreparation{State: current, Runnable: []RuntimeRunnableCommand{{
				Command: current.Outstanding[0], Resources: []CapacityResourceClaim{{ID: "pool", Limit: limit}},
			}}}
		},
		ExecuteCommand: func(protocol.BuildState, RuntimeRunnableCommand, RuntimeExecutionContext) (RuntimeExecutionResult, error) {
			return RuntimeExecutionResult{Status: "deferred", Reason: "later"}, nil
		},
	}, BuildSchedulerOptions{})
	if message := recoverMessage(func() { conflict.Run([]ScheduledBuild{{ID: "left", State: &left}, {ID: "right", State: &right}}) }); message != "resource pool has conflicting default concurrency 1 and 2" {
		t.Fatal(message)
	}
}

func TestDurableSchedulerUsesTheIndexedMachine(t *testing.T) {
	var preparations, executions, appends int
	state := protocol.BuildState{Format: "hypit.build@1", Status: "running", Outstanding: []protocol.BuildCommand{{Kind: "invoke-producer", ID: "step"}}}
	snapshot := &BuildSnapshot{Build: "indexed", Definition: protocol.BuildDefinition{}, State: state}
	scheduler := NewInProcessBuildScheduler(RuntimeCommandExecutor{
		Prepare: func(protocol.BuildState) RuntimePreparation {
			panic(errorsMessage("durable scheduling rebuilt a detached BuildState"))
		},
		PrepareExecution: func(execution RuntimeBuildExecution) RuntimePreparation {
			preparations++
			commands := execution.Commands()
			runnable := make([]RuntimeRunnableCommand, len(commands))
			for i, command := range commands {
				runnable[i] = RuntimeRunnableCommand{Command: command}
			}
			return RuntimePreparation{State: execution.View(), Runnable: runnable}
		},
		ExecuteExecutionCommand: func(_ RuntimeBuildExecution, command RuntimeRunnableCommand, _ RuntimeExecutionContext) (RuntimeExecutionResult, error) {
			executions++
			return RuntimeExecutionResult{Status: "completed", Event: protocol.CommandResult{
				Kind: "command-failed", Command: command.Command.ID, Code: "STOPPED", Message: "stopped",
			}}, nil
		},
	}, BuildSchedulerOptions{BuildStore: memoryBuilds{appends: &appends}})
	results := scheduler.Run([]ScheduledBuild{{ID: "indexed", Snapshot: snapshot}})
	if preparations != 1 || executions != 1 || appends != 1 || results[0].Status != "failed" {
		t.Fatalf("prep %d exec %d append %d %+v", preparations, executions, appends, results)
	}
	if !IsStreamingResourceStore(streamStore{}) || IsStreamingResourceStore(byteStore{}) {
		t.Fatal("streaming detection")
	}
}

type memoryStore struct {
	name   string
	secret *CredentialValue
}

func (s memoryStore) Resolve(ref CredentialRef) (*CredentialValue, error) {
	if ref.Store == s.name {
		return s.secret, nil
	}
	return nil, nil
}

func (s memoryStore) Owns(ref CredentialRef) bool              { return ref.Store == s.name }
func (s memoryStore) Put(CredentialRef, CredentialValue) error { return nil }
func (s memoryStore) Delete(CredentialRef) (bool, error)       { return true, nil }

type byteStore struct{}

func (byteStore) Has(protocol.ResourceId) (bool, error) { return false, nil }
func (byteStore) Get(protocol.ResourceId) ([]byte, bool, error) {
	return nil, false, nil
}
func (byteStore) Put([]byte, string) (protocol.BlobRef, error) { return protocol.BlobRef{}, nil }
func (byteStore) Write(protocol.BlobRef, []byte) error         { return nil }

type streamStore struct{ byteStore }

func (streamStore) PutStream(io.Reader, string) (protocol.BlobRef, error) {
	return protocol.BlobRef{}, nil
}
func (streamStore) WriteStream(protocol.BlobRef, io.Reader) error { return nil }
func (streamStore) Open(protocol.ResourceId) (io.ReadCloser, bool, error) {
	return nil, false, nil
}

type memoryBuilds struct{ appends *int }

func (memoryBuilds) Create(string, protocol.BuildDefinition) (BuildSnapshot, error) {
	return BuildSnapshot{}, nil
}
func (memoryBuilds) Read(string) (*BuildSnapshot, error) { return nil, nil }
func (m memoryBuilds) Append(string, protocol.BuildFact) error {
	*m.appends++
	return nil
}

func errorsMessage(message string) error { return &plainError{message} }

type plainError struct{ message string }

func (e *plainError) Error() string { return e.message }

func recoverMessage(fn func()) (message string) {
	defer func() {
		rec := recover()
		err, ok := rec.(error)
		if !ok {
			panic(rec)
		}
		message = err.Error()
	}()
	fn()
	return ""
}
