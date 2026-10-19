package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

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
