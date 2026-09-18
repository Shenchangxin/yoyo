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
	skill := "---\nname: demo-skill\ndescription: Fixture pack used by market install tests.\n---\n\nSay hello.\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/CATALOG.md":
			_, _ = w.Write([]byte(md))
		case "/skills/demo-skill/SKILL.md":
			_, _ = w.Write([]byte(skill))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	old := skillmarket.Base
	skillmarket.Base = srv.URL
	t.Cleanup(func() { skillmarket.Base = old })

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
	if err := a.UninstallMarketSkill("demo-skill"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Home.Skills(), "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("still on disk: %v", err)
	}
}
