// context_convo_test.go 锁住长期对话的检查点形状：摘要按先后记下人提过的每一件事，不把第一句话冻成「目标」，
// 也不把最早的几句永远钉在尾巴前面；第二次检查点接着上一份往下记；模型写的摘要要有三个小节，否则用后备的；
// 不带 Profile 的会话还是原来写代码的那一套。

package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// convoHistory 是一段主会话：人先后交代几件互不相干的事，路由每次回一段话。一轮大约一百个 token。
func convoHistory(asks ...string) []Message {
	var out []Message
	for _, ask := range asks {
		out = append(out, Message{Role: RoleUser, Content: ask})
		out = append(out, Message{Role: RoleAssistant, Content: "已经交给任务了。" + strings.Repeat("这一轮先做到这里", 8)})
	}
	return out
}

// readCheckpoint 读出会话里最后一条检查点的摘要和留下的尾巴。
func readCheckpoint(t *testing.T, store *trace.Store, spill *Spill, id string) (summary string, replay []Message) {
	t.Helper()
	evs, err := store.Read(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Type == trace.TypeCompact && ev.Payload["kind"] == "checkpoint" {
			summary, _ = ev.Payload["summary"].(string)
		}
	}
	return summary, MessagesFromEventsOpts(evs, spill)
}

// scriptedSummary 回一份固定的摘要，并记下收到的系统提示和消息。
type scriptedSummary struct {
	reply  string
	system string
	seen   []Message
	calls  int
}

// Chat 记录请求并返回 reply。
func (s *scriptedSummary) Chat(_ context.Context, req ChatRequest) (Message, error) {
	s.calls++
	if len(req.Messages) > 0 {
		s.system = req.Messages[0].Content
		s.seen = req.Messages
	}
	return Message{Role: RoleAssistant, Content: s.reply}, nil
}

// smallKeep 是只留得下最后一轮的聊天视界。
func smallKeep() artifact.LoopPreset {
	loop := ApplyChatHorizon(artifact.LoopPreset{})
	loop.KeepTokens = 150
	return loop
}

// TestConversationTailStartsAtATurn 锁住尾巴从一轮的开头起：切点落在回复上时带上这一轮的人话；
// 这一轮大得离谱时不硬带，退到原来的切点。
func TestConversationTailStartsAtATurn(t *testing.T) {
	long := strings.Repeat("这一轮先做到这里", 20) // 约 170 个 token，一条就超过 keep
	msgs := []Message{
		{Role: RoleUser, Content: "第一件事"},
		{Role: RoleAssistant, Content: "好。"},
		{Role: RoleUser, Content: "第二件事"},
		{Role: RoleAssistant, Content: long},
	}
	tail, dropped := retainTail(msgs, 150)
	if len(tail) != 2 || tail[0].Content != "第二件事" || len(dropped) != 2 {
		t.Fatalf("the tail does not start at the turn: tail %+v dropped %d", tail, len(dropped))
	}
	msgs[3].Content = strings.Repeat(long, 4)
	_, dropped = retainTail(msgs, 150)
	if len(dropped) != 3 {
		t.Fatalf("an oversized turn was pulled back whole: dropped %d", len(dropped))
	}
	if all, none := retainTail(msgs, 0); len(all) != len(msgs) || none != nil {
		t.Fatalf("keep 0 should keep everything: %d %d", len(all), len(none))
	}
}

// TestConversationCheckpointKeepsEveryAskInOrder 锁住后备摘要：七句话一句不少、按先后排，没有冻住的目标，尾巴前面不钉旧话。
func TestConversationCheckpointKeepsEveryAskInOrder(t *testing.T) {
	asks := []string{
		"你好",
		"把 #162 的 CI 失败修一下，改完先别推",
		"周报先起个草，按上周的格式",
		"以后报告结论先行",
		"明早评审的材料整理一下",
		"#162 可以推了",
		"帮我看下 hypit 为什么三天没动",
	}
	hist := convoHistory(asks...)
	store := trace.NewStore(t.TempDir())
	spill := BindSpill(t.TempDir(), t.TempDir(), "main")
	withSystem := append([]Message{{Role: RoleSystem, Content: "x"}}, hist...)
	forceCheckpoint(store, "main", withSystem, smallKeep(), spill, nil, "", 0, "runtime", nil, CompactOpts{Trigger: "budget", Profile: ConversationProfile()})
	summary, replay := readCheckpoint(t, store, spill, "main")

	if strings.Contains(summary, "## Objective") || strings.Contains(summary, "## Files") {
		t.Fatalf("a conversation got the coding summary:\n%s", summary)
	}
	// 尾巴留着最后一轮，前面六句都要在摘要里，而且按先后。
	at := -1
	for _, ask := range asks[:len(asks)-1] {
		i := strings.Index(summary, "- "+ask)
		if i < 0 {
			t.Fatalf("summary lost %q:\n%s", ask, summary)
		}
		if i < at {
			t.Fatalf("%q is out of order:\n%s", ask, summary)
		}
		at = i
	}
	if !strings.Contains(summary, convoTrailer) {
		t.Fatalf("no pointer to tickets and memory:\n%s", summary)
	}
	// 回放：一条摘要，然后是原样留下的尾巴，最早的「你好」没有被钉回来。
	if !strings.HasPrefix(replay[0].Content, checkpointLead) {
		t.Fatalf("replay head %+v", replay[0])
	}
	for _, m := range replay[1:] {
		if m.Role == RoleUser && m.Content == "你好" {
			t.Fatalf("the first thing ever said is pinned in front of the tail: %+v", replay)
		}
	}
	if last := replay[len(replay)-2]; last.Role != RoleUser || last.Content != asks[len(asks)-1] {
		t.Fatalf("the latest ask is not kept verbatim: %+v", last)
	}
	if notes := ReadNotes(spill); strings.TrimSpace(notes) != "" {
		t.Fatalf("a conversation wrote coding notes: %q", notes)
	}
}

// TestConversationCheckpointCarriesTheLedgerForward 锁住第二次检查点接着上一份记，新旧都在、先后不乱，超过上限时丢最早的。
func TestConversationCheckpointCarriesTheLedgerForward(t *testing.T) {
	store := trace.NewStore(t.TempDir())
	spill := BindSpill(t.TempDir(), t.TempDir(), "main")
	opt := CompactOpts{Trigger: "budget", Profile: ConversationProfile()}
	first := append([]Message{{Role: RoleSystem, Content: "x"}}, convoHistory("修 #162", "起草周报", "整理评审材料")...)
	hist := stripSystem(forceCheckpoint(store, "main", first, smallKeep(), spill, nil, "", 0, "runtime", nil, opt))
	hist = append(hist, convoHistory("周报第 3 节不写 Harbor", "把发布说明翻成英文", "周五的会改到下午三点")...)
	second := append([]Message{{Role: RoleSystem, Content: "x"}}, hist...)
	forceCheckpoint(store, "main", second, smallKeep(), spill, nil, "", 0, "runtime", nil, opt)
	summary, _ := readCheckpoint(t, store, spill, "main")
	want := []string{"修 #162", "起草周报", "整理评审材料", "周报第 3 节不写 Harbor", "把发布说明翻成英文"}
	at := -1
	for _, ask := range want {
		i := strings.Index(summary, "- "+ask)
		if i < 0 || i < at {
			t.Fatalf("%q missing or out of order:\n%s", ask, summary)
		}
		at = i
	}

	var many []string
	for i := 0; i < convoAsksKept+6; i++ {
		many = append(many, "第 "+string(rune('A'+i))+" 件事")
	}
	digest := convoDigest(convoHistory(many...), nil)
	if strings.Contains(digest, "第 A 件事") || !strings.Contains(digest, "第 G 件事") {
		t.Fatalf("the ledger did not drop the oldest asks:\n%s", digest)
	}
	if got := strings.Count(digest, "\n- "); got != convoAsksKept {
		t.Fatalf("kept %d asks", got)
	}
}

// TestConversationDigestSkipsWhatThePersonDidNotSay 锁住主机记录、催促和「继续」不进账本，@ 带进来的附件也不进。
func TestConversationDigestSkipsWhatThePersonDidNotSay(t *testing.T) {
	dropped := []Message{
		{Role: RoleUser, Content: HostNote([]HostLine{{ID: "a", Text: "任务 tkt-1的回报 · 做完了：改好了。"}})},
		{Role: RoleUser, Content: "继续"},
		{Role: RoleUser, Content: AttachMention("看下这个文件", "main.go 的正文……")},
		{Role: RoleAssistant, Content: "好。"},
	}
	digest := convoDigest(dropped, nil)
	if strings.Contains(digest, "改好了") || strings.Contains(digest, "继续") || strings.Contains(digest, "正文") {
		t.Fatalf("digest took lines the person did not say:\n%s", digest)
	}
	if !strings.Contains(digest, "- 看下这个文件") {
		t.Fatalf("digest lost the ask:\n%s", digest)
	}
	if convoDigest([]Message{{Role: RoleAssistant, Content: "好。"}}, nil) != "" {
		t.Fatal("an empty stretch produced a digest")
	}
}

// TestConversationCheckpointUsesItsOwnPrompt 锁住叫模型时用的是对话的提示：三个小节都在就采用，缺了就退回后备摘要。
func TestConversationCheckpointUsesItsOwnPrompt(t *testing.T) {
	hist := convoHistory("修 #162", "起草周报", "整理评审材料")
	withSystem := append([]Message{{Role: RoleSystem, Content: "x"}}, hist...)
	loop := smallKeep()
	good := &scriptedSummary{reply: convoAsksHead + "\n- 修 #162（还没推）\n\n" + convoDecidedHead + "\n- 改完先别推\n\n" + convoOpenHead + "\n无"}
	store := trace.NewStore(t.TempDir())
	spill := BindSpill(t.TempDir(), t.TempDir(), "main")
	forceCheckpoint(store, "main", withSystem, loop, spill, good, "m", 128000, "runtime", nil, CompactOpts{Profile: ConversationProfile()})
	summary, _ := readCheckpoint(t, store, spill, "main")
	if !strings.Contains(summary, "改完先别推") {
		t.Fatalf("the model's summary was not used:\n%s", summary)
	}
	if !strings.Contains(good.system, "读摘要的是之后接着和同一个人说话的助手") || strings.Contains(good.system, "coding agent") {
		t.Fatalf("wrong compaction prompt: %q", good.system)
	}
	for _, m := range good.seen {
		if m.Role == RoleUser && m.Content == "整理评审材料" {
			t.Fatalf("the summarizer was sent the tail that stays verbatim")
		}
	}

	bad := &scriptedSummary{reply: "## Objective\n修 #162"}
	store2 := trace.NewStore(t.TempDir())
	spill2 := BindSpill(t.TempDir(), t.TempDir(), "main")
	forceCheckpoint(store2, "main", withSystem, loop, spill2, bad, "m", 128000, "runtime", nil, CompactOpts{Profile: ConversationProfile()})
	summary, _ = readCheckpoint(t, store2, spill2, "main")
	if bad.calls != 2 {
		t.Fatalf("expected one retry, model was called %d times", bad.calls)
	}
	if strings.Contains(summary, "## Objective") || !strings.Contains(summary, "- 修 #162") {
		t.Fatalf("a malformed summary was used, or the fallback is missing:\n%s", summary)
	}
}

// TestCodingCheckpointIsUnchanged 锁住不带 Profile 的会话还是目标、文件、决定那一套，并照旧记工作笔记。
func TestCodingCheckpointIsUnchanged(t *testing.T) {
	hist := convoHistory("把解析错误补上返回值", "再加一个测试")
	withSystem := append([]Message{{Role: RoleSystem, Content: "x"}}, hist...)
	store := trace.NewStore(t.TempDir())
	spill := BindSpill(t.TempDir(), t.TempDir(), "code")
	forceCheckpoint(store, "code", withSystem, smallKeep(), spill, nil, "", 0, "runtime", nil, CompactOpts{Trigger: "budget"})
	summary, _ := readCheckpoint(t, store, spill, "code")
	if !strings.HasPrefix(summary, "## Objective\n把解析错误补上返回值") {
		t.Fatalf("coding summary changed:\n%s", summary)
	}
	if !strings.Contains(ReadNotes(spill), "把解析错误补上返回值") {
		t.Fatal("coding notes were not written")
	}
}
