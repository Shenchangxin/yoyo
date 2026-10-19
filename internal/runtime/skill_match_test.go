package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestMatchSkillsAutoloadDomainOrchestrator(t *testing.T) {
	skills := []artifact.Skill{
		{Name: "plan-first", Description: "Keep a visible task plan with update_plan. Use for multi-step work."},
		{Name: "weekly-report", Description: "Draft a weekly status report from notes and git history."},
		{Name: "novel-to-game", Description: "Turn a novel into a playable game. Use when adapting a book into a game."},
		{Name: "novel-game-analyze", Description: "Deconstruct a novel into SOURCE_BIBLE facts for a game adaptation."},
	}
	got := MatchSkills(SkillMatchReq{
		Voice:  "把这本小说做成游戏",
		Skills: skills,
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Mode:   "on",
	})
	if len(got) == 0 {
		t.Fatal("expected suggestions")
	}
	if got[0].Name != "novel-to-game" || got[0].Action != SkillActionAutoload {
		t.Fatalf("want autoload novel-to-game, got %+v", got)
	}
}

func TestMatchSkillsHintOnlyForBundled(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "帮我写周报",
		Skills: []artifact.Skill{{Name: "weekly-report", Description: "Draft a weekly status report."}},
		Mode:   "on",
	})
	if len(got) != 1 || got[0].Name != "weekly-report" || got[0].Action != SkillActionHint {
		t.Fatalf("%+v", got)
	}
}

func TestMatchSkillsOffAndSkip(t *testing.T) {
	sk := []artifact.Skill{{Name: "novel-to-game", Description: "Turn a novel into a playable game."}}
	packs := []PackSession{{ID: "novel-to-game"}}
	if got := MatchSkills(SkillMatchReq{Voice: "把这本小说做成游戏", Skills: sk, Packs: packs, Mode: "off"}); len(got) != 0 {
		t.Fatalf("off: %+v", got)
	}
	if got := MatchSkills(SkillMatchReq{Voice: "把这本小说做成游戏，跳过技能", Skills: sk, Packs: packs, Mode: "on"}); len(got) != 0 {
		t.Fatalf("skip: %+v", got)
	}
	if got := MatchSkills(SkillMatchReq{Voice: "把这本小说做成游戏", Skills: sk, Packs: packs, Mode: "on", Depth: 1}); len(got) != 0 {
		t.Fatalf("child: %+v", got)
	}
}

func TestMatchSkillsIgnoresProcessSkillsWithoutMethodology(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "brainstorm a plan for this bug",
		Skills: []artifact.Skill{{Name: "brainstorming", Description: "Brainstorm a design before writing code."}},
		Mode:   "on",
	})
	if len(got) != 0 {
		t.Fatalf("process skill leaked: %+v", got)
	}
}

func TestMatchSkillsRespectsDisableModelInvocation(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice: "把这本小说做成游戏",
		Skills: []artifact.Skill{{
			Name:                   "novel-to-game",
			Description:            "Turn a novel into a playable game.",
			DisableModelInvocation: true,
		}},
		Packs: []PackSession{{ID: "novel-to-game"}},
		Mode:  "on",
	})
	if len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestMatchSkillsHintsInternetReach(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "帮我调研一下这个开源项目",
		Skills: []artifact.Skill{{Name: "internet-reach", Description: "Use when the operator wants internet research or 调研."}},
		Mode:   "on",
	})
	if len(got) != 1 || got[0].Name != "internet-reach" || got[0].Action != SkillActionHint {
		t.Fatalf("%+v", got)
	}
}

func TestMatchSkillsHintModeNeverAutoloads(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "把这本小说做成游戏",
		Skills: []artifact.Skill{{Name: "novel-to-game", Description: "Turn a novel into a playable game."}},
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Mode:   "hint",
	})
	if len(got) != 1 || got[0].Action != SkillActionHint {
		t.Fatalf("%+v", got)
	}
}

func TestApplySkillMatchInjectsAutoload(t *testing.T) {
	sk := []artifact.Skill{{Name: "novel-to-game", Description: "Turn a novel into a playable game.", Body: "Follow the pipeline."}}
	bodies, dirs, _ := MapSkills(sk)
	inject, hints, auto := ApplySkillMatch(SkillMatchReq{
		Voice:  "adapt this novel into a playable game",
		Skills: sk,
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Mode:   "on",
	}, bodies, dirs, 2000)
	if len(auto) != 1 || auto[0] != "novel-to-game" {
		t.Fatalf("auto %+v hints %+v", auto, hints)
	}
	if !strings.Contains(inject, "## @skill:novel-to-game") || !strings.Contains(inject, "Follow the pipeline") {
		t.Fatalf("inject %q", inject)
	}
}

func TestFormatSkillHintsOmitsAutoload(t *testing.T) {
	s := FormatSkillHints([]SkillSuggestion{
		{Name: "novel-to-game", Action: SkillActionAutoload, Reason: "named"},
		{Name: "weekly-report", Action: SkillActionHint, Reason: "description overlap"},
	})
	if strings.Contains(s, "novel-to-game") || !strings.Contains(s, "weekly-report") {
		t.Fatal(s)
	}
	if !strings.Contains(s, "## Suggested skills this turn") {
		t.Fatal(s)
	}
}

func TestMatchSkillsPrefersYoyoGameWorldOverNovel(t *testing.T) {
	skills := []artifact.Skill{
		{Name: "novel-to-game", Description: "Turn a novel into a playable game. Use when adapting a book into a game."},
		{Name: "yoyo-game", Description: "Generate a spatial 3D game world (游戏世界) from a play intent. Use when the operator wants a playable game world or third-person collect."},
	}
	got := MatchSkills(SkillMatchReq{
		Voice:  "生成一个第三人称收集游戏世界",
		Skills: skills,
		Mode:   "on",
	})
	if len(got) == 0 {
		t.Fatal("expected hints")
	}
	if got[0].Name != "yoyo-game" {
		t.Fatalf("want yoyo-game first, got %+v", got)
	}
}

func TestMatchSkillsDoesNotAutoloadNovelOnSpatialGame(t *testing.T) {
	skills := []artifact.Skill{
		{Name: "novel-to-game", Description: "Turn a novel into a playable game. Use when adapting a book into a game."},
		{Name: "yoyo-game", Description: "Generate a spatial 3D game world (游戏世界) from a play intent."},
	}
	got := MatchSkills(SkillMatchReq{
		Voice:  "按这个玩法生成世界：神庙逃亡主题的第三人称3D跑酷小游戏",
		Skills: skills,
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Loaded: []string{"yoyo-game", "game-world-analyze"},
		Lane:   "game",
		Mode:   "on",
	})
	for _, s := range got {
		if s.Name == "novel-to-game" {
			t.Fatalf("spatial game lane leaked novel-to-game: %+v", got)
		}
	}
}

func TestMatchSkillsDoesNotAutoloadNovelWithoutNovelVoice(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "按这个玩法生成世界：神庙逃亡主题的第三人称3D跑酷小游戏",
		Skills: []artifact.Skill{{Name: "novel-to-game", Description: "Turn a novel into a playable game. Use when adapting a book into a game."}},
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Mode:   "on",
	})
	for _, s := range got {
		if s.Name == "novel-to-game" && s.Action == SkillActionAutoload {
			t.Fatalf("game intent autoloaded novel-to-game: %+v", got)
		}
	}
}

func TestMatchSkillsStillAutoloadsNovelAdapt(t *testing.T) {
	got := MatchSkills(SkillMatchReq{
		Voice:  "把这本小说做成游戏",
		Skills: []artifact.Skill{{Name: "novel-to-game", Description: "Turn a novel into a playable game. Use when adapting a book into a game."}},
		Packs:  []PackSession{{ID: "novel-to-game"}},
		Mode:   "on",
	})
	if len(got) == 0 || got[0].Name != "novel-to-game" || got[0].Action != SkillActionAutoload {
		t.Fatalf("%+v", got)
	}
}

func TestHarborDoesNotCallMatcher(t *testing.T) {
	// Matcher is session-path only. Harbor tools have no SkillHints unless set.
	tools := &WorkspaceTools{}
	if FormatSkillHints(tools.SkillHints) != "" {
		t.Fatal("harbor tools must not start with hints")
	}
}
