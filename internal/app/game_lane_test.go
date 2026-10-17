package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/game"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

func attachVideoLanes(a *App, tools *runtime.WorkspaceTools, sessionID string) {
	a.attachCutTools(tools, sessionID)
	a.attachStudioTools(tools, sessionID)
	a.attachGameTools(tools, sessionID)
	a.attachConductorTools(tools, sessionID)
}

func TestConductorLaneRegistersGameTools(t *testing.T) {
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
	attachVideoLanes(a, tools, sess.ID)
	if _, ok := tools.Extra["game_analyze_scene"]; !ok {
		t.Fatal("game_* must be in Extra so tool_search can find them before bind")
	}
	enabled := strings.Join(tools.ExtraEnabled, " ")
	if !strings.Contains(enabled, "game_create") {
		t.Fatalf("conductor should enable game_create: %v", tools.ExtraEnabled)
	}
	if strings.Contains(enabled, "game_compile") {
		t.Fatalf("unbound conductor should not dump the full pipeline: %v", tools.ExtraEnabled)
	}
	hit := tools.Call("tool_search", `{"query":"game_analyze_scene game_plan_rules game_compile"}`)
	if hit.Err != nil || !strings.Contains(hit.Content, `"name":"game_analyze_scene"`) {
		t.Fatalf("tool_search should hit game_analyze_scene: %+v", hit)
	}
}

func TestGameBindLeavesConductorAndEnablesPipeline(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Game == nil {
		t.Skip("game engine missing")
	}
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Game.Create(game.Project{Title: "Relic run", Intent: "Collect three relics and reach the shrine."})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Game.Bind(sess.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if a.conductorLane(sess.ID) {
		t.Fatal("bound game must leave conductor lane")
	}
	if !a.gameLane(sess.ID) {
		t.Fatal("expected game lane")
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	attachVideoLanes(a, tools, sess.ID)
	foundCompile := false
	for _, n := range tools.ExtraEnabled {
		if n == "game_compile" {
			foundCompile = true
		}
		if n == "cut_get" {
			t.Fatalf("game lane should drop cut extras: %v", tools.ExtraEnabled)
		}
	}
	if !foundCompile {
		t.Fatalf("game_compile not enabled: %v", tools.ExtraEnabled)
	}
	tools.Packs = []runtime.PackSession{{ID: "novel-to-game"}, {ID: "superpowers", Methodology: true}}
	tools.Loaded = []string{"novel-to-game", "yoyo-game"}
	a.attachGameTools(tools, sess.ID)
	for _, p := range tools.Packs {
		if p.ID == "novel-to-game" {
			t.Fatal("game lane must drop novel-to-game pack")
		}
	}
	for _, n := range tools.Loaded {
		if n == "novel-to-game" {
			t.Fatal("game lane must drop loaded novel-to-game")
		}
	}
}

func TestVideoRouteGameUnlocksMidTurn(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Video == nil || a.Game == nil {
		t.Skip("video/game engine missing")
	}
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
	}
	attachVideoLanes(a, tools, sess.ID)
	out, err := a.Video.CutTool(sess.ID, "video_route", map[string]any{
		"dest": "game", "title": "神庙逃亡", "intent": "Collect four relics and reach the gate.",
	})
	if err != nil {
		t.Fatal(err)
	}
	a.promoteGameTools(tools, sess.ID)
	var row map[string]any
	if err := json.Unmarshal([]byte(out), &row); err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(fmt.Sprint(row["id"]))
	if id == "" || id == "<nil>" {
		t.Fatalf("route id: %s", out)
	}
	p, err := a.Game.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stage == game.StageDraft {
		t.Fatalf("video_route dest=game with intent should compile, stage=%s", p.Stage)
	}
	snap := a.Game.Snapshot(p)
	playable := strings.TrimSpace(fmt.Sprint(snap["playable"]))
	if playable == "" || playable == "<nil>" {
		t.Fatalf("missing playable: %#v", snap)
	}
	abs := filepath.Join(a.VideoWorkspace(), filepath.FromSlash(playable))
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("playable missing on disk: %s %v", abs, err)
	}
	found := false
	for _, n := range tools.ExtraEnabled {
		if n == "game_analyze_scene" {
			found = true
		}
	}
	if !found {
		t.Fatalf("mid-turn bind must enable the pipeline: %v", tools.ExtraEnabled)
	}
	if a.conductorLane(sess.ID) {
		t.Fatal("after video_route dest=game the session is not conductor")
	}
}

func TestApplyGameLanePrompt(t *testing.T) {
	loop := runtime.DefaultLoop()
	loop, frags := applyGameLanePrompt(loop, nil, game.Project{})
	if loop.Bootstrap != gameBootstrap {
		t.Fatal(loop.Bootstrap)
	}
	if len(frags) == 0 || frags[len(frags)-1].Text != gameRuntimePin {
		t.Fatalf("%+v", frags)
	}
	bound := game.Project{ID: "world-1", Title: "神庙逃亡", Stage: "draft", Intent: "神庙逃亡主题的第三人称3D跑酷"}
	loop, frags = applyGameLanePrompt(artifact.LoopPreset{}, []artifact.PromptFragment{{Slot: "role", Text: "Inspect the workspace first."}}, bound)
	if !strings.Contains(loop.Bootstrap, "id=world-1") || !strings.Contains(loop.Bootstrap, "Do not call game_create") {
		t.Fatal(loop.Bootstrap)
	}
	if !strings.Contains(loop.Bootstrap, "Do not load_skill novel-to-game") {
		t.Fatal(loop.Bootstrap)
	}
	for _, f := range frags {
		if strings.Contains(strings.ToLower(f.Text), "inspect the workspace") {
			t.Fatal(f.Text)
		}
	}
}

func TestGamePlayHonorsFullAccess(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	asked := 0
	a.Caps = capability.NewBroker(capability.AutoPolicy{}, func(_ context.Context, req capability.Request) (capability.Decision, error) {
		asked++
		return capability.Deny, fmt.Errorf("should not ask for %s", req.Action)
	})
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	a.Caps.ApplyAuthMode(sess.ID, capability.AuthFull)
	if err := a.checkGameTool(sess.ID, "game_play", true, false); err != nil {
		t.Fatal(err)
	}
	if err := a.checkGameTool(sess.ID, "game_verify", true, false); err != nil {
		t.Fatal(err)
	}
	if asked != 0 {
		t.Fatalf("full access still asked for host game tools: %d", asked)
	}
}
