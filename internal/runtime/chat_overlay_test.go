package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/tool"
)

func TestApplyChatToolMenuOmitsOffice(t *testing.T) {
	var all []string
	for _, s := range tool.HostSpecs() {
		all = append(all, s.Name)
	}
	got := ApplyChatToolMenu(all)
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "read_file") || !strings.Contains(joined, "shell") {
		t.Fatalf("core missing: %v", got)
	}
	if strings.Contains(joined, "office_create") || strings.Contains(joined, "browser_open") {
		t.Fatalf("personal schema still dumps office/browser: %v", got)
	}
}

func TestChatOverlaySchemaDefersHostTools(t *testing.T) {
	tools := &WorkspaceTools{
		ChatOverlay: true,
		Advertised:  ApplyChatToolMenu(nil),
	}
	js := AllToolJSON(tools)
	names := toolJSONNames(js)
	if !strings.Contains(names, "read_file") || !strings.Contains(names, "tool_search") {
		t.Fatalf("core/search missing: %s", names)
	}
	if strings.Contains(names, "office_create") {
		t.Fatalf("office_create should be deferred: %s", names)
	}
	search := tools.Call("tool_search", `{"query":"office"}`)
	if search.Err != nil || !strings.Contains(search.Content, "office_create") {
		t.Fatalf("tool_search should unlock host specs: %+v", search)
	}
	js = AllToolJSON(tools)
	names = toolJSONNames(js)
	if strings.Contains(names, "office_create") {
		t.Fatal("office_create must wait until checkpoint")
	}
	tools.CommitToolUnlocks()
	js = AllToolJSON(tools)
	names = toolJSONNames(js)
	if !strings.Contains(names, "office_create") {
		t.Fatal("office_create not advertised after checkpoint unlock")
	}
}

func TestHarborAdvertisedKeepsFullCatalogJail(t *testing.T) {
	tools := &WorkspaceTools{Advertised: []string{"read_file", "write_file", "shell"}}
	if tools.Call("office_create", `{"path":"a.docx"}`).Err == nil {
		t.Fatal("harbor advertised list must remain the call jail")
	}
	js := AllToolJSON(tools)
	if strings.Contains(toolJSONNames(js), "office_create") {
		t.Fatal("harbor schema leaked office")
	}
}

func TestChatOverlayCanCallDeferredHostTool(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{
		Workspace:   dir,
		ChatOverlay: true,
		Advertised:  ApplyChatToolMenu(nil),
	}
	res := tools.Call("office_create", `{"path":"week.docx","kind":"docx","title":"Week"}`)
	if res.Err != nil {
		t.Fatalf("chat overlay must keep I3 reachable: %v", res.Err)
	}
}

func TestShellCatRedirectsToReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir, ChatOverlay: true}
	res := tools.Call("shell", `{"command":"cat a.go"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "read_file") || !strings.Contains(res.Content, "package a") {
		t.Fatalf("%q", res.Content)
	}
	harbor := &WorkspaceTools{Workspace: dir}
	raw := harbor.Call("shell", `{"command":"cat a.go"}`)
	if strings.Contains(raw.Content, "redirected shell read") {
		t.Fatal("harbor must not rewrite cat")
	}
}

func TestParseShellRead(t *testing.T) {
	rel, off, lim, ok := parseShellRead(`sed -n '10,20p' webui/src/App.tsx`)
	if !ok || rel != "webui/src/App.tsx" || off != 10 || lim != 11 {
		t.Fatalf("sed %s %d %d %v", rel, off, lim, ok)
	}
	rel, off, lim, ok = parseShellRead(`head -n 5 README.md`)
	if !ok || rel != "README.md" || off != 1 || lim != 5 {
		t.Fatalf("head %s %d %d %v", rel, off, lim, ok)
	}
	if _, _, _, ok = parseShellRead(`cat a.go | wc -l`); ok {
		t.Fatal("pipeline must stay in shell")
	}
	rel, off, lim, ok = parseShellRead(`cd webui/src && sed -n '1,140p' types.ts`)
	if !ok || rel != "webui/src/types.ts" || off != 1 || lim != 140 {
		t.Fatalf("cd+sed %s %d %d %v", rel, off, lim, ok)
	}
	rel, _, _, ok = parseShellRead(`cd webui && cat package.json`)
	if !ok || rel != "webui/package.json" {
		t.Fatalf("cd+cat %s %v", rel, ok)
	}
	if _, _, _, ok = parseShellRead(`cd webui && cat a.ts b.ts`); ok {
		t.Fatal("multi-file cat must stay in shell")
	}
	rel, _, _, ok = parseShellRead(`cd /c/Users/scx/proj && cat server.log`)
	if !ok {
		t.Fatal("msys cd+cat")
	}
	slash := filepath.ToSlash(rel)
	if !strings.HasSuffix(slash, "server.log") {
		t.Fatalf("msys join %s", rel)
	}
	if strings.Contains(slash, "proj/c/Users") || strings.Contains(slash, "proj\\c\\Users") {
		t.Fatalf("doubled msys path %s", rel)
	}
}

func TestChatAllowsPlanInAnyLanguage(t *testing.T) {
	tools := &WorkspaceTools{ChatOverlay: true, OperatorVoice: "完善该项目"}
	res := tools.Call("update_plan", `{"plan":[{"step":"Recon backend","status":"in_progress"}]}`)
	if res.Err != nil {
		t.Fatalf("must not special-case Chinese: %v", res.Err)
	}
}

func TestWorkingMemoryRefreshesDuringToolRun(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	tools := &WorkspaceTools{
		Workspace: dir, Spill: sp, ChatOverlay: true,
		Skills: map[string]string{"plan-first": "Call update_plan before the first write."},
	}
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"写文件","status":"in_progress"},{"step":"验证","status":"pending"}]}`,
		}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "w1", Name: "write_file",
			Arguments: `{"path":"out.txt","content":"shipped"}`,
		}}},
		{Role: RoleAssistant, Content: "done"},
	}}}
	_, err := Run(context.Background(), RunRequest{
		SessionID:   "mem",
		User:        "完善该项目",
		Workspace:   dir,
		Tools:       tools,
		Client:      client,
		Loop:        DefaultLoop(),
		SoftHorizon: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	notes := ReadNotes(sp)
	if !strings.Contains(notes, "out.txt") {
		t.Fatalf("notes missing write path:\n%s", notes)
	}
	if !strings.Contains(notes, "写文件") {
		t.Fatalf("notes missing open plan step:\n%s", notes)
	}
	if len(client.seen) < 2 {
		t.Fatalf("chats %d", len(client.seen))
	}
	second := messagesBlob(client.seen[1])
	if !strings.Contains(second, "plan-first") || !strings.Contains(second, "update_plan") {
		t.Fatalf("open plan must re-inject plan-first:\n%s", second)
	}
}

func TestRewriteStallNudgeOnSoftHorizon(t *testing.T) {
	dir := t.TempDir()
	var steps []Message
	for i := 0; i < 3; i++ {
		steps = append(steps, Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID:        "w" + itoa(i),
			Name:      "write_file",
			Arguments: `{"path":"webui/src/App.tsx","content":"x"}`,
		}}})
	}
	steps = append(steps, Message{Role: RoleAssistant, Content: "still going"})
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: steps}}
	_, err := Run(context.Background(), RunRequest{
		User:        "完善前端",
		Workspace:   dir,
		Tools:       &WorkspaceTools{Workspace: dir},
		Client:      client,
		Loop:        DefaultLoop(),
		SoftHorizon: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, msgs := range client.seen {
		if strings.Contains(messagesBlob(msgs), "App.tsx") && strings.Contains(messagesBlob(msgs), rewriteStallPrefixEN) {
			found = true
		}
	}
	if !found {
		t.Fatal("expected rewrite stall nudge")
	}
}

func TestRewriteStallDoesNotFireOnHarbor(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.MaxTurns = 8
	var steps []Message
	for i := 0; i < 3; i++ {
		steps = append(steps, Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID:        "w" + itoa(i),
			Name:      "write_file",
			Arguments: `{"path":"App.tsx","content":"x"}`,
		}}})
	}
	steps = append(steps, Message{Role: RoleAssistant, Content: "done"})
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: steps}}
	_, err := Run(context.Background(), RunRequest{
		User:      "完善前端",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      loop,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, msgs := range client.seen {
		if strings.Contains(messagesBlob(msgs), rewriteStallPrefixZH) || strings.Contains(messagesBlob(msgs), rewriteStallPrefixEN) {
			t.Fatal("harbor must not inject chat stall nudges")
		}
	}
}

func TestSnipMarkerKeepsWritePaths(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "old"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "w1", Name: "write_file", Arguments: `{"path":"webui/src/App.tsx","content":"x"}`}}},
		{Role: RoleTool, ToolCallID: "w1", Name: "write_file", Content: "wrote"},
		{Role: RoleUser, Content: "new"},
		{Role: RoleAssistant, Content: "ok"},
	}
	out, n := snipWindow(msgs, 2, nil)
	if n == 0 || out == nil {
		t.Fatal("expected snip")
	}
	found := false
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "webui/src/App.tsx") && strings.Contains(m.Content, "elided") {
			found = true
		}
	}
	if !found {
		t.Fatalf("snip lost write paths: %+v", out)
	}
}

func TestSnipPreservesOperatorGoal(t *testing.T) {
	goal := "本项目是一个基于golang开发的RBAC权限管理系统，请完善前端"
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: goal},
	}
	for i := 0; i < 8; i++ {
		id := "r" + itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "read_file", Arguments: `{"path":"a.go"}`}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: "package a"},
		)
	}
	msgs = append(msgs, Message{Role: RoleUser, Content: "继续"})
	out, n := snipWindow(msgs, 2, nil)
	if n == 0 || out == nil {
		t.Fatal("expected snip")
	}
	found := false
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "RBAC") {
			found = true
		}
	}
	if !found {
		t.Fatalf("snip dropped the operator goal: %+v", out)
	}
}

func TestVerifyArtifactPinsWhenStoppingWithOpenPlan(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{
		Workspace:   dir,
		ChatOverlay: true,
		Skills: map[string]string{
			"plan-first":      "Call update_plan before the first write.",
			"verify-artifact": "Before you stop, list the files the task asked for and read them.",
		},
	}
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"写文件","status":"in_progress"},{"step":"验证","status":"pending"}]}`,
		}}},
		{Role: RoleAssistant, Content: "plan ready, please confirm"},
		{Role: RoleAssistant, Content: "stopping now"},
	}}}
	_, err := Run(context.Background(), RunRequest{
		User:        "完善该项目",
		Workspace:   dir,
		Tools:       tools,
		Client:      client,
		Loop:        DefaultLoop(),
		SoftHorizon: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !tools.VerifyHint {
		t.Fatal("open-plan end_turn must set verify hint")
	}
	found := false
	for i, msgs := range client.seen {
		if i == 0 {
			continue
		}
		if strings.Contains(messagesBlob(msgs), "verify-artifact") && strings.Contains(messagesBlob(msgs), "Before you stop") {
			found = true
		}
	}
	if !found {
		t.Fatal("verify-artifact must land in working memory after a premature stop")
	}
}

func toolJSONNames(js []ToolJSON) string {
	var b strings.Builder
	for _, j := range js {
		name, _ := j.Function["name"].(string)
		b.WriteString(name)
		b.WriteByte(' ')
	}
	return b.String()
}

func messagesBlob(msgs []Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
