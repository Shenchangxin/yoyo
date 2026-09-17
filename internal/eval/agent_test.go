package eval

import "testing"

func TestAgentArgsDefaultWorkspace(t *testing.T) {
	got := AgentArgs("", "Write hello.txt")
	if len(got) != 5 || got[0] != "yoyo" || got[2] != "--workspace" || got[3] != "/app" {
		t.Fatalf("%v", got)
	}
	if got[4] != "Write hello.txt" {
		t.Fatalf("%v", got)
	}
}
