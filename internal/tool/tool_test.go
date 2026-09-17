package tool

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type staticTool struct{ s artifact.ToolSpec }

func (s staticTool) Spec() artifact.ToolSpec { return s.s }
func (s staticTool) Annotations() Annotations {
	return AnnFromSpec(s.s)
}
func (s staticTool) Call(context.Context, Invocation) Result { return Result{} }

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

func TestHostSpecsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range HostSpecs() {
		if s.Name == "" || seen[s.Name] {
			t.Fatalf("bad spec %q", s.Name)
		}
		seen[s.Name] = true
	}
	if !seen["web_search"] || !seen["ask_user"] || !seen["run_skill_script"] {
		t.Fatal("missing new tools")
	}
}
