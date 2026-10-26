// 锁住未绑定的视频频道走指挥席而不是短片工具，画布不重复注册 ask_user，以及短片提示会换掉检查仓库的开场白。

package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/video"
)

// TestConductorLaneGetsCutToolsNotStudio 锁住未绑定的视频会话有 cut_get 和 video_route，没有 studio_get，并且 shell、list_dir、glob、grep 不在广告名和允许名里。
func TestConductorLaneGetsCutToolsNotStudio(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	a.attachPersonal(tools)
	a.attachCutTools(tools, sess.ID)
	a.attachStudioTools(tools, sess.ID)
	a.attachConductorTools(tools, sess.ID)
	for _, n := range []string{"shell", "list_dir", "glob", "grep"} {
		for _, got := range tools.Advertised {
			if got == n {
				t.Fatalf("%s still advertised", n)
			}
		}
		for _, got := range tools.AllowedTools {
			if got == n {
				t.Fatalf("%s still allowed", n)
			}
		}
	}
	foundCut, foundRoute, foundStudio := false, false, false
	for _, n := range tools.ExtraEnabled {
		if n == "cut_get" {
			foundCut = true
		}
		if n == "video_route" {
			foundRoute = true
		}
		if n == "studio_get" {
			foundStudio = true
		}
	}
	if !foundCut || !foundRoute {
		t.Fatalf("conductor tools missing: %v", tools.ExtraEnabled)
	}
	if foundStudio {
		t.Fatalf("unbound video channel should not attach studio tools: %v", tools.ExtraEnabled)
	}
	seen := map[string]int{}
	for _, j := range runtime.AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		seen[name]++
		if seen[name] > 1 {
			t.Fatalf("duplicate tool %s", name)
		}
	}
	if seen["cut_get"] != 1 || seen["video_route"] != 1 || seen["ask_user"] != 1 {
		t.Fatalf("conductor tool list %#v", seen)
	}
}

// TestCanvasAskUserDoesNotDuplicateHostTool 锁住画布工具挂上后 ask_user 只出现一次，且不在 Extra 里再注册。
func TestCanvasAskUserDoesNotDuplicateHostTool(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Video == nil {
		t.Skip("video engine missing")
	}
	raw, err := a.Video.CanvasCall("canvas.projects.create", map[string]any{"title": "画布"})
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := raw.(map[string]any)
	id, _ := doc["id"].(string)
	if id == "" {
		t.Fatalf("canvas id %#v", raw)
	}
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Video.BindCanvasSession(sess.ID, id); err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	a.attachPersonal(tools)
	a.attachCanvasTools(tools, sess.ID)
	a.attachCutTools(tools, sess.ID)
	seen := map[string]int{}
	for _, j := range runtime.AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		seen[name]++
		if seen[name] > 1 {
			t.Fatalf("duplicate tool %s", name)
		}
	}
	if seen["ask_user"] != 1 {
		t.Fatalf("ask_user count %d", seen["ask_user"])
	}
	if _, ok := tools.Extra["ask_user"]; ok {
		t.Fatal("canvas ask_user should not be registered beside the host tool")
	}
}

// TestStudioBoundAttachesStudioTools 锁住已绑定短片项目的会话启用 studio_get，并且不再带指挥席的 video_route。
func TestStudioBoundAttachesStudioTools(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Video == nil {
		t.Skip("video engine missing")
	}
	p, err := a.Video.CreateStudioProject(video.StudioProject{Title: "通勤穿搭"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Video.BindStudioSession(sess.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	a.attachCutTools(tools, sess.ID)
	a.attachStudioTools(tools, sess.ID)
	a.attachConductorTools(tools, sess.ID)
	found := false
	for _, n := range tools.ExtraEnabled {
		if n == "studio_get" {
			found = true
		}
		if n == "video_route" {
			t.Fatalf("bound shorts should not keep conductor video_route: %v", tools.ExtraEnabled)
		}
	}
	if !found {
		t.Fatalf("studio tools not enabled: %v", tools.ExtraEnabled)
	}
}

// TestOpenCutReturnsConductorLane 锁住短片项目送去剪辑后，工具回到 cut_get 和 video_route，studio_get 消失。
func TestOpenCutReturnsConductorLane(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Video == nil {
		t.Skip("video engine missing")
	}
	p, err := a.Video.CreateStudioProject(video.StudioProject{Title: "通勤穿搭"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Video.BindStudioSession(sess.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	hash, err := a.Video.PutBytes([]byte("open-cut-source"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Video.OpenCut(video.CutOpen{Title: "通勤穿搭", Hash: hash, SessionID: sess.ID}); err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	a.attachCutTools(tools, sess.ID)
	a.attachStudioTools(tools, sess.ID)
	a.attachConductorTools(tools, sess.ID)
	foundCut, foundRoute, foundStudio := false, false, false
	for _, n := range tools.ExtraEnabled {
		if n == "cut_get" {
			foundCut = true
		}
		if n == "video_route" {
			foundRoute = true
		}
		if n == "studio_get" {
			foundStudio = true
		}
	}
	if !foundCut || !foundRoute {
		t.Fatalf("conductor tools missing after send-to-cut: %v", tools.ExtraEnabled)
	}
	if foundStudio {
		t.Fatalf("studio tools should drop after send-to-cut: %v", tools.ExtraEnabled)
	}
}

// TestStudioTitleFromLegacyPrompt 锁住做成短片：后面的标题被 studioTitleFromPrompt 留下。
func TestStudioTitleFromLegacyPrompt(t *testing.T) {
	if got := studioTitleFromPrompt("做成短片：爱用洗衣凝珠的人天塌了"); got != "爱用洗衣凝珠的人天塌了" {
		t.Fatalf("got %q", got)
	}
}

// TestStudioJobPromptNamesProject 锁住已有项目 id 的提示包含 studio_id、studio_get、studio_update、studio_approve_script 和 studio_generate_shot。
func TestStudioJobPromptNamesProject(t *testing.T) {
	p := studioJobPrompt(video.Topic{Title: "爱用洗衣凝珠的人天塌了", Angle: "反转"}, video.StudioProject{ID: "proj-1"})
	for _, want := range []string{"studio_id: proj-1", "studio_get", "studio_update", "studio_approve_script", "studio_generate_shot", "不要检查代码仓库"} {
		if !strings.Contains(p, want) {
			t.Fatalf("missing %q in %s", want, p)
		}
	}
}

// TestApplyStudioLanePromptDropsInspect 锁住短片车道会去掉检查工作区的开场白，并放上 studioRuntimePin。
func TestApplyStudioLanePromptDropsInspect(t *testing.T) {
	loop := runtime.DefaultLoop()
	frags := []artifact.PromptFragment{{Slot: "bootstrap", Text: "Inspect the workspace first."}}
	loop, frags = applyStudioLanePrompt(loop, frags)
	if strings.Contains(strings.ToLower(loop.Bootstrap), "inspect the workspace") {
		t.Fatal(loop.Bootstrap)
	}
	sawPin := false
	for _, f := range frags {
		if strings.Contains(strings.ToLower(f.Text), "inspect the workspace") {
			t.Fatal(f.Text)
		}
		if f.Text == studioRuntimePin {
			sawPin = true
		}
	}
	if !sawPin {
		t.Fatal("missing studio runtime pin")
	}
}

// TestApplyConductorLanePrompt 锁住指挥席开场白等于 conductorBootstrap，最后一条片段是 conductorRuntimePin。
func TestApplyConductorLanePrompt(t *testing.T) {
	loop := runtime.DefaultLoop()
	loop, frags := applyConductorLanePrompt(loop, nil)
	if loop.Bootstrap != conductorBootstrap {
		t.Fatal(loop.Bootstrap)
	}
	if len(frags) == 0 || frags[len(frags)-1].Text != conductorRuntimePin {
		t.Fatalf("%+v", frags)
	}
}
