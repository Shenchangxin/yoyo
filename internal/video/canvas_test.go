package video

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanvasProjectRevisionAndCAS(t *testing.T) {
	dir := t.TempDir()
	cas := &memCAS{}
	e, err := Open(dir, cas, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	res, err := e.putCanvasBytes("image", "image/png", "a.png", []byte("\x89PNG\r\n\x1a\nxxxx"), 8, 8, 0, "res1")
	if err != nil {
		t.Fatal(err)
	}
	if res.PublicURL == "" || res.ID != "res1" {
		t.Fatalf("resource %+v", res)
	}

	doc := map[string]any{
		"id": "c1", "revision": 0, "title": "Board",
		"nodes":       []any{map[string]any{"id": "n1", "type": "image", "metadata": map[string]any{"storageKey": "resource:res1"}}},
		"connections": []any{},
	}
	raw, _ := json.Marshal(doc)
	sum, env := e.upsertCanvasProject(raw, false)
	if env.Code != 0 {
		t.Fatalf("upsert: %+v", env)
	}
	if intAny(sum["revision"]) != 1 {
		t.Fatalf("revision %v", sum["revision"])
	}

	doc["revision"] = 0
	raw, _ = json.Marshal(doc)
	_, env = e.upsertCanvasProject(raw, false)
	if env.Code != 428 {
		t.Fatalf("expected conflict, got %+v", env)
	}

	doc["revision"] = 1
	doc["title"] = "Board 2"
	raw, _ = json.Marshal(doc)
	sum, env = e.upsertCanvasProject(raw, false)
	if env.Code != 0 || intAny(sum["revision"]) != 2 {
		t.Fatalf("second save %+v %+v", sum, env)
	}

	got := e.canvasHTTP(map[string]any{"method": "GET", "path": "/auth/session"})
	if got.Code != 0 {
		t.Fatalf("session %+v", got)
	}
	user := asMap(asMap(got.Data)["user"])
	if fmt.Sprint(user["id"]) != "local" {
		t.Fatalf("user %+v", user)
	}
	features := asMap(asMap(got.Data)["features"])
	if boolAny(features["creditsEnabled"]) {
		t.Fatal("credits should be off")
	}

	if err := e.BindCanvasSession("s1", "c1"); err != nil {
		t.Fatal(err)
	}
	out, err := e.CanvasTool("s1", "canvas_get_state", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(out, "c1") {
		t.Fatalf("state %s", out)
	}

	task, err := e.createCanvasTask(map[string]any{"type": "text", "prompt": "hi", "projectId": "c1"})
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(task["id"])
	if _, err := e.appendTextDelta(id, "hello "); err != nil {
		t.Fatal(err)
	}
	done, err := e.completeTextReplay(id, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(done["status"]) != "succeeded" {
		t.Fatalf("text task %+v", done)
	}

	j := Job{ID: NewID(), Type: "canvas", Status: "running", Params: marshalJSON(JobParams{CanvasProjectID: "c1", Mode: "image"}), CreatedAt: Now(), UpdatedAt: Now()}
	if err := e.insertJob(j); err != nil {
		t.Fatal(err)
	}
	png := tinyJPEG()
	if err := e.finishMedia(&j, png, false); err != nil {
		t.Fatal(err)
	}
	p := parseParams(j.Params)
	if p.ResourceID == "" || p.ResultState != "READY" {
		t.Fatalf("cas materialize %+v hash=%s", p, j.ResultHash)
	}
	if _, err := e.getCanvasResource(p.ResourceID); err != nil {
		t.Fatal(err)
	}

	broken := Job{ID: NewID(), Type: "canvas", Status: "failed", ResultHash: "missing", Params: marshalJSON(JobParams{CanvasProjectID: "c1", Mode: "image", RecoverAttempts: 0}), CreatedAt: Now(), UpdatedAt: Now()}
	if err := e.insertJob(broken); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RecoverCanvasMedia(broken.ID); err == nil {
		t.Fatal("expected recover failure when CAS object is missing")
	}
}

func TestCanvasHTTPCreateProject(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	env := e.canvasHTTP(map[string]any{"method": "POST", "path": "/canvas-projects", "body": map[string]any{"title": "One"}})
	if env.Code != 0 {
		t.Fatalf("%+v", env)
	}
	proj := asMap(asMap(env.Data)["project"])
	if fmt.Sprint(proj["title"]) != "One" {
		t.Fatalf("%+v", proj)
	}
}

func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestCanvasAgentToolsAndOps(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	names := strings.Join(CanvasToolNames(), ",")
	for _, want := range []string{"canvas_apply_ops", "generate_media", "remember_lesson", "canvas_get_state", "plan_update"} {
		if !strings.Contains(names, want) {
			t.Fatalf("missing tool %s in %s", want, names)
		}
	}
	env := e.canvasHTTP(map[string]any{"method": "POST", "path": "/canvas-projects", "body": map[string]any{"title": "Ops"}})
	proj := asMap(asMap(env.Data)["project"])
	id := fmt.Sprint(proj["id"])
	if err := e.BindCanvasSession("s-ops", id); err != nil {
		t.Fatal(err)
	}
	out, err := e.CanvasTool("s-ops", "canvas_apply_ops", map[string]any{
		"operations": []any{map[string]any{"op": "add_node", "type": "text", "content": "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(out, "hello") && !containsStr(out, "text") && !containsStr(out, "ok") {
		t.Fatalf("apply ops %s", out)
	}
	task, err := e.CanvasTool("s-ops", "generate_media", map[string]any{"prompt": "a cat", "mode": "image"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(task, "queued") && !containsStr(task, "canvas") {
		t.Fatalf("generate %s", task)
	}
}

func TestPublicAppearanceBranding(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	env := e.canvasHTTP(map[string]any{"method": "GET", "path": "/public/appearance"})
	if env.Code != 0 {
		t.Fatalf("%+v", env)
	}
	raw, _ := json.Marshal(env.Data)
	if strings.Contains(string(raw), "影策") {
		t.Fatalf("appearance still contains 影策: %s", raw)
	}
	app := asMap(asMap(env.Data)["appearance"])
	if fmt.Sprint(app["brandName"]) != "Yoyo" {
		t.Fatalf("brand %v", app["brandName"])
	}
	if debrandYingce("Zhipu AI / 影策") != "Zhipu AI" {
		t.Fatalf("vendor debrand")
	}
	if debrandYingce("接入影策画布") != "接入无限画布" {
		t.Fatalf("docs debrand")
	}
}

func TestProjectHistoryAndSession(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	env := e.canvasHTTP(map[string]any{"method": "POST", "path": "/canvas-projects", "body": map[string]any{"title": "Street"}})
	proj := asMap(asMap(env.Data)["project"])
	id := fmt.Sprint(proj["id"])
	if err := e.BindCanvasSession("s-hist", id); err != nil {
		t.Fatal(err)
	}
	items, err := e.ProjectHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected canvas history")
	}
	found := false
	for _, it := range items {
		if fmt.Sprint(it["id"]) == id && fmt.Sprint(it["kind"]) == "canvas" && fmt.Sprint(it["session_id"]) == "s-hist" {
			found = true
		}
	}
	if !found {
		t.Fatalf("history %+v", items)
	}
	st := e.SessionProject("s-hist")
	if fmt.Sprint(st["canvas_id"]) != id || fmt.Sprint(st["kind"]) != "canvas" {
		t.Fatalf("session project %+v", st)
	}
}

func TestFindYingcePluginDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "plugins", "yingce")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo.yingce-plugin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOYO_YINGCE_PLUGINS", dir)
	if got := findYingcePluginDir(t.TempDir()); got != dir {
		t.Fatalf("got %s", got)
	}
}
