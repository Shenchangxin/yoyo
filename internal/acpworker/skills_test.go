// 外部 agent 读技能目录的测试：装好的技能能读不能写，旁边的东西读不到；对方支持额外目录时技能目录交给会话。

package acpworker

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// TestFSSkillDirsReadOnly 锁住外部 agent 能读交给它的技能（SKILL.md 和包里的脚本），但不能改，也读不到技能目录旁边的文件。
// 没交技能的工人读不到技能目录。工单目录照常能写。
func TestFSSkillDirsReadOnly(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")
	skill := filepath.Join(base, "skills", "pdf")
	for _, p := range []string{work, filepath.Join(skill, "scripts")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const body = "---\nname: pdf\n---\nUse scripts/split.py.\n"
	md := filepath.Join(skill, "SKILL.md")
	script := filepath.Join(skill, "scripts", "split.py")
	sibling := filepath.Join(base, "skills", "secret.md")
	for path, text := range map[string]string{md: body, script: "print('ok')\n", sibling: "no"} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()

	bare := open(Config{Spec: superagent.WorkerSpec{TicketID: "T-0", Dir: work}}, newPool(maxLive))
	if _, err := bare.readFile(ctx, acp.ReadTextFileRequest{Path: md}); err == nil {
		t.Fatal("a worker without skills should not read the skill directory")
	}

	spec := superagent.WorkerSpec{TicketID: "T-1", Dir: work, Skills: []superagent.SkillRef{{Name: "pdf", Dir: skill}, {Name: "no-dir"}}}
	w := open(Config{Spec: spec}, newPool(maxLive))
	if got, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: md}); err != nil || got != body {
		t.Fatalf("read SKILL.md = %q, %v", got, err)
	}
	if got, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: script}); err != nil || got != "print('ok')\n" {
		t.Fatalf("read pack script = %q, %v", got, err)
	}
	if _, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: sibling}); err == nil {
		t.Fatal("a file next to the skill directory should not be readable")
	}
	fresh := filepath.Join(skill, "new.md")
	for _, p := range []string{md, fresh} {
		if err := w.writeFile(ctx, acp.WriteTextFileRequest{Path: p, Content: "changed"}); err == nil {
			t.Fatalf("write %s should be refused", p)
		}
	}
	if got, _ := os.ReadFile(md); string(got) != body {
		t.Fatalf("SKILL.md changed to %q", got)
	}
	if _, err := os.Stat(fresh); !os.IsNotExist(err) {
		t.Fatal("a refused write left a file in the skill directory")
	}
	if err := w.writeFile(ctx, acp.WriteTextFileRequest{Path: filepath.Join(work, "out.md"), Content: "ok"}); err != nil {
		t.Fatalf("ticket dir write: %v", err)
	}
}

// TestExtraDirsHandSkillDirs 锁住对方支持额外目录时，可写目录、只读目录之后接上去重后的技能目录；不支持时什么都不交。
func TestExtraDirsHandSkillDirs(t *testing.T) {
	spec := superagent.WorkerSpec{
		Roots:     []string{"r"},
		ReadRoots: []string{"ro"},
		Skills:    []superagent.SkillRef{{Name: "a", Dir: "s1"}, {Name: "b", Dir: "s1"}, {Name: "c"}, {Name: "d", Dir: "s2"}},
	}
	var on acp.AgentCapabilities
	on.SessionCapabilities.AdditionalDirectories = true
	if got := extraDirs(on, spec); !reflect.DeepEqual(got, []string{"r", "ro", "s1", "s2"}) {
		t.Fatalf("extraDirs = %v", got)
	}
	if got := extraDirs(acp.AgentCapabilities{}, spec); got != nil {
		t.Fatalf("without the capability extraDirs = %v", got)
	}
}
