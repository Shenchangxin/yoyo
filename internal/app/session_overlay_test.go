// overlay 保留基线、只钉当前会话、复用回合和拒绝视频。

package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// overlayArmedSolver 只在系统提示里有验证说明、且任务提到 hello.txt 时才写文件。基线提示写不出文件。
type overlayArmedSolver struct{}

// Chat 在系统提示武装之后才发出 write_file，否则只回复已经看过工作区。
func (overlayArmedSolver) Chat(_ context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	var sys, user string
	for _, m := range req.Messages {
		switch m.Role {
		case runtime.RoleSystem, runtime.RoleDeveloper:
			sys += m.Content
		case runtime.RoleUser:
			user += m.Content
		}
	}
	armed := strings.Contains(sys, "Create required output artifacts early")
	if strings.Contains(user, "hello.txt") && armed && !overlayToolWrote(req, "hello.txt") {
		args, _ := json.Marshal(map[string]string{"path": "hello.txt", "content": "hello"})
		return runtime.Message{Role: runtime.RoleAssistant, ToolCalls: []runtime.ToolCall{{
			ID: "w1", Name: "write_file", Arguments: string(args),
		}}}, nil
	}
	return runtime.Message{Role: runtime.RoleAssistant, Content: "explored workspace, no file written"}, nil
}

// overlayToolWrote 判断请求历史里是否已经写过这个文件。
func overlayToolWrote(req runtime.ChatRequest, name string) bool {
	for _, m := range req.Messages {
		if m.Role != runtime.RoleTool && m.Role != runtime.RoleAssistant {
			continue
		}
		if strings.Contains(m.Content, "wrote "+name) {
			return true
		}
		for _, tc := range m.ToolCalls {
			if tc.Name == "write_file" && strings.Contains(tc.Arguments, name) {
				return true
			}
		}
	}
	return false
}

// overlayRefs 读出 active、canary 和 staging 三个引用。
func overlayRefs(a *app.App) (active, canary, staging string) {
	return a.ActiveHash(), a.Refs.GetOrEmpty(artifact.RefCanary), a.Refs.GetOrEmpty(artifact.RefStaging)
}

// assertOverlayRefsFrozen 确认三个引用都没有被改写。
func assertOverlayRefsFrozen(t *testing.T, a *app.App, active, canary, staging string) {
	t.Helper()
	gotA, gotC, gotS := overlayRefs(a)
	if gotA != active || gotC != canary || gotS != staging {
		t.Fatalf("refs moved: active %s/%s canary %s/%s staging %s/%s", gotA, active, gotC, canary, gotS, staging)
	}
}

// TestOverlaySessionKeepBaseLeavesFollowActive 确认保留基线时不钉会话、不改引用、不写操作者工作区。
func TestOverlaySessionKeepBaseLeavesFollowActive(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	active, canary, staging := overlayRefs(a)
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "marker.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	if sess.HarnessPolicy != session.FollowActive {
		t.Fatalf("policy %s", sess.HarnessPolicy)
	}
	res, err := a.OverlaySession(context.Background(), app.OverlayRun{
		SessionID: sess.ID,
		Task:      "Write a file named hello.txt with the word hello",
		Client:    runtime.FailingSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Adopted || !res.KeptBase {
		t.Fatalf("want keep_base: %+v", res)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HarnessPolicy != session.FollowActive {
		t.Fatalf("KEEP_BASE must not pin: %s", got.HarnessPolicy)
	}
	if got.Harness != sess.Harness {
		t.Fatalf("harness %s want %s", got.Harness, sess.Harness)
	}
	assertOverlayRefsFrozen(t, a, active, canary, staging)
	if _, err := os.Stat(filepath.Join(a.Home.Library(), evolve.DomainSlug(ws), "INDEX.md")); !os.IsNotExist(err) {
		t.Fatal("KEEP_BASE must not write the experience library")
	}
	if _, err := os.Stat(filepath.Join(ws, "hello.txt")); !os.IsNotExist(err) {
		t.Fatal("probe must not write the operator workspace")
	}
	b, err := os.ReadFile(filepath.Join(ws, "marker.txt"))
	if err != nil || string(b) != "keep" {
		t.Fatalf("operator tree mutated: %s %v", b, err)
	}
}

// TestOverlaySessionAdoptPinsOnlyThisSession 确认采纳后只钉当前会话的 harness，active、canary 和 staging 不动。
func TestOverlaySessionAdoptPinsOnlyThisSession(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	active, canary, staging := overlayRefs(a)
	ws := t.TempDir()
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	parent := session.ResolveHarness(session.NormalizePolicy(sess.HarnessPolicy), sess.Harness, a.ActiveHash())
	res, err := a.OverlaySession(context.Background(), app.OverlayRun{
		SessionID: sess.ID,
		Task:      "Write a file named hello.txt with the word hello",
		Client:    overlayArmedSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Adopted || res.KeptBase {
		t.Fatalf("want adopt: %+v", res)
	}
	if res.Hash == "" || res.Hash == parent {
		t.Fatalf("expected child snapshot, got %s parent %s", res.Hash, parent)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HarnessPolicy != session.Pin || got.Harness != res.Hash {
		t.Fatalf("pin %+v hash %s", got.HarnessPolicy, got.Harness)
	}
	snap, err := a.LoadSnapshot(res.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Parent != parent {
		t.Fatalf("parent %s want %s", snap.Parent, parent)
	}
	parentLoop, _, _, _, _, _, err := a.Materials(parent)
	if err != nil {
		t.Fatal(err)
	}
	childLoop, _, _, _, _, _, err := a.Materials(res.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if !artifact.LoopTopologyEqual(parentLoop, childLoop) {
		t.Fatalf("l3 topology drifted")
	}
	assertOverlayRefsFrozen(t, a, active, canary, staging)
	if res.LibraryStatus != evolve.LibraryAdded || res.Experience == nil {
		t.Fatalf("library %s", res.LibraryStatus)
	}
	idx := filepath.Join(a.Home.Library(), evolve.DomainSlug(ws), "INDEX.md")
	b, err := os.ReadFile(idx)
	if err != nil || !strings.Contains(string(b), res.Experience.Cue) {
		t.Fatalf("index %s %v", b, err)
	}
	if a.ActiveHash() != active {
		t.Fatal("checkout still owns production")
	}
	if _, err := os.Stat(filepath.Join(ws, "hello.txt")); !os.IsNotExist(err) {
		t.Fatal("adopt probe must not merge back onto the operator workspace")
	}
}

// TestOverlaySessionReuseTurnCachesBase 确认复用上一轮时基线标记为缓存，失败的交付不算成功。
func TestOverlaySessionReuseTurnCachesBase(t *testing.T) {
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
	if _, err := a.Send(context.Background(), sess.ID, "Write a file named hello.txt with the word hello", runtime.FailingSolver{}, nil); err != nil {
		t.Fatal(err)
	}
	active, canary, staging := overlayRefs(a)
	res, err := a.OverlaySession(context.Background(), app.OverlayRun{
		SessionID: sess.ID,
		ReuseTurn: true,
		Client:    overlayArmedSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rollouts) == 0 || !res.Rollouts[0].Cached {
		t.Fatalf("base should be cached: %+v", res.Rollouts)
	}
	if res.Rollouts[0].Delivered {
		t.Fatal("cached failing turn must not count as delivered")
	}
	if !res.Adopted {
		t.Fatalf("want adopt after cached miss: %+v", res)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HarnessPolicy != session.Pin {
		t.Fatalf("policy %s", got.HarnessPolicy)
	}
	assertOverlayRefsFrozen(t, a, active, canary, staging)
}

// TestOverlaySessionHeldOutNotInEvidence 确认 held-out 的题号不出现在 overlay 证据里。
func TestOverlaySessionHeldOutNotInEvidence(t *testing.T) {
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
	res, err := a.OverlaySession(context.Background(), app.OverlayRun{
		SessionID: sess.ID,
		Task:      "do not solve write-answer; Write a file named hello.txt with the word hello",
		Client:    overlayArmedSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	blob := strings.Join(res.Evidence, "\n")
	if strings.Contains(blob, "write-answer") {
		t.Fatalf("held-out leaked: %s", blob)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
}

// TestOverlaySessionRejectsVideo 确认视频会话不能跑 overlay。
func TestOverlaySessionRejectsVideo(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSessionOn(t.TempDir(), session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.OverlaySession(context.Background(), app.OverlayRun{
		SessionID: sess.ID,
		Task:      "Write a file named hello.txt with the word hello",
		Client:    runtime.FailingSolver{},
	})
	if err == nil {
		t.Fatal("video overlay")
	}
}
