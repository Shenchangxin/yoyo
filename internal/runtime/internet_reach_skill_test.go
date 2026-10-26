// 内置 internet-reach 技能必须声明 web_search 和 reach_status，正文不得教 curl 或 mcporter。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestBundledInternetReachSkill 确认技能名为 internet-reach，allowed-tools 含 web_search 和 reach_status，正文不含 curl - 和 mcporter call。
func TestBundledInternetReachSkill(t *testing.T) {
	p := filepath.Join("..", "..", "skills", "internet-reach", "SKILL.md")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := artifact.ParseSkillMD(string(raw), p)
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "internet-reach" {
		t.Fatalf("%q", sk.Name)
	}
	if !strings.Contains(sk.AllowedTools, "web_search") || !strings.Contains(sk.AllowedTools, "reach_status") {
		t.Fatalf("allowed-tools %q", sk.AllowedTools)
	}
	lower := strings.ToLower(sk.Body)
	if strings.Contains(lower, "curl -") || strings.Contains(lower, "mcporter call") {
		t.Fatalf("bundled skill must not teach curl/mcporter:\n%s", sk.Body)
	}
}
