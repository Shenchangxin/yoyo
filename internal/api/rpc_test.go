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

func TestConfigSetMergesPartial(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.Config.SearchURL = "https://keep.example/{q}"
	a.Config.CrashResume = true
	if err := a.SaveConfig(); err != nil {
		t.Fatal(err)
	}
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 20, Method: "config.set", Params: jsonRaw(`{"model":"gpt-4.1-mini","theme":"dark"}`)})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	if a.Config.Model != "gpt-4.1-mini" {
		t.Fatalf("model %q", a.Config.Model)
	}
	if a.Config.SearchURL != "https://keep.example/{q}" {
		t.Fatalf("search wiped %q", a.Config.SearchURL)
	}
	if !a.Config.CrashResume {
		t.Fatal("crash_resume wiped")
	}
	if a.Config.Theme != "dark" {
		t.Fatalf("theme %q", a.Config.Theme)
	}
}

func TestRPCPacksListAndEnable(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "packs.list"})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	raw, _ := json.Marshal(got.Result)
	if !strings.Contains(string(raw), `"id":"superpowers"`) {
		t.Fatalf("%s", raw)
	}
	if strings.Contains(string(raw), `"enabled":true`) {
		t.Fatal("catalog must default off")
	}
	en := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "packs.enable", Params: jsonRaw(`{"id":"superpowers","scope":"workspace","enabled":true}`)})
	if en.Error != nil {
		t.Fatal(en.Error)
	}
	again := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "packs.list"})
	raw2, _ := json.Marshal(again.Result)
	if !strings.Contains(string(raw2), `"installed":false`) {
		t.Fatalf("%s", raw2)
	}
	if strings.Contains(string(raw2), `"enabled":true`) {
		t.Fatal("uninstalled pack must not report enabled")
	}
}

func TestRPCPagesProfilesAndPause(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "pages.save", Params: jsonRaw(`{"title":"Brief","content":"hello pages"}`)})
	if created.Error != nil {
		t.Fatal(created.Error)
	}
	list := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "pages.list"})
	if list.Error != nil {
		t.Fatal(list.Error)
	}
	raw, _ := json.Marshal(list.Result)
	if !strings.Contains(string(raw), "Brief") {
		t.Fatalf("%s", raw)
	}
	pro := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "profiles.list"})
	if pro.Error != nil {
		t.Fatal(pro.Error)
	}
	pr, _ := json.Marshal(pro.Result)
	if !strings.Contains(string(pr), "researcher") {
		t.Fatalf("%s", pr)
	}
	pause := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 4, Method: "app.pause.set", Params: jsonRaw(`{"paused":true}`)})
	if pause.Error != nil {
		t.Fatal(pause.Error)
	}
	if !a.IsPaused() {
		t.Fatal("paused")
	}
	health := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 5, Method: "health"})
	h, _ := json.Marshal(health.Result)
	if !strings.Contains(string(h), `"paused":true`) {
		t.Fatalf("%s", h)
	}
}
