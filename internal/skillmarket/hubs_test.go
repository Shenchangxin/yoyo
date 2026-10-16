package skillmarket

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseInstallID(t *testing.T) {
	cases := []struct {
		in, hub, remote string
	}{
		{"", HubWorkBuddy, ""},
		{"demo-skill", HubWorkBuddy, "demo-skill"},
		{"workbuddy:browser", HubWorkBuddy, "browser"},
		{"clawhub:awspace/pdf", HubClawHub, "awspace/pdf"},
		{"@clawhub:awspace/pdf", HubClawHub, "awspace/pdf"},
		{"skillhub:kennyshawchn/exportdou", HubClawSkillHub, "kennyshawchn/exportdou"},
		{"mdr:anthropics/skills/pdf", HubMDR, "anthropics/skills/pdf"},
		{"skills.sh:web-design-guidelines", HubVercel, "web-design-guidelines"},
		{"skillhubcn:find-skill-skillhub", HubSkillHubCN, "find-skill-skillhub"},
		{"skillhub.cn:dev-god", HubSkillHubCN, "dev-god"},
		{"skillhubcn:ake/dev-god", HubSkillHubCN, "ake/dev-god"},
	}
	for _, c := range cases {
		hub, remote := ParseInstallID(c.in)
		if hub != c.hub || remote != c.remote {
			t.Fatalf("%q => %s %q, want %s %q", c.in, hub, remote, c.hub, c.remote)
		}
	}
	if IsHub("") {
		t.Fatal("empty hub should not match")
	}
	if ItemID(HubClawHub, "awspace/pdf") != "clawhub:awspace/pdf" {
		t.Fatal(ItemID(HubClawHub, "awspace/pdf"))
	}
	if InstallSlug("awspace/pdf") != "pdf" {
		t.Fatal(InstallSlug("awspace/pdf"))
	}
}

func TestLoadHubSearchOnly(t *testing.T) {
	c, err := LoadHub(t.TempDir(), LoadOpts{Hub: HubSkillsMP})
	if err != nil {
		t.Fatal(err)
	}
	if !c.SearchOnly || len(c.Items) != 0 || c.Hub != HubSkillsMP {
		t.Fatalf("%+v", c)
	}
	if len(c.Hubs) != len(HubCatalog()) {
		t.Fatalf("hubs=%d", len(c.Hubs))
	}
	c, err = LoadHub(t.TempDir(), LoadOpts{Hub: HubMDR})
	if err != nil || !c.SearchOnly {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestListAndFetchClawHub(t *testing.T) {
	skill := "---\nname: pdf\ndescription: PDF toolkit for tests.\n---\n\nUse pypdf.\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/api/v1/skills" && q.Get("nonSuspiciousOnly") == "true":
			_, _ = w.Write([]byte(`{"items":[{"ownerHandle":"awspace","slug":"pdf","displayName":"Pdf","summary":"PDF toolkit","latestVersion":{"version":"1.0.0"},"stats":{"downloads":9,"stars":2}}]}`))
		case r.URL.Path == "/api/v1/search":
			_, _ = w.Write([]byte(`{"results":[{"displayName":"Pdf","featured":true,"install":{"reference":"awspace/pdf"},"native":{"ownerHandle":"awspace","skill":{"slug":"pdf","summary":"PDF toolkit","stats":{"downloads":9,"stars":2}}}}]}`))
		case r.URL.Path == "/api/v1/skills/pdf" && q.Get("ownerHandle") == "awspace" && !strings.Contains(r.URL.Path, "/file"):
			_, _ = w.Write([]byte(`{"skill":{"slug":"pdf","displayName":"Pdf"},"latestVersion":{"version":"1.0.0"},"moderation":{"isMalwareBlocked":false,"verdict":"clean"}}`))
		case r.URL.Path == "/api/v1/skills/pdf/versions/1.0.0":
			_, _ = w.Write([]byte(`{"version":{"version":"1.0.0","files":[{"path":"SKILL.md","size":80,"contentType":"text/markdown"},{"path":"pack.zip","size":12,"contentType":"application/zip"}],"security":{"status":"ok"}}}`))
		case r.URL.Path == "/api/v1/skills/pdf/file" && q.Get("path") == "SKILL.md":
			_, _ = w.Write([]byte(skill))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old, oldGet := ClawHubBase, Get
	ClawHubBase = srv.URL
	Get = func(rawURL string) ([]byte, error) {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, errStatus(resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, MaxBody))
	}
	t.Cleanup(func() { ClawHubBase, Get = old, oldGet })

	c, err := listClawHub("", 10)
	if err != nil || len(c.Items) != 1 || c.Items[0].ID != "clawhub:awspace/pdf" {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = listClawHub("pdf", 10)
	if err != nil || len(c.Items) != 1 || !c.Items[0].Featured {
		t.Fatalf("search %+v %v", c, err)
	}
	pack, err := fetchClawHub("awspace/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !packHasSkillMD(pack.Files) || len(pack.Files) != 1 {
		t.Fatalf("files=%d zip leaked?", len(pack.Files))
	}
}

func TestListAndFetchSkillHub(t *testing.T) {
	skill := "---\nname: demo\ndescription: SkillHub fixture for tests.\n---\n\nHello.\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/skills":
			_, _ = w.Write([]byte(`[{"name":"demo","slug":"demo","description":"fixture","downloads":3,"stars":1,"owner":{"handle":"skillhub"}}]`))
		case "/api/cli/search":
			_, _ = w.Write([]byte(`[{"name":"demo","slug":"demo","description":"fixture","owner":"skillhub"}]`))
		case "/api/skills/skillhub/demo":
			_, _ = w.Write([]byte(`{"name":"demo","slug":"demo","owner":{"handle":"skillhub"},"versions":[{"version":"1.0.0","skillMd":` + jsonString(skill) + `}]}`))
		case "/api/cli/skills/skillhub/demo/install":
			_, _ = w.Write([]byte(`{"skill":{"name":"demo","slug":"demo","owner":"skillhub"},"version":"1.0.0","skillMd":` + jsonString(skill) + `,"files":[{"path":"SKILL.md","content":` + jsonString(skill) + `,"isBinary":false},{"path":"notes.md","content":"notes","isBinary":false}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old, oldGet := ClawSkillHubBase, Get
	ClawSkillHubBase = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { ClawSkillHubBase, Get = old, oldGet })

	c, err := listClawSkillHub("", 10)
	if err != nil || len(c.Items) != 1 || c.Items[0].ID != "clawskillhub:skillhub/demo" {
		t.Fatalf("%+v %v", c, err)
	}
	pack, err := fetchClawSkillHub("skillhub/demo")
	if err != nil || !packHasSkillMD(pack.Files) {
		t.Fatalf("%+v %v", pack, err)
	}
	if n := len(pack.Files); n != 2 {
		t.Fatalf("want 2 files after dedupe, got %d", n)
	}
}

func TestListGitHubHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/git/trees/") {
			_, _ = w.Write([]byte(`{"sha":"abc","tree":[{"path":"skills/pdf/SKILL.md","type":"blob","size":40},{"path":"skills/pdf/scripts/run.py","type":"blob","size":12},{"path":"README.md","type":"blob","size":8}]}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/skills/pdf/SKILL.md") {
			_, _ = w.Write([]byte("---\nname: pdf\ndescription: Official PDF skill for tests.\n---\n\nDo pdf.\n"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/skills/pdf/scripts/run.py") {
			_, _ = w.Write([]byte("print(1)\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	oldAPI, oldRaw, oldGet := GitHubAPI, GitHubRaw, Get
	GitHubAPI = srv.URL
	GitHubRaw = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { GitHubAPI, GitHubRaw, Get = oldAPI, oldRaw, oldGet })

	c, err := listGitHubHub(HubAnthropic, "anthropics/skills", "main", []string{"skills"})
	if err != nil || len(c.Items) != 1 || c.Items[0].Slug != "pdf" || c.Items[0].ID != "anthropic:skills/pdf" {
		t.Fatalf("%+v %v", c, err)
	}
	pack, err := fetchGitHubSkill("anthropics/skills", "main", "skills/pdf", "https://github.com/anthropics/skills/tree/main/skills/pdf")
	if err != nil || len(pack.Files) != 2 {
		t.Fatalf("%+v %v", pack, err)
	}
}

func TestMarkdownRegistryAndSkillsMP(t *testing.T) {
	md := "---\nname: pdf\ndescription: Registry PDF skill for tests.\n---\n\nPDF.\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/search":
			_, _ = w.Write([]byte(`{"artifacts":[{"id":"art1","name":"pdf","kind":"skill","path":"skills/pdf/SKILL.md","description":"PDF","owner":"anthropics","repo":"skills","stars":9}]}`))
		case r.URL.Path == "/api/v1/resolve":
			_, _ = w.Write([]byte(`{"artifacts":[{"name":"pdf","kind":"skill","path":"skills/pdf/SKILL.md","owner":"anthropics","repo":"skills"}]}`))
		case r.URL.Path == "/api/v1/skills/search":
			_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"x","name":"nano-pdf","author":"openclaw","description":"pdf cli","githubUrl":"https://github.com/openclaw/openclaw/tree/main/skills/nano-pdf","skillUrl":"https://skillsmp.com/x","stars":3}]}}`))
		case strings.Contains(r.URL.Path, "/git/trees/"):
			_, _ = w.Write([]byte(`{"sha":"abc","tree":[{"path":"skills/pdf/SKILL.md","type":"blob","size":20}]}`))
		case strings.HasSuffix(r.URL.Path, "/skills/pdf/SKILL.md"):
			_, _ = w.Write([]byte(md))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldMDR, oldMP, oldAPI, oldRaw, oldGet := MDRBase, SkillsMPBase, GitHubAPI, GitHubRaw, Get
	MDRBase = srv.URL
	SkillsMPBase = srv.URL
	GitHubAPI = srv.URL
	GitHubRaw = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() {
		MDRBase, SkillsMPBase, GitHubAPI, GitHubRaw, Get = oldMDR, oldMP, oldAPI, oldRaw, oldGet
	})

	c, err := listMarkdownRegistry("pdf", 10)
	if err != nil || len(c.Items) != 1 || c.Items[0].ID != "mdr:anthropics/skills/pdf" {
		t.Fatalf("%+v %v", c, err)
	}
	pack, err := fetchMarkdownRegistry("anthropics/skills/pdf")
	if err != nil || !packHasSkillMD(pack.Files) {
		t.Fatalf("%+v %v", pack, err)
	}
	c, err = listSkillsMP("pdf", 10)
	if err != nil || len(c.Items) != 1 || c.Items[0].Repo != "openclaw/openclaw" {
		t.Fatalf("skillsmp %+v %v", c, err)
	}
}

func TestInstallRecordKeepsRemote(t *testing.T) {
	raw := []byte("---\nname: pdf\ndescription: Fixture pack used by hub install tests.\n---\n\nDo pdf.\n")
	_, scan := ScanSkillMD(raw, "pdf")
	if !scan.OK {
		t.Fatalf("%+v", scan)
	}
	dir, err := InstallRecord(t.TempDir(), "pdf", scan, []PackFile{{Rel: "SKILL.md", Data: raw}}, Record{
		Source:  HubClawHub,
		Catalog: "clawhub.ai",
		Remote:  "awspace/pdf",
		URL:     "https://clawhub.ai/awspace/skills/pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, ok := ReadRecord(dir)
	if !ok || rec.Remote != "awspace/pdf" || rec.Source != HubClawHub {
		t.Fatalf("%+v", rec)
	}
}

func TestParseGitHubURL(t *testing.T) {
	gh, ok := parseGitHubURL("https://github.com/openclaw/openclaw/tree/main/skills/nano-pdf")
	if !ok || gh.Repo != "openclaw/openclaw" || gh.Dir != "skills/nano-pdf" {
		t.Fatalf("%+v %v", gh, ok)
	}
	gh, ok = parseGitHubURL("https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md")
	if !ok || gh.Dir != "skills/pdf" {
		t.Fatalf("%+v %v", gh, ok)
	}
}

func TestFetchGitHubDirSkipsOversized(t *testing.T) {
	skill := "---\nname: pdf\ndescription: PDF toolkit for tests.\n---\n\nUse scripts/extract.py.\n"
	script := "print(1)\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/repos/anthropics/skills/git/trees/"):
			_, _ = w.Write([]byte(`{"sha":"abc","tree":[
			  {"path":"skills/pdf/SKILL.md","type":"blob","size":80},
			  {"path":"skills/pdf/scripts/extract.py","type":"blob","size":10},
			  {"path":"skills/pdf/references/huge.md","type":"blob","size":2000000},
			  {"path":"skills/pdf/LICENSE","type":"blob","size":0}
			]}`))
		case r.URL.Path == "/anthropics/skills/main/skills/pdf/SKILL.md":
			_, _ = w.Write([]byte(skill))
		case r.URL.Path == "/anthropics/skills/main/skills/pdf/scripts/extract.py":
			_, _ = w.Write([]byte(script))
		case r.URL.Path == "/anthropics/skills/main/skills/pdf/LICENSE":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldAPI, oldRaw, oldGet := GitHubAPI, GitHubRaw, Get
	GitHubAPI = srv.URL
	GitHubRaw = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { GitHubAPI, GitHubRaw, Get = oldAPI, oldRaw, oldGet })

	files, warn, err := fetchGitHubDir("anthropics/skills", "main", "skills/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if warn == "" {
		t.Fatal("expected skip warning")
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Rel] = true
	}
	if !got["SKILL.md"] || !got["scripts/extract.py"] {
		t.Fatalf("%+v warn=%q", packRels(files), warn)
	}
	if got["references/huge.md"] {
		t.Fatal("oversized file installed")
	}
}

func TestExpandHA(t *testing.T) {
	raw := "https://raw.githubusercontent.com/anthropics/skills/main/skills/pdf/SKILL.md"
	got := expandHA(raw)
	if len(got) < 3 || got[0] != raw {
		t.Fatalf("%v", got)
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "cdn.jsdelivr.net/gh/anthropics/skills@main/skills/pdf/SKILL.md") {
		t.Fatalf("jsdelivr missing: %v", got)
	}
	if !strings.Contains(joined, "cdn.jsdmirror.com") || !strings.Contains(joined, "gh-proxy.com") {
		t.Fatalf("cn mirrors missing: %v", got)
	}
	api := expandHA("https://api.github.com/repos/anthropics/skills/git/trees/main?recursive=1")
	if len(api) != 2 || !strings.HasPrefix(api[1], "https://gh-proxy.com/") {
		t.Fatalf("%v", api)
	}
	local := expandHA("http://127.0.0.1:9/CATALOG.md")
	if len(local) != 1 {
		t.Fatalf("%v", local)
	}
}

func TestSkillHubCNListAndFetch(t *testing.T) {
	skill := "---\nname: find-skill-skillhub\ndescription: Search SkillHub CN for tests.\n---\n\nFind skills.\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/skills":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"skills":[{"name":"find skill","slug":"find-skill-skillhub","description_zh":"在 SkillHub 搜索","category":"ai-agent","downloads":9,"stars":2,"version":"1.0.2","namespace":{"handle":"user_290ac21c"},"subCategories":[{"name":"工作流编排"}]}]}}`))
		case r.URL.Path == "/api/v1/skills/find-skill-skillhub":
			_, _ = w.Write([]byte(`{"slug":"find-skill-skillhub","latestVersion":{"version":"1.0.2"},"securityReports":{"keen":{"status":"benign"}}}`))
		case r.URL.Path == "/api/v1/skills/find-skill-skillhub/files":
			_, _ = w.Write([]byte(`{"count":2,"version":"1.0.2","files":[{"path":"SKILL.md","size":80},{"path":"notes.md","size":5}]}`))
		case r.URL.Path == "/api/v1/skills/find-skill-skillhub/file":
			if r.URL.Query().Get("path") == "SKILL.md" {
				_, _ = w.Write([]byte(skill))
				return
			}
			_, _ = w.Write([]byte("notes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old, oldGet := SkillHubCNBase, Get
	SkillHubCNBase = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { SkillHubCNBase, Get = old, oldGet })

	c, err := listSkillHubCN("", 10)
	if err != nil || len(c.Items) != 1 || c.Items[0].ID != "skillhubcn:user_290ac21c/find-skill-skillhub" {
		t.Fatalf("%+v %v", c, err)
	}
	pack, err := fetchSkillHubCN("find-skill-skillhub")
	if err != nil || !packHasSkillMD(pack.Files) || len(pack.Files) != 2 {
		t.Fatalf("%+v %v", pack, err)
	}
}

func TestClawHubListWithoutOwner(t *testing.T) {
	row := clawListItem{Slug: "self-improving-agent", DisplayName: "self-improving agent", Summary: "learn"}
	it, ok := itemFromClawList(row)
	if !ok || it.ID != "clawhub:self-improving-agent" || it.Slug != "self-improving-agent" {
		t.Fatalf("%+v %v", it, ok)
	}
}

func TestCatalogLimit(t *testing.T) {
	if catalogLimit("", 0) != defaultBrowseLimit || catalogLimit("pdf", 0) != defaultSearchLimit {
		t.Fatal(catalogLimit("", 0), catalogLimit("pdf", 0))
	}
	if catalogLimit("", 12) != 12 || catalogLimit("", 80_000) != maxCatalogLimit {
		t.Fatal(catalogLimit("", 12), catalogLimit("", 80_000))
	}
}

func TestClawHubFollowsCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/skills" {
			http.NotFound(w, r)
			return
		}
		cur := r.URL.Query().Get("cursor")
		if cur == "" {
			_, _ = w.Write([]byte(`{"items":[` + clawListJSON("alpha") + `,` + clawListJSON("beta") + `],"nextCursor":"page-2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[` + clawListJSON("gamma") + `]}`))
	}))
	defer srv.Close()
	old, oldGet := ClawHubBase, Get
	ClawHubBase = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { ClawHubBase, Get = old, oldGet })
	c, err := listClawHub("", 3)
	if err != nil || len(c.Items) != 3 {
		t.Fatalf("%+v %v", c, err)
	}
	if c.Items[0].Slug != "alpha" || c.Items[2].Slug != "gamma" {
		t.Fatalf("%+v", c.Items)
	}
}

func TestSkillHubCNPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/skills" {
			http.NotFound(w, r)
			return
		}
		page := r.URL.Query().Get("page")
		var skills []string
		if page == "2" {
			skills = []string{skillHubCNJSON("two", "keep-going")}
		} else {
			n := 100
			skills = make([]string, n)
			for i := 0; i < n; i++ {
				skills[i] = skillHubCNJSON("one", "skill-"+strconv.Itoa(i))
			}
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"total":101,"skills":[` + strings.Join(skills, ",") + `]}}`))
	}))
	defer srv.Close()
	old, oldGet := SkillHubCNBase, Get
	SkillHubCNBase = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { SkillHubCNBase, Get = old, oldGet })
	c, err := listSkillHubCN("", 101)
	if err != nil || len(c.Items) != 101 || c.Total != 101 {
		t.Fatalf("n=%d total=%d err=%v", len(c.Items), c.Total, err)
	}
}

func TestListGitHubSkillsKeepsCollidingSlugs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/git/trees/") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"sha":"abc","tree":[
		  {"path":".claude/skills/web/SKILL.md","type":"blob","size":8},
		  {"path":"skills/web/SKILL.md","type":"blob","size":8}
		]}`))
	}))
	defer srv.Close()
	oldAPI, oldGet := GitHubAPI, Get
	GitHubAPI = srv.URL
	Get = func(rawURL string) ([]byte, error) { return testHTTPGet(rawURL) }
	t.Cleanup(func() { GitHubAPI, Get = oldAPI, oldGet })
	c, err := listGitHubHub(HubClaudeSkills, "alirezarezvani/claude-skills", "main", []string{""})
	if err != nil || len(c.Items) != 2 {
		t.Fatalf("%+v %v", c, err)
	}
	ids := map[string]bool{}
	for _, it := range c.Items {
		ids[it.ID] = true
		if it.Slug != "web" {
			t.Fatalf("%+v", it)
		}
	}
	if !ids["claude-skills:.claude/skills/web"] || !ids["claude-skills:skills/web"] {
		t.Fatalf("%v", ids)
	}
}

func TestAppendTreeSkills(t *testing.T) {
	items := []Item{{Slug: "github", Name: "github", Category: "Tools"}}
	tree := gitTree{}
	tree.Tree = []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	}{
		{Path: "skills/github/SKILL.md", Type: "blob"},
		{Path: "skills/browser/SKILL.md", Type: "blob"},
		{Path: "skills/browser/scripts/run.sh", Type: "blob"},
	}
	got := appendTreeSkills(items, tree)
	if len(got) != 2 || got[1].Slug != "browser" {
		t.Fatalf("%+v", got)
	}
}

func clawListJSON(slug string) string {
	return `{"ownerHandle":"owner","slug":"` + slug + `","displayName":"` + slug + `","summary":"` + slug + `","latestVersion":{"version":"1.0.0"},"stats":{"downloads":1,"stars":1}}`
}

func skillHubCNJSON(handle, slug string) string {
	return `{"name":"` + slug + `","slug":"` + slug + `","description_zh":"` + slug + `","category":"ai-agent","downloads":1,"stars":1,"version":"1.0.0","namespace":{"handle":"` + handle + `"}}`
}

func TestParseJSDelivrTree(t *testing.T) {
	tree, err := parseJSDelivrTree([]byte(`{"files":[{"name":"/skills/pdf/SKILL.md","size":12},{"name":"/README.md","size":4}]}`))
	if err != nil || len(tree.Tree) != 2 || tree.Tree[0].Path != "skills/pdf/SKILL.md" {
		t.Fatalf("%+v %v", tree, err)
	}
}

func testHTTPGet(rawURL string) ([]byte, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, errStatus(resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxBody))
}

type statusErr string

func (e statusErr) Error() string { return string(e) }

func errStatus(s string) error { return statusErr("skill market: " + s) }

func TestLiveCatalogSize(t *testing.T) {
	if os.Getenv("YOYO_LIVE_SKILLS") == "" {
		t.Skip("set YOYO_LIVE_SKILLS=1 to list real hub catalogs")
	}
	cases := []struct {
		hub, query string
		min        int
	}{
		{HubWorkBuddy, "", 20},
		{HubClawHub, "", 200},
		{HubSkillHubCN, "", 200},
		{HubAnthropic, "", 8},
		{HubClawSkillHub, "", 20},
	}
	ok := 0
	dir := t.TempDir()
	for _, c := range cases {
		cat, err := LoadHub(dir, LoadOpts{Hub: c.hub, Query: c.query, Refresh: true})
		if err != nil {
			t.Logf("%s: %v", c.hub, err)
			continue
		}
		t.Logf("%s items=%d total=%d truncated=%v", c.hub, len(cat.Items), cat.Total, cat.Truncated)
		if len(cat.Items) < c.min {
			t.Errorf("%s too small: %d (want >= %d)", c.hub, len(cat.Items), c.min)
			continue
		}
		ok++
	}
	if ok == 0 {
		t.Fatal("no hub reachable")
	}
}

func TestLiveFetchCompleteness(t *testing.T) {
	if os.Getenv("YOYO_LIVE_SKILLS") == "" {
		t.Skip("set YOYO_LIVE_SKILLS=1 to fetch real hub packs")
	}
	cases := []struct {
		hub, remote string
		wantScript  bool
	}{
		{HubWorkBuddy, "arxiv-watcher", true},
		{HubAnthropic, "pdf", true},
		{HubSkillHubCN, "find-skill-skillhub", false},
	}
	ok := 0
	home := t.TempDir()
	for _, c := range cases {
		pack, err := FetchFrom(c.hub, c.remote)
		if err != nil {
			t.Logf("%s:%s fetch: %v", c.hub, c.remote, err)
			continue
		}
		raw := []byte{}
		for _, f := range pack.Files {
			if strings.EqualFold(f.Rel, "SKILL.md") {
				raw = f.Data
				break
			}
		}
		if len(raw) == 0 {
			t.Errorf("%s:%s missing SKILL.md files=%v warn=%q", c.hub, c.remote, packRels(pack.Files), pack.Warn)
			continue
		}
		sk, scan := ScanSkillMD(raw, InstallSlug(c.remote))
		if !scan.OK {
			t.Errorf("%s:%s scan %+v", c.hub, c.remote, scan)
			continue
		}
		dir, err := InstallRecord(home, InstallSlug(c.remote)+"-"+c.hub, scan, pack.Files, Record{Source: c.hub, Remote: c.remote})
		if err != nil {
			t.Errorf("%s:%s install: %v", c.hub, c.remote, err)
			continue
		}
		onDisk := []string{}
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			onDisk = append(onDisk, filepath.ToSlash(rel))
			return nil
		})
		incomplete := PackIncomplete(dir, sk.Body)
		t.Logf("%s:%s files=%d warn=%q incomplete=%v disk=%v", c.hub, c.remote, len(pack.Files), pack.Warn, incomplete, onDisk)
		if c.wantScript && incomplete {
			t.Errorf("%s:%s expected scripts on disk, got %v warn=%q", c.hub, c.remote, onDisk, pack.Warn)
			continue
		}
		ok++
	}
	if ok == 0 {
		t.Fatal("no hub reachable")
	}
}

func jsonString(s string) string {
	b := strings.Builder{}
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
