// host_note_test.go 锁住常驻代理怎么接住主机代写的行：两轮之间新来的并成一条主机记录追加在日志末尾，
// 同一行只给模型看一次，前面的消息一个字节都不动；检查点重灌之后不重复追加。

package dshdesk

import (
	"context"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// countNotes 数出 msgs 里有几条主机记录，以及其中包含 needle 的条数。
func countNotes(msgs []runtime.Message, needle string) (notes, hits int) {
	for _, m := range msgs {
		if m.Role != runtime.RoleUser || !runtime.IsHostNote(m.Content) {
			continue
		}
		notes++
		if strings.Contains(m.Content, needle) {
			hits++
		}
	}
	return notes, hits
}

// TestLiveAgentSeesHostLinesOnce 锁住同一个代理的第二轮能看到两轮之间的工人回报，第三轮不再重复，新来的另起一条。
func TestLiveAgentSeesHostLinesOnce(t *testing.T) {
	host := New()
	client := &seeingClient{reply: "好的。"}
	run := func(user string, lines []runtime.HostLine) []runtime.Message {
		t.Helper()
		if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: user, Client: client, HostLines: lines}); err != nil {
			t.Fatal(err)
		}
		return client.seen[len(client.seen)-1]
	}
	first := run("把 CI 修一下", nil)
	report := runtime.HostLine{ID: "handoff:tkt-1:a", Text: "任务 tkt-1的回报 · 做完了：改成按本地时区取日期。"}
	second := run("它改了哪里？", []runtime.HostLine{report})
	if notes, hits := countNotes(second, "改成按本地时区取日期"); notes != 1 || hits != 1 {
		t.Fatalf("second turn: notes %d hits %d\n%+v", notes, hits, second)
	}
	// 追加只发生在末尾：第一轮发出去的每条消息，第二轮原样还在同一位置。
	for i, m := range first {
		if second[i].Role != m.Role || second[i].Content != m.Content {
			t.Fatalf("prefix changed at %d: %+v -> %+v", i, m, second[i])
		}
	}
	last := second[len(second)-1]
	if last.Role != runtime.RoleUser || !strings.Contains(last.Content, "它改了哪里？") {
		t.Fatalf("the person's words are not last: %+v", last)
	}
	if note := second[len(second)-2]; !runtime.IsHostNote(note.Content) {
		t.Fatalf("the note is not right before the person's words: %+v", note)
	}

	notice := runtime.HostLine{ID: "notice:s:1", Text: "通知：更新已下载。"}
	third := run("还有别的吗", []runtime.HostLine{report, notice})
	if notes, hits := countNotes(third, "改成按本地时区取日期"); notes != 2 || hits != 1 {
		t.Fatalf("third turn repeated the report: notes %d hits %d", notes, hits)
	}
	if _, hits := countNotes(third, "更新已下载"); hits != 1 {
		t.Fatalf("third turn missed the notice: %+v", third)
	}
}

// TestReseedDoesNotRepeatHostLines 锁住检查点重灌之后，历史里已经带着的行不再追加一遍。
func TestReseedDoesNotRepeatHostLines(t *testing.T) {
	host := New()
	client := &seeingClient{reply: "好的。"}
	line := runtime.HostLine{ID: "handoff:tkt-1:a", Text: "任务 tkt-1的回报 · 做完了：改好了。"}
	if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: "第一句", Client: client}); err != nil {
		t.Fatal(err)
	}
	history := []runtime.Message{
		{Role: runtime.RoleUser, Content: "Context checkpoint (untrusted working memory; pins were reassembled separately):\n要点"},
		{Role: runtime.RoleUser, Content: runtime.HostNote([]runtime.HostLine{line})},
	}
	if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: "第二句", Client: client, Checkpoint: "1:x", History: history, HostLines: []runtime.HostLine{line}}); err != nil {
		t.Fatal(err)
	}
	if notes, hits := countNotes(client.seen[len(client.seen)-1], "改好了"); notes != 1 || hits != 1 {
		t.Fatalf("after reseed: notes %d hits %d", notes, hits)
	}
	if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: "第三句", Client: client, Checkpoint: "1:x", History: history, HostLines: []runtime.HostLine{line}}); err != nil {
		t.Fatal(err)
	}
	if notes, hits := countNotes(client.seen[len(client.seen)-1], "改好了"); notes != 1 || hits != 1 {
		t.Fatalf("next turn after reseed: notes %d hits %d", notes, hits)
	}
}
