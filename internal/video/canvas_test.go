// 行为：测试画布修订号、资源登记、工具操作、外观品牌、历史绑定和插件目录查找。

package video

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanvasProjectRevisionAndCAS 检查画布修订号冲突、本机会话、文本回放，以及媒体落成资源；缺失的 CAS 对象恢复失败。
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

// TestCanvasHTTPCreateProject 检查 POST canvas-projects 能建出带标题的画布。
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

// containsStr 判断字符串是否包含子串。
func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

// TestCanvasAgentToolsAndOps 检查画布工具名都在，并对绑定画布执行添加节点和排队生成。
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

// TestPublicAppearanceBranding 检查公开外观品牌是 Yoyo，且去品牌会去掉影策字样。
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

// TestProjectHistoryAndSession 检查新建画布出现在项目历史里，并按会话绑回画布。
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

// TestFindYingcePluginDir 检查 YOYO_YINGCE_PLUGINS 指向含插件包的目录时会被选中。
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

// TestDecodeProjectDocRejectsNull 检查 null 和空文档会被拒绝，缺少项目正文的 PUT 返回 400。
func TestDecodeProjectDocRejectsNull(t *testing.T) {
	if _, err := decodeProjectDoc([]byte("null")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := decodeProjectDoc(nil); err == nil {
		t.Fatal("expected error")
	}
	env := (&Engine{}).canvasHTTP(map[string]any{"method": "PUT", "path": "canvas-projects/c1", "body": map[string]any{}})
	if env.Code != 400 {
		t.Fatalf("missing project: %+v", env)
	}
}

// TestSaveCanvasPayloadUsesRevision 检查保存画布必须带上当前修订号，旧修订号会被拒绝。
func TestSaveCanvasPayloadUsesRevision(t *testing.T) {
	e := testEngine(t)
	doc, err := e.createEmptyCanvas("板")
	if err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprint(doc["id"])
	raw, _ := json.Marshal(doc)
	if err := e.saveCanvasPayload(id, raw, "板", 1); err != nil {
		t.Fatal(err)
	}
	if err := e.saveCanvasPayload(id, raw, "板", 1); err == nil {
		t.Fatal("expected stale revision")
	}
	if err := e.saveCanvasPayload(id, raw, "板 2", 2); err != nil {
		t.Fatal(err)
	}
}
