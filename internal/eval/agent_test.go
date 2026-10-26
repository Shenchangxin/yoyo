// 锁住空工作区参数会落成 /app，并且参数顺序是 yoyo、run、--workspace、工作区、说明。

package eval

import "testing"

// TestAgentArgsDefaultWorkspace 要求 workspace 为空时第三段之后是 /app，最后一段是说明原文。
func TestAgentArgsDefaultWorkspace(t *testing.T) {
	got := AgentArgs("", "Write hello.txt")
	if len(got) != 5 || got[0] != "yoyo" || got[2] != "--workspace" || got[3] != "/app" {
		t.Fatalf("%v", got)
	}
	if got[4] != "Write hello.txt" {
		t.Fatalf("%v", got)
	}
}
