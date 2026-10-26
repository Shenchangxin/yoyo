// remotion_test.go 检查 Remotion 状态、工程脚手架、缺少 Node 时的安装，以及预设风格。

package video

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRemotionStatusNeverBundled 确认状态里 bundled 为假，hint 非空，并且空引擎不会报成已托管安装。
func TestRemotionStatusNeverBundled(t *testing.T) {
	st := RemotionStatus()
	if st["bundled"] != false {
		t.Fatalf("%+v", st)
	}
	if fmtSprint(st["hint"]) == "" {
		t.Fatal("hint")
	}
	if st["ready"] != false && st["source"] == "managed" {
		t.Fatal("empty engine must not report a managed install")
	}
}

// fmtSprint 把值当成字符串。nil 返回空串。
func fmtSprint(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// TestWriteRemotionProject 确认脚手架写出清单和源码，且没有 node_modules 时 managedRemotionReady 为假。
func TestWriteRemotionProject(t *testing.T) {
	e := testEngine(t)
	if err := e.writeRemotionProject(); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"package.json", "src/index.ts", "src/Root.tsx", "src/TitleCard.tsx", "remotion.config.ts"} {
		p := filepath.Join(e.remotionDir(), filepath.FromSlash(rel))
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			t.Fatalf("%s: %v", rel, err)
		}
	}
	if e.managedRemotionReady() {
		t.Fatal("scaffold without node_modules must not be ready")
	}
}

// TestInstallRemotionNeedsNode 把 node 和 npm 设成 - 后，安装阶段为 error，错误非空，ready 不为真。
func TestInstallRemotionNeedsNode(t *testing.T) {
	e := testEngine(t)
	e.rmNode = "-"
	e.rmNPM = "-"
	out := e.InstallRemotion()
	inst, _ := out["install"].(map[string]any)
	if inst == nil {
		t.Fatalf("%+v", out)
	}
	if inst["phase"] != "error" {
		t.Fatalf("phase %v", inst["phase"])
	}
	if fmtSprint(inst["error"]) == "" {
		t.Fatal("error")
	}
	if out["ready"] == true {
		t.Fatal("ready")
	}
}

// TestPresetStyles 确认预设不少于 200 条、取值不重复，且 PrefixStyle 会改写传入的提示词。
func TestPresetStyles(t *testing.T) {
	seeds := PresetStyles()
	if len(seeds) < 200 {
		t.Fatalf("%d", len(seeds))
	}
	seen := map[string]bool{}
	for _, s := range seeds {
		if s.Value == "" || s.Prompt == "" {
			t.Fatalf("%+v", s)
		}
		if seen[s.Value] {
			t.Fatalf("dup %s", s.Value)
		}
		seen[s.Value] = true
	}
	e := testEngine(t)
	got := e.PrefixStyle("anime", "a girl at dusk")
	if got == "a girl at dusk" || got == "" {
		t.Fatal(got)
	}
}
