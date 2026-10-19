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

func TestEvolveOverlayCLIRejectsPromote(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay", "--session", "s1", "--promote"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "promote") {
		t.Fatalf("%v", err)
	}
}

func TestEvolveOverlayCLIRejectsSuiteK(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay", "--session", "s1", "--k", "3"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "k") {
		t.Fatalf("%v", err)
	}
}

func TestEvolveOverlayCLIRequiresSession(t *testing.T) {
	cmd := evolveCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--overlay"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--session") {
		t.Fatalf("%v", err)
	}
}

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
