package superagent

import (
	"strings"
	"testing"
)

// TestWorkerPromptCarriesTheOperatorProfile 锁住工人能读到操作者留下的偏好，没有画像时不写这一段。
func TestWorkerPromptCarriesTheOperatorProfile(t *testing.T) {
	e := &Executor{Profile: func() string { return "Preferences:\n- 报告用中文\n" }}
	prompt := workerPrompt(e, "C:/work/tickets/tkt-1", "tkt-1")
	if !strings.Contains(prompt, "报告用中文") || !strings.Contains(prompt, "以说明为准") {
		t.Fatalf("prompt = %q", prompt)
	}
	bare := workerPrompt(&Executor{Profile: func() string { return "  " }}, "", "tkt-2")
	if strings.Contains(bare, "操作者已经留下") {
		t.Fatalf("empty profile wrote a section: %q", bare)
	}
}
