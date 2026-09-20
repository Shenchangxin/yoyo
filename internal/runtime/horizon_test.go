package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestDefaultLoopHorizonUnchanged(t *testing.T) {
	loop := DefaultLoop()
	if loop.MaxTurns != 32 || loop.MaxToolMessages != 40 {
		t.Fatalf("harbor loop drifted: %+v", loop)
	}
	if loop.MicroKeep != 4 {
		t.Fatalf("harbor microkeep %d", loop.MicroKeep)
	}
	if loop.CompactionTokens != 24_000 {
		t.Fatalf("compaction %d", loop.CompactionTokens)
	}
}

func TestApplyChatHorizon(t *testing.T) {
	got := ApplyChatHorizon(DefaultLoop())
	if got.MaxTurns < 64 || got.MaxToolMessages < 160 || got.MicroKeep < 16 {
		t.Fatalf("%+v", got)
	}
	wide := DefaultLoop()
	wide.MaxTurns = 99
	wide.MaxToolMessages = 120
	got = ApplyChatHorizon(wide)
	if got.MaxTurns != 99 {
		t.Fatalf("must not shrink turns: %+v", got)
	}
	if got.MaxToolMessages < 160 {
		t.Fatalf("chat tool floor %+v", got)
	}
	wide.MaxToolMessages = 200
	wide.MicroKeep = 24
	got = ApplyChatHorizon(wide)
	if got.MaxToolMessages != 200 || got.MicroKeep != 24 {
		t.Fatalf("must not shrink large CAS: %+v", got)
	}
	if DefaultLoop().MaxTurns != 32 {
		t.Fatal("ApplyChatHorizon mutated DefaultLoop")
	}
	if DefaultLoop().MaxToolMessages != 40 {
		t.Fatal("ApplyChatHorizon mutated Harbor tool cap")
	}
}

func TestChatConductDoesNotTouchHarborLoop(t *testing.T) {
	base := DefaultLoop()
	frags := ChatConductFragments("完善该项目，前端用 react+ts", "zh-CN", false)
	if len(frags) != 1 || !strings.Contains(frags[0].Text, "中文") {
		t.Fatalf("%+v", frags)
	}
	if !strings.Contains(frags[0].Text, "不要先写长篇评估") {
		t.Fatal("missing start-work")
	}
	if !strings.Contains(frags[0].Text, "步骤") || !strings.Contains(frags[0].Text, "整树重写") {
		t.Fatal("missing plan-language and no-blind-rewrite")
	}
	got := DefaultLoop()
	if got.Execution != base.Execution || got.Bootstrap != base.Bootstrap || got.MaxTurns != 32 {
		t.Fatalf("harbor loop mutated: %+v", got)
	}
	plan := ChatConductFragments("完善该项目", "zh-CN", true)
	if strings.Contains(plan[0].Text, "不要先写长篇评估") {
		t.Fatal("plan mode must not force writes")
	}
	if !strings.Contains(plan[0].Text, "计划步骤") {
		t.Fatal("plan mode must still match language")
	}
	en := ChatConductFragments("fix the build", "en", false)
	if !strings.Contains(en[0].Text, "same language") {
		t.Fatalf("%q", en[0].Text)
	}
}

func TestOperatorVoiceSkipsControl(t *testing.T) {
	got := OperatorVoice(toolBudgetNudge, []Message{
		{Role: RoleUser, Content: "完善前端"},
		{Role: RoleUser, Content: toolBudgetNudge},
	})
	if got != "完善前端" {
		t.Fatalf("%q", got)
	}
}

func TestAssembleIdentitySingleBootstrap(t *testing.T) {
	loop := DefaultLoop()
	dup := loop.Bootstrap
	s := AssembleIdentity(loop, []artifact.PromptFragment{
		{Slot: "bootstrap", Text: dup},
		{Slot: "bootstrap", Text: "seed extra bootstrap"},
		{Slot: "verification", Text: "write a placeholder file first"},
		{Slot: "runtime", Text: "runtime pin"},
	})
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "## bootstrap") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("bootstrap headings=%d\n%s", n, s)
	}
	if strings.Count(s, dup) != 1 {
		t.Fatalf("exact bootstrap duplicate survived\n%s", s)
	}
	if !strings.Contains(s, "seed extra bootstrap") {
		t.Fatal("unique bootstrap fragment dropped")
	}
	if !strings.Contains(s, "placeholder file") {
		t.Fatal("verification fragment dropped")
	}
	if !strings.Contains(s, "runtime pin") {
		t.Fatal("runtime fragment dropped")
	}
}

func TestChatConductLandsInRuntimeSlot(t *testing.T) {
	loop := DefaultLoop()
	plain := AssembleIdentity(loop, nil)
	if strings.Contains(plain, "不要先写长篇评估") {
		t.Fatal("harbor identity picked up chat conduct")
	}
	s := AssembleIdentity(loop, ChatConductFragments("完善该项目", "zh-CN", false))
	if !strings.Contains(s, "不要先写长篇评估") {
		t.Fatal("chat conduct dropped")
	}
	if !strings.Contains(s, "整树重写") {
		t.Fatal("blind-rewrite pin dropped")
	}
	if !strings.Contains(s, "## runtime") {
		t.Fatalf("runtime heading missing\n%s", s)
	}
}
