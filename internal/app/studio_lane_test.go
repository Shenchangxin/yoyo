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

func TestStudioLaneDropsWorkspaceInspectTools(t *testing.T) {
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
	a.attachStudioTools(tools, sess.ID)
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
	found := false
	for _, n := range tools.ExtraEnabled {
		if n == "studio_get" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("studio tools not enabled: %v", tools.ExtraEnabled)
	}
}

func TestStudioTitleFromLegacyPrompt(t *testing.T) {
	if got := studioTitleFromPrompt("做成短片：爱用洗衣凝珠的人天塌了"); got != "爱用洗衣凝珠的人天塌了" {
		t.Fatalf("got %q", got)
	}
}

func TestStudioJobPromptNamesProject(t *testing.T) {
	p := studioJobPrompt(video.Topic{Title: "爱用洗衣凝珠的人天塌了", Angle: "反转"}, video.StudioProject{ID: "proj-1"})
	for _, want := range []string{"studio_id: proj-1", "studio_get", "studio_update", "studio_approve_script", "studio_generate_shot", "不要检查代码仓库"} {
		if !strings.Contains(p, want) {
			t.Fatalf("missing %q in %s", want, p)
		}
	}
}

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
