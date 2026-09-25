package api

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRPCHealthAndUnknown(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ok := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "health"})
	if ok.Error != nil {
		t.Fatal(ok.Error)
	}
	m, _ := ok.Result.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", ok.Result)
	}
	bad := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "nope"})
	if bad.Error == nil {
		t.Fatal("expected method error")
	}
}

func TestRPCHarnessLineage(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "harness.get"})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	m, _ := got.Result.(map[string]any)
	if m["active"] == "" || m["active"] == nil {
		t.Fatalf("active %+v", m)
	}
	nodes, ok := m["lineage"].([]app.LineageNode)
	if !ok || len(nodes) == 0 {
		t.Fatalf("lineage %+v", m["lineage"])
	}
	if nodes[0].Hash == "" {
		t.Fatalf("node %+v", nodes[0])
	}
}

func TestRPCThreadLifecycle(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "thread.start", Params: jsonRaw(`{"workspace":"` + filepath.ToSlash(ws) + `"}`)})
	if created.Error != nil {
		t.Fatal(created.Error)
	}
	m, _ := created.Result.(app.SessionMeta)
	if m.ID == "" {
		t.Fatalf("%+v", created.Result)
	}
	pin := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "thread.pin", Params: jsonRaw(`{"session":"` + m.ID + `","pinned":true}`)})
	if pin.Error != nil {
		t.Fatal(pin.Error)
	}
	iso := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 4, Method: "thread.isolate.set", Params: jsonRaw(`{"session":"` + m.ID + `","isolate":true}`)})
	if iso.Error != nil {
		t.Fatal(iso.Error)
	}
	im, _ := iso.Result.(app.SessionMeta)
	if !im.Isolate || im.Worktree == "" {
		t.Fatalf("isolate %+v", iso.Result)
	}
	dump := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 5, Method: "session.dump", Params: jsonRaw(`{"session":"` + m.ID[:8] + `"}`)})
	if dump.Error != nil {
		t.Fatal(dump.Error)
	}
	dm, _ := dump.Result.(app.SessionDump)
	if dm.ID != m.ID {
		t.Fatalf("dump %+v", dump.Result)
	}
	del := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "thread.delete", Params: jsonRaw(`{"session":"` + m.ID + `"}`)})
	if del.Error != nil {
		t.Fatal(del.Error)
	}
}

func jsonRaw(s string) json.RawMessage { return json.RawMessage(s) }

func TestRPCWorkspacePreview(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "n.go"), []byte("package n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"workspace": ws, "path": "n.go"})
	if err != nil {
		t.Fatal(err)
	}
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 9, Method: "workspace.preview", Params: params})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	m, _ := got.Result.(map[string]any)
	if m["lang"] != "go" {
		t.Fatalf("%+v", got.Result)
	}
	text, _ := m["text"].(string)
	if !strings.Contains(text, "package n") {
		t.Fatalf("text %+v", m)
	}
}

func TestRPCCanvasHTTP(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 11, Method: "canvas.http", Params: jsonRaw(`{"method":"GET","path":"canvas-projects"}`)})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	if got.Result == nil {
		t.Fatal("canvas.http empty result")
	}
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 12, Method: "canvas.http", Params: jsonRaw(`{"method":"POST","path":"canvas-projects","body":{"title":"Board"}}`)})
	if created.Error != nil {
		t.Fatalf("create canvas: %v", created.Error)
	}
}

func TestLineClientHealth(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	pr, pw := io.Pipe()
	qr, qw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ServeRPC(ctx, a, pr, qw) }()
	cli := NewLineClient(qr, pw)
	v, err := cli.Call("health", nil)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := v.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", v)
	}
}
