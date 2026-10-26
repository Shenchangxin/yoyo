// 工作区预览走 resolve。http(s) 与 data: 不是工作区预览。打开工作区页面不要求真的 Chrome。

package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/browser"
)

// TestWorkspacePreviewRelAcceptsRelativeAndFileURL 确认相对路径、workspace:// 和 Windows file:/// 经 resolve 成功，https、data 和缺失文件失败。
func TestWorkspacePreviewRelAcceptsRelativeAndFileURL(t *testing.T) {
	dir := t.TempDir()
	rel := "games/pelican-bicycle.html"
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	got, err := tools.workspacePreviewRel(rel)
	if err != nil || got != rel {
		t.Fatalf("rel: %q %v", got, err)
	}
	got, err = tools.workspacePreviewRel("workspace://" + rel)
	if err != nil || got != rel {
		t.Fatalf("workspace: %q %v", got, err)
	}
	if runtime.GOOS == "windows" {
		u := "file:///" + strings.ReplaceAll(abs, `\`, "/")
		got, err = tools.workspacePreviewRel(u)
		if err != nil || got != rel {
			t.Fatalf("file url %s: %q %v", u, got, err)
		}
	}
	if _, err := tools.workspacePreviewRel("https://example.com"); err == nil {
		t.Fatal("remote url must not be a workspace preview")
	}
	if _, err := tools.workspacePreviewRel("data:text/html,hi"); err == nil {
		t.Fatal("data url must not be a workspace preview")
	}
	if _, err := tools.workspacePreviewRel("missing.html"); err == nil {
		t.Fatal("missing file must error")
	}
}

// TestBrowserOpenWorkspaceDoesNotNeedIsolatedChrome 确认工作区 HTML 在 Chrome 可执行文件不存在时仍打开，结果含 workstation Browser pane，且不要求 browser_screenshot。
func TestBrowserOpenWorkspaceDoesNotNeedIsolatedChrome(t *testing.T) {
	dir := t.TempDir()
	rel := "index.html"
	if err := os.WriteFile(filepath.Join(dir, rel), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOYO_CHROME", filepath.Join(t.TempDir(), "no-chrome.exe"))
	h, err := browser.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	tools := &WorkspaceTools{Workspace: dir, Browser: h}
	res := tools.Call("browser_open", `{"url":"index.html"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "workstation Browser pane") {
		t.Fatalf("%s", res.Content)
	}
	if !strings.Contains(res.Content, "index.html") {
		t.Fatalf("%s", res.Content)
	}
	if strings.Contains(res.Content, "Call browser_screenshot to start isolated Chrome") {
		t.Fatalf("open must not defer Chrome to screenshot: %s", res.Content)
	}
	if !strings.Contains(filepath.ToSlash(h.PreviewPath()), "index.html") {
		t.Fatalf("preview %s", h.PreviewPath())
	}
}

// TestBrowserOpenPathBeatsLocalhostURL 确认同时给出 path 和 localhost url 时采用 path。
func TestBrowserOpenPathBeatsLocalhostURL(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join(dir, "pelican-bike")
	if err := os.MkdirAll(rel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rel, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("browser_open", `{"path":"pelican-bike/index.html","url":"http://localhost/"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if strings.Contains(res.Content, "localhost") || !strings.Contains(res.Content, "pelican-bike/index.html") {
		t.Fatalf("%s", res.Content)
	}
}

// TestBrowserOpenRejectsDataURL 确认 data: 被拒绝。
func TestBrowserOpenRejectsDataURL(t *testing.T) {
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("browser_open", `{"url":"data:text/html,<h1>x</h1>"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "data:") {
		t.Fatalf("%+v", res)
	}
}

// TestParseBrowserOpenStripsLane 确认 lane=attached 与 profile=github 从目标文本里剥掉。
func TestParseBrowserOpenStripsLane(t *testing.T) {
	target, lane, profile := parseBrowserOpen("https://example.com lane=attached profile=github")
	if lane != "attached" || profile != "github" || strings.Contains(target, "lane=") || strings.Contains(target, "profile=") {
		t.Fatalf("%q %q %q", target, lane, profile)
	}
}

// TestBrowserHandlePrefersRef 确认 ref 优先于 selector，并补上 ref=。
func TestBrowserHandlePrefersRef(t *testing.T) {
	got := browserHandle(map[string]any{"ref": "btn_0", "selector": "#old"})
	if got != "ref=btn_0" {
		t.Fatalf("%q", got)
	}
	got = browserHandle(map[string]any{"selector": "text=Start"})
	if got != "text=Start" {
		t.Fatalf("%q", got)
	}
}

// TestPersistRecordedSkillWritesWorkspace 确认录制技能写到 .yoyo/skills/login-github/SKILL.md，并装进 Skills。
func TestPersistRecordedSkillWritesWorkspace(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir, Skills: map[string]string{}}
	sess := &browser.RecordSession{
		ID:        "r1",
		SkillName: "login-github",
		SkillBody: "---\nname: login-github\n---\n# x\n",
		Events:    []browser.RecordEvent{{Kind: "click"}},
		Steps:     []browser.SemanticStep{{Kind: "click"}},
	}
	res := tools.persistRecordedSkill(sess, "")
	if res.Err != nil || !strings.Contains(res.Content, "skill=") {
		t.Fatalf("%+v", res)
	}
	rel := filepath.Join(".yoyo", "skills", "login-github", "SKILL.md")
	if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
		t.Fatal(err)
	}
	if tools.Skills["login-github"] == "" {
		t.Fatal("skill not loaded into session map")
	}
}
