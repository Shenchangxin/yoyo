// 锁住工单执行变量的迁移：准备命令同时拿到 YOYO_SUPERAGENT_* 和废弃的 DSH_SUPERAGENT_*，工人提示只写新名。

package superagent

import (
	"strings"
	"testing"
)

// TestWithLegacyEnvMirrorsNewNames 确认每个新名都多出一份同值的旧名，其他变量不变，已有的旧键不被覆盖。
func TestWithLegacyEnvMirrorsNewNames(t *testing.T) {
	got := withLegacyEnv(map[string]string{
		envDir:                  "/w/t1",
		envTicket:               "t1",
		"PATH":                  "/bin",
		"DSH_SUPERAGENT_TICKET": "kept",
	})
	if got[envDir] != "/w/t1" || got["DSH_SUPERAGENT_DIR"] != "/w/t1" {
		t.Fatalf("dir not mirrored: %v", got)
	}
	if got["DSH_SUPERAGENT_TICKET"] != "kept" {
		t.Fatalf("existing legacy key overwritten: %v", got)
	}
	if got["PATH"] != "/bin" || got["DSH_PATH"] != "" {
		t.Fatalf("unrelated keys changed: %v", got)
	}
	if len(withLegacyEnv(nil)) != 0 {
		t.Fatal("nil env must give an empty map")
	}
}

// TestWorkerPromptUsesNewEnvNames 确认工人提示写出非空的新名变量，不写旧名，也不写空值。
func TestWorkerPromptUsesNewEnvNames(t *testing.T) {
	e := &Executor{Env: func(dir, ticketID string) map[string]string {
		return map[string]string{envDir: dir, envTicket: ticketID, envGitHubRepo: ""}
	}}
	prompt := workerPrompt(e, "/w/t1", "t1")
	if !strings.Contains(prompt, envDir+"=/w/t1") || !strings.Contains(prompt, envTicket+"=t1") {
		t.Fatalf("prompt missing new env lines: %q", prompt)
	}
	if strings.Contains(prompt, legacyEnvPrefix) {
		t.Fatalf("prompt must not repeat legacy names: %q", prompt)
	}
	if strings.Contains(prompt, envGitHubRepo) {
		t.Fatalf("empty values must not be written: %q", prompt)
	}
}
