// 行为：测试技能 Markdown 和 ZIP 安装，以及插件开关在重开引擎后仍保持。

package video

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

// TestInstallSkillKeepsMarkdownBody 检查 Markdown 安装保留正文，空文件会被拒绝。
func TestInstallSkillKeepsMarkdownBody(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	body := []byte("---\nname: 分镜助手\ndescription: 把段落拆成镜头\n---\n\n先写场景。\n")
	env := e.canvasHTTP(map[string]any{
		"method":    "POST",
		"path":      "skills/install",
		"body":      map[string]any{"sourceType": "markdown", "tag": "drama", "file": "SKILL.md"},
		"file_b64":  base64.StdEncoding.EncodeToString(body),
		"file_name": "SKILL.md",
	})
	if env.Code != 0 {
		t.Fatalf("install: %+v", env)
	}
	skill := asMap(asMap(env.Data)["skill"])
	if skill["skillName"] != "分镜助手" || skill["description"] != "把段落拆成镜头" {
		t.Fatalf("meta: %+v", skill)
	}
	if !bytes.Contains([]byte(fmt.Sprint(skill["instruction"])), []byte("先写场景")) {
		t.Fatalf("instruction dropped: %v", skill["instruction"])
	}
	id := fmt.Sprint(skill["skillId"])
	got, err := e.getCanvasSkill(id)
	if err != nil {
		t.Fatal(err)
	}
	file, err := skillFileContent(got, "SKILL.md")
	if err != nil || !bytes.Contains([]byte(file), []byte("先写场景")) {
		t.Fatalf("stored file: %q %v", file, err)
	}

	empty := e.canvasHTTP(map[string]any{
		"method":   "POST",
		"path":     "skills/install",
		"body":     map[string]any{"sourceType": "markdown"},
		"file_b64": base64.StdEncoding.EncodeToString([]byte("   \n")),
	})
	if empty.Code == 0 {
		t.Fatal("empty markdown should fail")
	}
}

// TestInstallSkillZipReadsEntry 检查 ZIP 里的 SKILL.md 能装上，并且搜索和分类计数符合预期。
func TestInstallSkillZipReadsEntry(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("pack/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("# 广告钩子\n\n前三秒先给结果。\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	env := e.canvasHTTP(map[string]any{
		"method":    "POST",
		"path":      "skills/install",
		"body":      map[string]any{"sourceType": "zip", "tag": "ecommerce"},
		"file_b64":  base64.StdEncoding.EncodeToString(buf.Bytes()),
		"file_name": "pack.zip",
	})
	if env.Code != 0 {
		t.Fatalf("zip install: %+v", env)
	}
	skill := asMap(asMap(env.Data)["skill"])
	if skill["skillName"] != "广告钩子" {
		t.Fatalf("name: %v", skill["skillName"])
	}
	if !bytes.Contains([]byte(fmt.Sprint(skill["instruction"])), []byte("前三秒先给结果")) {
		t.Fatalf("zip body dropped: %v", skill["instruction"])
	}

	page := e.listCanvasSkills(map[string]any{"scope": "public", "search": "不存在的关键词"})
	counts, _ := page["scopeCounts"].(map[string]int)
	if counts["public"] < 70 || counts["created"] < 1 {
		t.Fatalf("scope counts should ignore search: %+v", counts)
	}
	byAuthor := e.listCanvasSkills(map[string]any{"scope": "public", "search": "工作室", "pageSize": 5})
	if intAny(byAuthor["total"]) == 0 {
		t.Fatal("author search should match builtin skills")
	}
	byLabel := e.listCanvasSkills(map[string]any{"scope": "public", "search": "短剧影视", "pageSize": 5})
	labelHits, _ := byLabel["skills"].([]map[string]any)
	if len(labelHits) == 0 {
		t.Fatal("category label search returned nothing")
	}
	for _, skill := range labelHits {
		if fmt.Sprint(skill["tag"]) != "drama" {
			t.Fatalf("category label search leaked %v", skill["tag"])
		}
	}
}

// TestPluginActivationPersists 检查插件开关会写下来，重开引擎后仍然保持。
func TestPluginActivationPersists(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	payload := e.listPluginPayload()
	plugins, _ := payload["plugins"].([]map[string]any)
	if len(plugins) == 0 {
		t.Fatal("expected bundled plugins")
	}
	manifest := asMap(plugins[0]["manifest"])
	id := fmt.Sprint(manifest["id"])
	off := e.canvasHTTP(map[string]any{
		"method": "PUT",
		"path":   "plugins/" + id + "/activation",
		"body":   map[string]any{"enabled": false},
	})
	if off.Code != 0 {
		t.Fatalf("disable: %+v", off)
	}
	state := asMap(asMap(off.Data)["state"])
	if boolAny(state["effectiveEnabled"]) {
		t.Fatalf("disable response still enabled: %+v", state)
	}

	again := e.listPluginPayload()
	states := asMap(again["states"])
	if boolAny(asMap(states[id])["effectiveEnabled"]) {
		t.Fatalf("plugin %s snapped back on", id)
	}

	on := e.canvasHTTP(map[string]any{
		"method": "PUT",
		"path":   "plugins/eagle-asset-connector/activation",
		"body":   map[string]any{"enabled": true},
	})
	if on.Code != 0 {
		t.Fatalf("eagle enable: %+v", on)
	}
	listed := asMap(e.listPluginPayload()["states"])
	eagle := asMap(listed["eagle-asset-connector"])
	if !boolAny(eagle["effectiveEnabled"]) || !boolAny(eagle["canConfigure"]) {
		t.Fatalf("eagle state: %+v", eagle)
	}

	e.Close()
	reopened, err := Open(dir, &memCAS{}, &memVault{m: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restored := asMap(reopened.listPluginPayload()["states"])
	if boolAny(asMap(restored[id])["effectiveEnabled"]) {
		t.Fatal("disabled plugin did not survive reopen")
	}
	if !boolAny(asMap(restored["eagle-asset-connector"])["effectiveEnabled"]) {
		t.Fatal("eagle enable did not survive reopen")
	}
}
