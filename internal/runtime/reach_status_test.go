// reach_status 在没有 CLI 时停在说明。doctor 的 JSON 按键排序。正文不推荐 curl。

package runtime

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestReachStatusWithoutCLI 确认 PATH 上没有 agent-reach 时不安装，并说明用宿主工具搜索。
func TestReachStatusWithoutCLI(t *testing.T) {
	prev := lookAgentReach
	lookAgentReach = func() string { return "" }
	t.Cleanup(func() { lookAgentReach = prev })
	res := (&WorkspaceTools{}).reachStatus()
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "not on PATH") || !strings.Contains(res.Content, "web_search") {
		t.Fatalf("%s", res.Content)
	}
}

// TestReachStatusDoctorJSON 确认 doctor 的 JSON 被按键展开，SearchAPI 的状态写进正文。
func TestReachStatusDoctorJSON(t *testing.T) {
	prevLook, prevRun := lookAgentReach, runAgentReachDoctor
	lookAgentReach = func() string { return filepath.Join("bin", "agent-reach") }
	runAgentReachDoctor = func(context.Context, string) (string, error) {
		return `{"web":{"status":"ok","active_backend":"Jina Reader","message":"ready"},"exa_search":{"status":"warn","message":"unverified"}}`, nil
	}
	t.Cleanup(func() {
		lookAgentReach, runAgentReachDoctor = prevLook, prevRun
	})
	res := (&WorkspaceTools{SearchAPI: func(string) (string, error) { return "x", nil }}).reachStatus()
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "SearchAPI: hooked") || !strings.Contains(res.Content, "web:") || !strings.Contains(res.Content, "exa_search") {
		t.Fatalf("%s", res.Content)
	}
}

// TestReachStatusNeverMentionsCurl 确认正文不含 curl。
func TestReachStatusNeverMentionsCurl(t *testing.T) {
	prev := lookAgentReach
	lookAgentReach = func() string { return "" }
	t.Cleanup(func() { lookAgentReach = prev })
	res := (&WorkspaceTools{}).reachStatus()
	if strings.Contains(strings.ToLower(res.Content), "curl ") {
		t.Fatalf("must not recommend curl: %s", res.Content)
	}
}
