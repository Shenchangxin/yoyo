// 锁住空会话的对话 token 为 0、聊天目录预算是帽、路径规则在读到目录后才进入动态尾，以及延后工具名字不进 tool_search 描述。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestEmptyChatHasNoConversationTokens 确认新会话的 ChatTokens 和技能正文为 0，项目记忆只留路径，延后工具的 schema 不进开场数组。
func TestEmptyChatHasNoConversationTokens(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "YOYO.memory.md"), []byte(strings.Repeat("remember this fact. ", 80)), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace:   dir,
		ChatOverlay: true,
		CatalogLane: "agent",
		Advertised:  ApplyChatToolMenu(nil),
		Skills: map[string]string{
			"studio-script": "long body",
			"yoyo-conduct":  "playbook",
		},
	}
	tools.Packs = []PackSession{{ID: "superpowers", BootstrapSkill: "using-superpowers", Methodology: true}}
	ApplyPackBootstrap(tools)
	rep := MeasureContext(MeasureOpts{
		Loop:          ApplyChatHorizon(DefaultLoop()),
		Workspace:     dir,
		Tools:         tools,
		Skills:        []artifact.Skill{{Name: "studio-script", Description: strings.Repeat("d", 400)}, {Name: "yoyo-conduct", Description: "Failure playbook."}},
		ModelWindow:   128_000,
		ProfileMemory: "- likes short answers\n",
	})
	if rep.ChatTokens != 0 {
		t.Fatalf("new session chat=%d bill=%+v", rep.ChatTokens, rep)
	}
	if rep.SkillBodyTokens != 0 {
		t.Fatalf("skill body preloaded: %d", rep.SkillBodyTokens)
	}
	if rep.MemoryTokens > 80 {
		t.Fatalf("project memory was inlined: %d", rep.MemoryTokens)
	}
	if !strings.Contains(ProjectMemoryPin(dir), "YOYO.memory.md") || strings.Contains(ProjectMemoryPin(dir), "remember this fact") {
		t.Fatal("memory pin should be a path")
	}
	names := toolJSONNames(AllToolJSON(tools))
	if strings.Contains(names, "office_create") || strings.Contains(names, "browser_verify") {
		t.Fatalf("deferred schema leaked: %s", names)
	}
	if rep.SchemaTokens > 8_000 {
		t.Fatalf("opening schemas still bloated: %d", rep.SchemaTokens)
	}
}

// TestChatCatalogBudgetIsACeiling 确认聊天目录预算以 ChatSkillCatalogBudget 为帽，更小的显式预算保留；没有 ChatOverlay 时用调用方给的预算。
func TestChatCatalogBudgetIsACeiling(t *testing.T) {
	fat := pinsOptsFromTools(&WorkspaceTools{ChatOverlay: true, SkillCatalogBudget: 80_000}, true)
	if fat.CatalogBudget != ChatSkillCatalogBudget {
		t.Fatalf("chat catalog %d", fat.CatalogBudget)
	}
	small := pinsOptsFromTools(&WorkspaceTools{ChatOverlay: true, SkillCatalogBudget: 1000}, true)
	if small.CatalogBudget != 1000 {
		t.Fatalf("smaller budget %d", small.CatalogBudget)
	}
	harbor := pinsOptsFromTools(&WorkspaceTools{SkillCatalogBudget: 80_000}, false)
	if harbor.CatalogBudget != 80_000 {
		t.Fatalf("harbor catalog %d", harbor.CatalogBudget)
	}
}

// TestCatalogLaneAndOneLineDescription 确认 agent 通道不列出 studio-script，描述压成一行，截断字样不像规则截断。
func TestCatalogLaneAndOneLineDescription(t *testing.T) {
	long := strings.Repeat("word ", 80)
	got := renderSkillCatalog([]artifact.Skill{
		{Name: "studio-script", Description: long},
		{Name: "yoyo-conduct", Description: long},
	}, PinsOpts{Lane: "agent", CatalogBudget: 4000, ChatCatalog: true})
	if strings.Contains(got, "studio-script:") {
		t.Fatalf("studio skill listed on agent lane:\n%s", got)
	}
	if !strings.Contains(got, "yoyo-conduct:") {
		t.Fatalf("conduct missing:\n%s", got)
	}
	if strings.Count(got, "word") > 40 {
		t.Fatalf("description was not one line:\n%s", got)
	}
	if strings.Contains(got, "YOYO.md truncated") {
		t.Fatalf("catalog cap must not look like a rules truncation:\n%s", got)
	}
}

// TestPathRulesLoadOnRead 确认读到 web/app.go 后嵌套规则进入路径规则和笔记，根上那份第一命中不重复进动态尾，也不进稳定前缀的规则层。
func TestPathRulesLoadOnRead(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("root rule stays pinned"), 0o644); err != nil {
		t.Fatal(err)
	}
	web := filepath.Join(root, "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "AGENTS.md"), []byte("use the web router"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "app.go"), []byte("package web\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: root}
	if rules := tools.PathRules(); rules != "" {
		t.Fatalf("rules before read: %s", rules)
	}
	res := tools.Call("read_file", `{"path":"web/app.go"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(tools.PathRules(), "use the web router") {
		t.Fatalf("nested rule missing: %s", tools.PathRules())
	}
	if strings.Contains(tools.PathRules(), "root rule stays pinned") {
		t.Fatal("root pin was copied into the tail")
	}
	bill := buildBill(billInput{Loop: ApplyChatHorizon(DefaultLoop()), Workspace: root, Tools: tools, Soft: true})
	if strings.Contains(bill.Rules, "use the web router") {
		t.Fatal("nested rule entered the stable prefix")
	}
	if !strings.Contains(bill.Notes, "use the web router") {
		t.Fatal("nested rule missing from notes tail")
	}
}

// TestHostToolIndexOmitsSchemasFromDescription 确认延后工具的名字在 .yoyo/tools/INDEX.md 里，不出现在 tool_search 的描述中。
func TestHostToolIndexOmitsSchemasFromDescription(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir, ChatOverlay: true, Advertised: ApplyChatToolMenu(nil)}
	js := AllToolJSON(tools)
	for _, j := range js {
		name, _ := j.Function["name"].(string)
		if name != "tool_search" {
			continue
		}
		desc, _ := j.Function["description"].(string)
		if strings.Contains(desc, "office_create") || strings.Contains(desc, "Deferred extra tools:") {
			t.Fatalf("deferred names leaked into tool_search: %s", desc)
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".yoyo", "tools", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "office_create") || !strings.Contains(string(raw), "connector_read") {
		t.Fatalf("index missing deferred names:\n%s", raw)
	}
}
