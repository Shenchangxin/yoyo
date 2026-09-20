package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestDumpSessionByIDIncludesTrajectoryAndSpill(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("artifact-bytes-", 200)
	sp := runtime.BindSpill(a.Home.Root, ws, sess.ID)
	if sp == nil {
		t.Fatal("spill")
	}
	if id := sp.Put("c1", body); id != "c1" {
		t.Fatalf("put %q", id)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: sess.ID, Payload: map[string]any{"text": "dump", "id": "u1"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeToolCall, SessionID: sess.ID, Payload: map[string]any{"id": "c1", "name": "shell"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeToolResult, SessionID: sess.ID, Payload: map[string]any{
		"id": "c1", "name": "shell", "content": body[:80], "spill_id": "c1", "bytes": len(body),
	}})

	dump, err := a.DumpSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dump.ID != sess.ID {
		t.Fatalf("id %q", dump.ID)
	}
	if len(dump.Trajectory) != 3 {
		t.Fatalf("trajectory %d", len(dump.Trajectory))
	}
	var sawSpill bool
	for _, art := range dump.Artifacts {
		if art.Kind == "spill" && art.ID == "c1" && art.Text == body && art.Bytes == len(body) {
			sawSpill = true
		}
	}
	if !sawSpill {
		t.Fatalf("artifacts %+v", dump.Artifacts)
	}
	if dump.Paths.Trajectory == "" || dump.Paths.SpillDir == "" || dump.Paths.Dump == "" {
		t.Fatalf("paths %+v", dump.Paths)
	}
	if _, err := os.Stat(dump.Paths.Dump); err != nil {
		t.Fatalf("dump.json: %v", err)
	}
	b, err := os.ReadFile(dump.Paths.Dump)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), body) {
		t.Fatal("dump.json missing spill body")
	}

	prefix := sess.ID[:8]
	got, err := a.ResolveSessionID(prefix)
	if err != nil || got != sess.ID {
		t.Fatalf("prefix %q -> %q %v", prefix, got, err)
	}
	viaPrefix, err := a.DumpSession(prefix)
	if err != nil || viaPrefix.ID != sess.ID || len(viaPrefix.Trajectory) != 3 {
		t.Fatalf("dump prefix: id=%s n=%d err=%v", viaPrefix.ID, len(viaPrefix.Trajectory), err)
	}
}

func TestResolveSessionIDAmbiguousAndMissing(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: "aaa111", Payload: map[string]any{"text": "a"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: "aaa222", Payload: map[string]any{"text": "b"}})
	if _, err := a.ResolveSessionID("aaa"); err == nil {
		t.Fatal("expected ambiguous")
	}
	if _, err := a.ResolveSessionID("no-such-session"); err == nil {
		t.Fatal("expected missing")
	}
	id, err := a.ResolveSessionID("aaa111")
	if err != nil || id != "aaa111" {
		t.Fatalf("exact %q %v", id, err)
	}
}

func TestDumpSessionJSONLWithoutMeta(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	id := "eval-deadbeef~write-hello~r0"
	body := "harbor-artifact-body"
	_ = a.Traces.Append(trace.Event{Type: trace.TypeEval, SessionID: id, Payload: map[string]any{"task": "write-hello"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: id, Payload: map[string]any{"text": "run"}})
	sp := runtime.BindSpill(a.Home.Root, "", id)
	if sp == nil {
		t.Fatal("spill")
	}
	sp.Put("out", body)

	if _, err := a.GetSession(id); err == nil {
		t.Fatal("eval sessions have no meta")
	}
	dump, err := a.DumpSession(id)
	if err != nil {
		t.Fatal(err)
	}
	if dump.ID != id || len(dump.Trajectory) != 2 {
		t.Fatalf("dump %+v n=%d", dump.ID, len(dump.Trajectory))
	}
	var saw bool
	for _, art := range dump.Artifacts {
		if art.ID == "out" && art.Text == body {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("artifacts %+v", dump.Artifacts)
	}
	idx := a.ListSessionIndex()
	var found bool
	for _, e := range idx {
		if e.ID == id && e.Events == 2 && !e.HasMeta && e.SpillArtifacts >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("index %+v", idx)
	}
	if _, err := os.Stat(filepath.Join(a.Home.Sessions(), id+".jsonl")); err != nil {
		t.Fatal(err)
	}
}
