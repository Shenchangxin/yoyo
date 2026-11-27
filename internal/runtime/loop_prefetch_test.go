// 锁住只读预取只落一次轨迹，以及 git_status 在非仓库目录不算错误。
// 也锁住默认策略不强制询问 shell，ask 模式会问，bypass 不问。

package runtime

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// TestGitNotRepoIsNotError 确认没有 git 仓库时 git_status 不返回错误，正文写明 not a git repository。
func TestGitNotRepoIsNotError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	res := (&WorkspaceTools{Workspace: dir}).Call("git_status", `{}`)
	if res.Err != nil {
		t.Fatalf("err %v content %q", res.Err, res.Content)
	}
	if strings.HasPrefix(res.Content, "ERROR:") {
		t.Fatalf("content %q", res.Content)
	}
	if !strings.Contains(res.Content, "not a git repository") {
		t.Fatalf("content %q", res.Content)
	}
}

// TestPolicyRequiresDoesNotForceAskDefault 确认默认策略不强制询问 shell，ask 模式会问，bypass 不问。
func TestPolicyRequiresDoesNotForceAskDefault(t *testing.T) {
	p := DefaultPolicy()
	if policyRequires(p, capability.Shell) {
		t.Fatal("default policy must not ForceAsk shell")
	}
	p.Mode = "ask"
	if !policyRequires(p, capability.Shell) {
		t.Fatal("ask mode should ForceAsk shell")
	}
	p.Mode = "bypass"
	if policyRequires(p, capability.Shell) {
		t.Fatal("bypass ForceAsk")
	}
}
