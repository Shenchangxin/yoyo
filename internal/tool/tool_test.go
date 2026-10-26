// 检查规格过滤，以及内置规格的名字不重复。

package tool

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type staticTool struct{ s artifact.ToolSpec }

// Spec 返回夹具里的规格。
func (s staticTool) Spec() artifact.ToolSpec { return s.s }

// Annotations 用 AnnFromSpec 从夹具规格抄注解。
func (s staticTool) Annotations() Annotations {
	return AnnFromSpec(s.s)
}

// Call 返回空结果。
func (s staticTool) Call(context.Context, Invocation) Result { return Result{} }

// TestFilterAdvertiseAndAllowed 确认广告列表里的 read_file 和始终允许的 load_skill 留下，shell 被允许列表排除。
func TestFilterAdvertiseAndAllowed(t *testing.T) {
	r := NewRegistry()
	for _, s := range HostSpecs() {
		r.Register(staticTool{s})
	}
	got := r.Filter([]string{"read_file", "shell", "load_skill"}, []string{"read_file"})
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if !names["read_file"] || !names["load_skill"] {
		t.Fatalf("%v", names)
	}
	if names["shell"] {
		t.Fatal("shell should be excluded by allowed-tools")
	}
}

// TestHostSpecsUnique 确认 HostSpecs 的名字非空且不重复，并包含 web_search、ask_user、run_skill_script 和浏览器工具。
func TestHostSpecsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range HostSpecs() {
		if s.Name == "" || seen[s.Name] {
			t.Fatalf("bad spec %q", s.Name)
		}
		seen[s.Name] = true
	}
	if !seen["web_search"] || !seen["ask_user"] || !seen["run_skill_script"] || !seen["reach_status"] {
		t.Fatal("missing new tools")
	}
	if !seen["browser_act"] || !seen["browser_verify"] {
		t.Fatal("missing browser_act/browser_verify")
	}
	if !seen["browser_scroll"] || !seen["browser_tabs"] || !seen["browser_record"] || !seen["browser_profile"] {
		t.Fatal("missing browser interaction tools")
	}
}
