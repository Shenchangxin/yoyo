// 锁住 DefaultLoop 的评测数字不被 ApplyChatHorizon 改写，偏小的聊天视界被抬高，RulesTokens 被写成 1600。
// 同一文件还锁住行为文案不改 Harbor 循环，以及身份、语言和环境提示的几条拼装结果。childLoopFor 的实现在 tools_task.go。

package runtime

import (
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestDefaultLoopHorizonUnchanged 确认 DefaultLoop 的 MaxTurns 是 32，MaxToolMessages 是 40，MicroKeep 是 4，CompactionKeep 是 24，CompactionTokens 是 24000。
func TestDefaultLoopHorizonUnchanged(t *testing.T) {
	loop := DefaultLoop()
	if loop.MaxTurns != 32 || loop.MaxToolMessages != 40 {
		t.Fatalf("harbor loop drifted: %+v", loop)
	}
	if loop.MicroKeep != 4 {
		t.Fatalf("harbor microkeep %d", loop.MicroKeep)
	}
	if loop.CompactionKeep != 24 {
		t.Fatalf("harbor compaction keep %d", loop.CompactionKeep)
	}
	if loop.CompactionTokens != 24_000 {
		t.Fatalf("compaction %d", loop.CompactionTokens)
	}
}

// TestApplyChatHorizon 确认偏小的 MaxTurns 等被抬高，RulesTokens 写成 ChatRulesTokens 并盖掉传入的 32k，StackAgentsMD 变成 false，AllowLLMCompact 变成 true；已经更大的轮数和保留数不被缩小，DefaultLoop 本身不变且 AllowLLMCompact 仍是 false。
func TestApplyChatHorizon(t *testing.T) {
	fat := DefaultLoop()
	fat.RulesTokens = 32_000
	fat.StackAgentsMD = true
	got := ApplyChatHorizon(fat)
	if got.MaxTurns < 64 || got.MaxToolMessages < 160 || got.MicroKeep < 32 || got.CompactionKeep < 96 {
		t.Fatalf("%+v", got)
	}
	if got.RulesTokens != ChatRulesTokens || got.KeepTokens < 16000 || got.TaskMaxTurns < 40 {
		t.Fatalf("chat context floors %+v", got)
	}
	if got.StackAgentsMD {
		t.Fatal("nested AGENTS.md must load on read, not stack into the prefix")
	}
	if !got.AllowLLMCompact {
		t.Fatal("desktop checkpoints must ask the model for a structured summary")
	}
	if DefaultLoop().AllowLLMCompact {
		t.Fatal("harbor loop must stay deterministic")
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
	wide.MicroKeep = 48
	wide.CompactionKeep = 120
	got = ApplyChatHorizon(wide)
	if got.MaxToolMessages != 200 || got.MicroKeep != 48 || got.CompactionKeep != 120 {
		t.Fatalf("must not shrink large CAS: %+v", got)
	}
	if DefaultLoop().MaxTurns != 32 {
		t.Fatal("ApplyChatHorizon mutated DefaultLoop")
	}
	if DefaultLoop().MaxToolMessages != 40 {
		t.Fatal("ApplyChatHorizon mutated Harbor tool cap")
	}
	if DefaultLoop().MicroKeep != 4 || DefaultLoop().CompactionKeep != 24 {
		t.Fatal("ApplyChatHorizon mutated Harbor keep")
	}
}

// TestChatConductDoesNotTouchHarborLoop 确认聊天行为文案不改 DefaultLoop 的 Execution、Bootstrap 和 MaxTurns 32。
func TestChatConductDoesNotTouchHarborLoop(t *testing.T) {
	base := DefaultLoop()
	frags := ChatConductFragments(false)
	joined := fragmentText(frags)
	if !strings.Contains(joined, "same language") {
		t.Fatalf("%+v", frags)
	}
	if !strings.Contains(joined, "You decide the deliverable") {
		t.Fatal("missing deliverable judgment")
	}
	if strings.Contains(joined, "The latest operator utterance is already the task: start writing workspace artifacts this turn") {
		t.Fatal("must not unconditionally force writes")
	}
	if !strings.Contains(joined, "plan step") || !strings.Contains(joined, "rewrite the tree") {
		t.Fatal("missing plan-language and no-blind-rewrite")
	}
	if !strings.Contains(joined, "ask_user") {
		t.Fatal("ambiguous plan vs implement must go through the agent")
	}
	if !strings.Contains(joined, "observation that changed") {
		t.Fatal("missing observation-vs-instrument law")
	}
	method := fragmentText(ChatConduct(ConductOpts{Methodology: true}))
	if !strings.Contains(method, "observation that changed") {
		t.Fatal("methodology conduct dropped observation law")
	}
	got := DefaultLoop()
	if got.Execution != base.Execution || got.Bootstrap != base.Bootstrap || got.MaxTurns != 32 {
		t.Fatalf("harbor loop mutated: %+v", got)
	}
	plan := ChatConductFragments(true)
	if strings.Contains(fragmentText(plan), "start writing those workspace artifacts") {
		t.Fatal("plan mode must not force writes")
	}
	if !strings.Contains(plan[0].Text, "same language") {
		t.Fatal("plan mode must still match language")
	}
}

// fragmentText 把片段正文按行拼成一段，供上面的文案断言使用。
func fragmentText(frags []artifact.PromptFragment) string {
	var b strings.Builder
	for _, f := range frags {
		b.WriteString(f.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

// TestChildLoopProfiles 确认 explore 和 qa 不进计划模式，explore 的 MaxTurns 不小于 24，且 childLoopFor 不改 DefaultLoop 的 32 轮。
func TestChildLoopProfiles(t *testing.T) {
	parent := RunRequest{Loop: DefaultLoop(), SoftHorizon: true}
	explore := childLoopFor(parent, "explore")
	if explore.PlanMode {
		t.Fatal("explore must return findings, not enter plan mode")
	}
	if explore.MaxTurns < 24 || !strings.Contains(explore.TaskInstruction, "explore") {
		t.Fatalf("explore %+v", explore)
	}
	qa := childLoopFor(parent, "qa")
	if qa.PlanMode {
		t.Fatal("qa evaluates; it is not plan mode")
	}
	if !strings.Contains(qa.TaskInstruction, "evaluator") {
		t.Fatalf("qa %+v", qa)
	}
	impl := childLoopFor(parent, "implement")
	if impl.PlanMode {
		t.Fatal("implement must not force plan mode")
	}
	if DefaultLoop().MaxTurns != 32 || DefaultLoop().PlanMode {
		t.Fatal("childLoopFor mutated DefaultLoop")
	}
}

// TestOperatorVoiceSkipsControl 确认控制行 toolBudgetNudge 不盖住操作者原文「完善前端」。
func TestOperatorVoiceSkipsControl(t *testing.T) {
	got := OperatorVoice(toolBudgetNudge, []Message{
		{Role: RoleUser, Content: "完善前端"},
		{Role: RoleUser, Content: toolBudgetNudge},
	})
	if got != "完善前端" {
		t.Fatalf("%q", got)
	}
}

// TestOperatorVoiceSkipsResume 确认「继续」退回更早那句含 RBAC 的操作者原文。
func TestOperatorVoiceSkipsResume(t *testing.T) {
	got := OperatorVoice("继续", []Message{
		{Role: RoleUser, Content: "本项目是一个基于golang开发的RBAC权限管理系统，请完善前端"},
		{Role: RoleUser, Content: "继续"},
	})
	if !strings.Contains(got, "RBAC") {
		t.Fatalf("%q", got)
	}
}

// TestAssembleIdentitySingleBootstrap 确认身份里 ## bootstrap 只出现一次，重复的 Bootstrap 正文只留一份，独特片段仍在。
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

// TestChatDoesNotPinALanguage 确认动态段不含 ## Voice，也不含「用中文回复」。
func TestChatDoesNotPinALanguage(t *testing.T) {
	dyn := AssembleDynamic("", nil, "## Objective\n完善前端\n", "", "")
	if strings.Contains(dyn, "## Voice") || strings.Contains(dyn, "用中文回复") {
		t.Fatalf("language must follow the user message, not a voice pin: %s", dyn)
	}
}

// TestChatConductLandsInRuntimeSlot 确认没有聊天片段时身份不含交付判断，加上片段后落在 ## runtime，并带上 chatBootstrap。
func TestChatConductLandsInRuntimeSlot(t *testing.T) {
	loop := DefaultLoop()
	plain := AssembleIdentity(loop, nil)
	if strings.Contains(plain, "You decide the deliverable") {
		t.Fatal("harbor identity picked up chat conduct")
	}
	s := AssembleIdentity(loop, ChatConductFragments(false))
	if !strings.Contains(s, "You decide the deliverable") {
		t.Fatal("chat conduct dropped")
	}
	if !strings.Contains(s, "rewrite the tree") {
		t.Fatal("blind-rewrite pin dropped")
	}
	if !strings.Contains(s, "## runtime") {
		t.Fatalf("runtime heading missing\n%s", s)
	}
	if !strings.Contains(s, chatBootstrap) {
		t.Fatal("chat bootstrap dropped")
	}
}

// TestLanguagePinLeadsIdentity 确认中文查询的语言钉在 You are Yoyo 之前，并且覆盖进度叙述。
func TestLanguagePinLeadsIdentity(t *testing.T) {
	pin := LanguagePin("帮我查询一下关于agent自进化的论文")
	s := AssembleIdentity(DefaultLoop(), []artifact.PromptFragment{pin})
	yi := strings.Index(s, "You are Yoyo")
	ci := strings.Index(s, "Chinese")
	if ci < 0 || yi < 0 || ci > yi {
		t.Fatalf("pin must lead identity\n%s", s)
	}
	if !strings.Contains(pin.Text, "progress") {
		t.Fatal("pin must cover progress narration, not only the final letter")
	}
}

// TestRefreshDynamicRepeatsLanguagePin 确认 SoftHorizon 的工作记忆再次钉上中文输出。
func TestRefreshDynamicRepeatsLanguagePin(t *testing.T) {
	req := RunRequest{User: "帮我查询一下关于agent自进化的论文", SoftHorizon: true}
	k := newContextKernel(&req)
	_ = k.refreshDynamic(nil)
	if !strings.Contains(k.dyn, "Output language") || !strings.Contains(k.dyn, "Chinese") {
		t.Fatalf("working memory must re-pin Chinese output:\n%s", k.dyn)
	}
}

// TestAssembleEnvironmentIsFacts 确认环境段以 os: 开头，含工作区和 shell，不含 Start-Process 和「用中文」。
func TestAssembleEnvironmentIsFacts(t *testing.T) {
	ws := t.TempDir()
	got := AssembleEnvironment(ws)
	if !strings.HasPrefix(got, "os: ") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, "workspace: "+ws) {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, "shell:") {
		t.Fatalf("%s", got)
	}
	if strings.Contains(got, "Start-Process") || strings.Contains(got, "用中文") {
		t.Fatalf("instructions leaked into env: %s", got)
	}
	if goruntime.GOOS == "windows" {
		if !strings.Contains(got, "pwsh") && !strings.Contains(got, "powershell") {
			t.Fatalf("windows shell fact: %s", got)
		}
	}
}
