package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/skillmarket"
)

func TestInstallMarketSkill(t *testing.T) {
	md := "## 1. 技能包 skills/\n\n### Tools（1）\n\n| 目录 | 名称 | 用来做什么 | 前置条件 |\n|------|------|------------|----------|\n| `demo-skill` | demo-skill | fixture | 无 |\n"
	skill := "---\nname: demo-skill\ndescription: Fixture pack used by market install tests.\n---\n\nSay hello via `scripts/hello.py`.\n"
	script := "print('hello')\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/CATALOG.md":
			_, _ = w.Write([]byte(md))
		case "/skills/demo-skill/SKILL.md":
			_, _ = w.Write([]byte(skill))
		case "/skills/demo-skill/scripts/hello.py":
			_, _ = w.Write([]byte(script))
		case "/contents/skills/demo-skill":
			_, _ = w.Write([]byte(`[
			  {"name":"SKILL.md","path":"skills/demo-skill/SKILL.md","type":"file"},
			  {"name":"scripts","path":"skills/demo-skill/scripts","type":"dir"}
			]`))
		case "/contents/skills/demo-skill/scripts":
			_, _ = w.Write([]byte(`[{"name":"hello.py","path":"skills/demo-skill/scripts/hello.py","type":"file"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	oldBase, oldContents := skillmarket.Base, skillmarket.ContentsBase
	skillmarket.Base = srv.URL
	skillmarket.ContentsBase = srv.URL + "/contents"
	t.Cleanup(func() {
		skillmarket.Base = oldBase
		skillmarket.ContentsBase = oldContents
	})

	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	out, err := a.InstallMarketSkill("demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("%+v", out)
	}
	sk := a.GetSkill("", "demo-skill")
	if sk == nil || sk["source"] != "market" {
		t.Fatalf("%+v", sk)
	}
	if sk["incomplete"] == "1" {
		t.Fatalf("pack still incomplete: %+v", sk)
	}
	if _, err := os.Stat(filepath.Join(a.Home.Skills(), "demo-skill", "scripts", "hello.py")); err != nil {
		t.Fatal(err)
	}
	if err := a.UninstallMarketSkill("demo-skill"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Home.Skills(), "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("still on disk: %v", err)
	}
}
