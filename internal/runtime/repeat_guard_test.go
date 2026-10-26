// 锁住规范化参数忽略键顺序，以及重复链跳过读取、到点提醒、不结束回合。

package runtime

import (
	"strings"
	"testing"
)

// TestCanonicalArgsIgnoresKeyOrder 确认键顺序不同的同一份 JSON 得到同一个规范化参数。
func TestCanonicalArgsIgnoresKeyOrder(t *testing.T) {
	a := canonicalArgs(`{"b":1,"a":{"d":true,"c":2}}`)
	b := canonicalArgs("{\"a\":{\"c\":2,\"d\":true},\"b\":1}")
	if a != b {
		t.Fatalf("%s vs %s", a, b)
	}
}

// TestRepeatChainSkipsReadsAndDoesNotStop 确认 read_file 和 update_plan 不打断 shell 链：第 3 次是短提醒，第 5 和 8 次是长提醒，第 9 次保持安静；换参数或 reset 后计数回到 1 且没有提醒。
func TestRepeatChainSkipsReadsAndDoesNotStop(t *testing.T) {
	var r repeatTracker
	req := RunRequest{User: "继续修", SoftHorizon: true}
	shell := ToolCall{Name: "shell", Arguments: `{"command":"node qa/check.cjs"}`}
	swapped := ToolCall{Name: "shell", Arguments: `{"cmd":"node qa/check.cjs"}`}
	if canonicalArgs(shell.Arguments) != canonicalArgs(`{"command":"node qa/check.cjs"}`) {
		t.Fatal("canon")
	}
	read := ToolCall{Name: "read_file", Arguments: `{"path":"game.js"}`}
	plan := ToolCall{Name: "update_plan", Arguments: `{"steps":[]}`}
	var last string
	for i := 1; i <= 9; i++ {
		got := r.observe(req, []ToolCall{read, shell, plan})
		switch i {
		case 3:
			if !strings.Contains(got, "不要再发这一次调用") {
				t.Fatalf("gentle %q", got)
			}
			if !isControlUser(got) {
				t.Fatal("gentle reminder must be control")
			}
		case 5, 8:
			if !strings.Contains(got, "检测到重复的工具调用") || !strings.Contains(got, "shell") {
				t.Fatalf("detail %d %q", i, got)
			}
			if !isControlUser(got) {
				t.Fatal("detail reminder must be control")
			}
		default:
			if got != "" {
				t.Fatalf("hit %d reminded %q", i, got)
			}
		}
		last = got
	}
	if last != "" {
		t.Fatal("past the last threshold must stay quiet")
	}
	if r.count != 9 {
		t.Fatalf("count %d", r.count)
	}
	// 参数形状一变就是新链，不会变成第十次提醒。
	if got := r.observe(req, []ToolCall{swapped}); got != "" || r.count != 1 {
		t.Fatalf("new args count=%d nudge=%q", r.count, got)
	}
	r.reset()
	if r.count != 0 || r.observe(req, []ToolCall{shell}) != "" {
		t.Fatal("a user steer clears the chain")
	}
}

// TestRepeatChainResetsOnDifferentTrackedCall 确认换成 str_replace 后链从 1 重新计，再回到原来的 shell 也是新链。
func TestRepeatChainResetsOnDifferentTrackedCall(t *testing.T) {
	var r repeatTracker
	req := RunRequest{User: "继续", SoftHorizon: true}
	shell := ToolCall{Name: "shell", Arguments: `{"command":"go test"}`}
	r.observe(req, []ToolCall{shell})
	r.observe(req, []ToolCall{shell})
	r.observe(req, []ToolCall{{Name: "str_replace", Arguments: `{"path":"game.js","old":"a","new":"b"}`}})
	if r.count != 1 || r.name != "str_replace" {
		t.Fatalf("edit must reset the chain: %+v", r)
	}
	if got := r.observe(req, []ToolCall{shell}); r.count != 1 || got != "" {
		t.Fatalf("shell after an edit starts over: count=%d %q", r.count, got)
	}
}
