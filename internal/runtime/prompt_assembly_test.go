// 锁住提示拼装里仍在生产上用到的几条：缓存键、Harbor 提示带上手册和工作区规则、桌面身份不钉语言、行为文案的两种模式。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestPromptCacheKeyStableAndBusts 确认同一 harness、前缀和工具列表得到同一个非空键，harness 从 h1 换成 h2 后键必须改变。
func TestPromptCacheKeyStableAndBusts(t *testing.T) {
	prefix := HarborPrompt(DefaultLoop(), nil, artifact.Playbook{}, nil, nil)
	tools := BuiltinToolJSON()
	a := PromptCacheKey("h1", prefix, tools)
	b := PromptCacheKey("h1", prefix, tools)
	if a != b || a == "" {
		t.Fatalf("%s %s", a, b)
	}
	if PromptCacheKey("h2", prefix, tools) == a {
		t.Fatal("harness change must bust cache")
	}
}

// TestHarborPromptCarriesPlaybookAndRules 确认 Harbor 一题的提示里有手册条目和工作区 YOYO.md 的规则，同一份材料拼两次字节相同。
// evolve 改的是手册和片段，这两样进不了提示，Harbor 打分就没有意义。
func TestHarborPromptCarriesPlaybookAndRules(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "YOYO.md"), []byte("always use tabs"), 0o644); err != nil {
		t.Fatal(err)
	}
	pb := artifact.Playbook{ID: "p", Bullets: []artifact.PlaybookBullet{
		{ID: "b1", Text: "prefer grep", Helpful: 3},
		{ID: "b2", Text: "small diffs", Helpful: 1},
	}}
	tools := &WorkspaceTools{Workspace: dir}
	first := HarborPrompt(DefaultLoop(), nil, pb, nil, tools)
	for _, want := range []string{"prefer grep", "YOYO.md", "always use tabs"} {
		if !strings.Contains(first, want) {
			t.Fatalf("harbor prompt missing %q:\n%s", want, first)
		}
	}
	if again := HarborPrompt(DefaultLoop(), nil, pb, nil, tools); again != first {
		t.Fatal("harbor prompt not deterministic for the same materials")
	}
}

// TestChatIdentityDoesNotPinALanguage 确认桌面身份加上行为文案后不含 ## Voice，也不含「用中文回复」。语言跟着人这一轮的话走，见 LanguagePin。
func TestChatIdentityDoesNotPinALanguage(t *testing.T) {
	s := AssembleIdentity(DefaultLoop(), ChatConduct(ConductOpts{}))
	if strings.Contains(s, "## Voice") || strings.Contains(s, "用中文回复") {
		t.Fatalf("language must follow the user message, not a voice pin: %s", s)
	}
}

// TestChatConductMethodologyYieldsWriteNow 锁住默认行为含立即写产物，方法论为真时改成先 load_skill。
func TestChatConductMethodologyYieldsWriteNow(t *testing.T) {
	plain := fragmentText(ChatConduct(ConductOpts{}))
	if !strings.Contains(plain, "start writing those workspace artifacts this turn") {
		t.Fatal("default conduct must keep write-now")
	}
	if strings.Contains(plain, "invoke the matching process skill") {
		t.Fatal("default must not require process skills")
	}
	if !strings.Contains(plain, "load_skill for that skill before writing") {
		t.Fatal("default must load a matching catalog skill")
	}
	if !strings.Contains(plain, "Do not load methodology process skills") {
		t.Fatal("default must keep process skills methodology-only")
	}
	meth := fragmentText(ChatConduct(ConductOpts{Methodology: true}))
	if strings.Contains(meth, "start writing those workspace artifacts this turn") {
		t.Fatal("methodology must yield write-now")
	}
	if !strings.Contains(meth, "load_skill") || !strings.Contains(meth, "Do not skip a skill") {
		t.Fatal(meth)
	}
	if !strings.Contains(meth, "read_skill_file") {
		t.Fatal("methodology conduct must name read_skill_file")
	}
}
