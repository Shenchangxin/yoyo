// 本文件锁住 evolve --overlay 会拒绝 --promote 和 --k，必须带 --session，且不改全局 ref。由 go test 调用。

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// TestEvolveOverlayCLIRejectsPromote 锁住 --overlay 配 --promote 会失败，且错误文案含 promote。
func TestEvolveOverlayCLIRejectsPromote(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay", "--session", "s1", "--promote"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "promote") {
		t.Fatalf("%v", err)
	}
}

// TestEvolveOverlayCLIRejectsSuiteK 锁住 --overlay 配 --k 会失败，且错误文案含 k。
func TestEvolveOverlayCLIRejectsSuiteK(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay", "--session", "s1", "--k", "3"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "k") {
		t.Fatalf("%v", err)
	}
}

// TestEvolveOverlayCLIRequiresSession 锁住只有 --overlay、没有 --session 时失败，错误文案含 --session。
func TestEvolveOverlayCLIRequiresSession(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--session") {
		t.Fatalf("%v", err)
	}
}

// TestEvolveOverlayCLIKeepBase 锁住启发式 overlay 输出 kept_base=true、标准错误含 Active unchanged，并且策略仍是 follow_active，active、canary、staging 都不变。
func TestEvolveOverlayCLIKeepBase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("YOYO_HOME", home)
	a, err := app.Open(home, evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	active := a.ActiveHash()
	canary := a.Refs.GetOrEmpty(artifact.RefCanary)
	staging := a.Refs.GetOrEmpty(artifact.RefStaging)
	id := sess.ID
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := evolveCmd()
	cmd.SilenceUsage = true
	var out, errb strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{
		"--overlay", "--session", id, "--solver", "heuristic",
		"--reuse-turn=false", "--task", "Write a file named hello.txt with the word hello",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "kept_base=true") {
		t.Fatalf("stdout %s", out.String())
	}
	if !strings.Contains(errb.String(), "Active unchanged") {
		t.Fatalf("stderr %s", errb.String())
	}

	a, err = app.Open(home, evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got, err := a.GetSession(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.HarnessPolicy != session.FollowActive {
		t.Fatalf("policy %s", got.HarnessPolicy)
	}
	if a.ActiveHash() != active || a.Refs.GetOrEmpty(artifact.RefCanary) != canary || a.Refs.GetOrEmpty(artifact.RefStaging) != staging {
		t.Fatal("cli overlay moved global refs")
	}
}

// evalsDir 返回当前工作目录上两级的 evals 绝对路径。取工作目录或 Abs 失败时让测试失败。
func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
