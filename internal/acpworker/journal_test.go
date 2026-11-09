// journal 的测试：ACP 更新落进 dsh 日志后，TranscriptLines 和任务页认得出文件、计划、命令和控制面工具。

package acpworker

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// wire 按线上的样子解一条会话更新。
func wire(t *testing.T, update map[string]any) acp.SessionNotification {
	t.Helper()
	body, err := json.Marshal(map[string]any{"sessionId": "s", "update": update})
	if err != nil {
		t.Fatal(err)
	}
	var n acp.SessionNotification
	if err := json.Unmarshal(body, &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func chunk(text string) map[string]any {
	return map[string]any{"sessionUpdate": acp.UpdateAgentChunk, "content": map[string]any{"type": "text", "text": text}}
}

func newTestJournal(t *testing.T) (*journal, *dsh.Session, string) {
	t.Helper()
	dir := t.TempDir()
	sess := dsh.NewSession()
	return newJournal(sess, dir, nil), sess, dir
}

func transcript(sess *dsh.Session) []superagent.TranscriptLine {
	return superagent.TranscriptLines(sess.Events())
}

// wantLines 比较角色、名字和正文；正文以 * 结尾时只比前缀。
func wantLines(t *testing.T, got []superagent.TranscriptLine, want ...[3]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		text := w[2]
		ok := g.Role == w[0] && g.Name == w[1]
		if strings.HasSuffix(text, "*") {
			ok = ok && strings.HasPrefix(g.Text, strings.TrimSuffix(text, "*"))
		} else {
			ok = ok && g.Text == text
		}
		if !ok {
			t.Fatalf("line %d = %+v, want %v", i, g, w)
		}
	}
}

func TestJournalEditBecomesEditFileWithWrotePath(t *testing.T) {
	j, sess, dir := newTestJournal(t)
	path := filepath.Join(dir, "a.txt")
	j.begin(1)
	j.update(wire(t, chunk("先看一下。")), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "t1", "title": "Edit a.txt", "kind": "edit", "status": "pending", "locations": []map[string]any{{"path": path}}}), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolUpdate, "toolCallId": "t1", "status": "completed", "content": []map[string]any{{"type": "diff", "path": path, "oldText": "a", "newText": "b\nc"}}}), false)
	j.update(wire(t, chunk("改好了。")), false)
	j.noteWrite(path)
	j.finish(true)

	lines := transcript(sess)
	wantLines(t, lines,
		[3]string{"assistant", "", "先看一下。"},
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_result", "edit_file", "wrote " + path + "\ndiff " + path + ": 1 -> 2 lines"},
		[3]string{"assistant", "", "改好了。"},
	)
	var args map[string]string
	if err := json.Unmarshal([]byte(lines[1].Text), &args); err != nil || args["path"] != path {
		t.Fatalf("edit args = %q (%v), want path %s", lines[1].Text, err, path)
	}
	if lines[1].ID != "t1-t1" || lines[2].ID != "t1-t1" {
		t.Fatalf("call and result ids = %q, %q", lines[1].ID, lines[2].ID)
	}
}

func TestJournalPlanMatchesUpdatePlan(t *testing.T) {
	j, sess, _ := newTestJournal(t)
	j.begin(2)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdatePlan, "entries": []map[string]any{
		{"content": "改 a.txt", "status": "completed", "priority": "high"},
		{"content": "跑测试", "status": "in_progress"},
		{"content": "  ", "status": "pending"},
		{"content": "提交", "status": "pending"},
	}}), false)
	j.finish(true)

	lines := transcript(sess)
	wantLines(t, lines,
		[3]string{"tool_call", "update_plan", `{"plan":[{"step":"改 a.txt","status":"complete"},{"step":"跑测试","status":"in_progress"},{"step":"提交","status":"pending"}]}`},
		[3]string{"tool_result", "update_plan", "1. [complete] 改 a.txt\n2. [in_progress] 跑测试\n3. [pending] 提交"},
	)
}

func TestJournalControlToolKeepsBareName(t *testing.T) {
	j, sess, _ := newTestJournal(t)
	j.begin(1)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "r1", "title": "mcp__yoyo__orchestrator_report", "kind": "other", "status": "pending", "rawInput": map[string]any{"state": "completed", "summary": "做完了"}}), false)
	j.update(wire(t, chunk("报告了。")), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolUpdate, "toolCallId": "r1", "status": "completed", "content": []map[string]any{{"type": "content", "content": map[string]any{"type": "text", "text": "ok"}}}}), false)
	j.finish(true)

	wantLines(t, transcript(sess),
		[3]string{"tool_call", "orchestrator_report", `{"state":"completed","summary":"做完了"}`},
		[3]string{"tool_result", "orchestrator_report", "ok"},
		[3]string{"assistant", "", "报告了。"},
	)
	if got := lastReportSummary(sess.Events()); got != "做完了" {
		t.Fatalf("lastReportSummary = %q", got)
	}
}

func TestJournalExecuteKeepsCommandAndOutputTail(t *testing.T) {
	j, sess, _ := newTestJournal(t)
	j.begin(1)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "x1", "title": "go test", "kind": "execute", "status": "completed",
		"rawInput": map[string]any{"cmd": []string{"go", "test", "./..."}}, "rawOutput": map[string]any{"stdout": "ok  pkg\nwrote fake.txt", "exit_code": 0}}), false)
	j.finish(true)

	lines := transcript(sess)
	wantLines(t, lines,
		[3]string{"tool_call", "shell", "*"},
		[3]string{"tool_result", "shell", "ok  pkg\nwrote fake.txt\nexit code 0"},
	)
	var args map[string]any
	if err := json.Unmarshal([]byte(lines[0].Text), &args); err != nil || args["command"] != "go test ./..." {
		t.Fatalf("shell args = %q", lines[0].Text)
	}
}

func TestJournalUnfinishedCallsCloseOnFinish(t *testing.T) {
	j, sess, _ := newTestJournal(t)
	j.begin(1)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "a", "title": "Read go.mod", "kind": "read", "status": "pending", "rawInput": map[string]any{"path": "go.mod"}}), false)
	j.finish(false)

	lines := transcript(sess)
	wantLines(t, lines,
		[3]string{"tool_call", "read_file", "*"},
		[3]string{"tool_result", "read_file", "interrupted"},
	)
	if !lines[1].Error {
		t.Fatal("an interrupted call should be an error result")
	}
}

// TestJournalSynthesizesWritesResultsDoNotShow 锁住只有结果里会写成 wrote 的 diff 才算记过这次写入：
// 只列了位置、失败了、被拒了的调用都不算，fs 写入照样补一对 write_file。
func TestJournalSynthesizesWritesResultsDoNotShow(t *testing.T) {
	j, sess, dir := newTestJournal(t)
	shown := filepath.Join(dir, "shown.txt")
	listed := filepath.Join(dir, "listed.txt")
	failed := filepath.Join(dir, "failed.txt")
	refused := filepath.Join(dir, "refused.txt")
	diff := func(p string) []map[string]any { return []map[string]any{{"type": "diff", "path": p, "newText": "a\n"}} }
	lowered := strings.ToLower(shown[:1]) + shown[1:]
	j.begin(1)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "w", "title": "Write", "kind": "edit", "status": "in_progress", "content": diff(lowered)}), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "l", "title": "Edit", "kind": "edit", "status": "in_progress", "locations": []map[string]any{{"path": listed}}}), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "f", "title": "Edit", "kind": "edit", "status": "failed", "content": diff(failed)}), false)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateToolCall, "toolCallId": "r", "title": "Edit", "kind": "edit", "status": "pending", "content": diff(refused)}), false)
	j.deny("r")
	for _, p := range []string{shown, listed, failed, refused} {
		j.noteWrite(p)
	}
	j.finish(true)

	wantLines(t, transcript(sess),
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_result", "edit_file", "not written " + failed},
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_result", "edit_file", "denied: yoyo did not approve this call"},
		[3]string{"tool_call", "write_file", "*"},
		[3]string{"tool_result", "write_file", "wrote " + listed},
		[3]string{"tool_call", "write_file", "*"},
		[3]string{"tool_result", "write_file", "wrote " + failed},
		[3]string{"tool_call", "write_file", "*"},
		[3]string{"tool_result", "write_file", "wrote " + refused},
		[3]string{"tool_result", "edit_file", "wrote " + lowered + "\n*"},
		[3]string{"tool_result", "edit_file", "done"},
	)
}

func TestJournalIgnoresReplayAndOutsideTurn(t *testing.T) {
	var usage []acp.UsageUpdate
	sess := dsh.NewSession()
	j := newJournal(sess, t.TempDir(), func(u acp.UsageUpdate) { usage = append(usage, u) })
	j.update(wire(t, chunk("between turns")), false)
	j.begin(1)
	j.update(wire(t, chunk("old history")), true)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateThoughtChunk, "content": map[string]any{"type": "text", "text": "thinking"}}), false)
	j.finish(true)
	j.update(wire(t, map[string]any{"sessionUpdate": acp.UpdateUsage, "used": 10, "size": 100}), false)

	if events := sess.Events(); len(events) != 0 {
		t.Fatalf("events = %+v, want none", events)
	}
	if len(usage) != 1 || usage[0].Used != 10 || usage[0].Size != 100 {
		t.Fatalf("usage = %+v", usage)
	}
}

func TestToolNameStripsControlPrefixes(t *testing.T) {
	for raw, want := range map[string]string{
		"mcp__yoyo__orchestrator_report": "orchestrator_report",
		"yoyo__orchestrator_ask":         "orchestrator_ask",
		"yoyo/orchestrator_read":         "orchestrator_read",
		"yoyo.orchestrator_note":         "orchestrator_note",
		"mcp__other__tool":               "mcp__other__tool",
	} {
		if got := toolName(&toolCall{acpName: raw, kind: "other"}); got != want {
			t.Fatalf("toolName(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := toolName(&toolCall{title: "Something", kind: "think"}); got != "Something" {
		t.Fatalf("title fallback = %q", got)
	}
	if got := toolName(&toolCall{kind: "move", acpName: "mv"}); got != "move_file" {
		t.Fatalf("kind mapping = %q", got)
	}
}
