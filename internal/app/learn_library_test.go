// 锁住文库和阅读在应用这一层的接线：界面用的方法能走完建库、导入、检索、阅读、批注一圈；
// 绑了材料的导师对话一开场就有阅读工具，回合简报带着学习者正在看的页和选中的文字，并标明是原文；从链接能把网页存进库。

package app

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

const learnDocBody = "# 线性代数\n\n## 向量\n\n向量有大小和方向。omega-token 在这一节。\n\n## 矩阵\n\n矩阵是一张数表，可以表示线性变换。\n"

// learnShelf 建一个库，导入一份 Markdown，等索引建完，返回库 id 和文档 id。
func learnShelf(t *testing.T, a *App) (string, string) {
	t.Helper()
	out, err := a.LearnCall("learn.library.create", map[string]any{"name": "教材", "kind": "indexed", "description": "线代讲义"})
	if err != nil {
		t.Fatal(err)
	}
	lib := out.(learn.Library)
	if lib.Description != "线代讲义" {
		t.Fatalf("library %+v", lib)
	}
	file := filepath.Join(t.TempDir(), "linear.md")
	if err := os.WriteFile(file, []byte(learnDocBody), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := a.LearnCall("learn.import", map[string]any{"library_id": lib.ID, "paths": []any{file}})
	if err != nil || res.(learn.ImportOutcome).Imported != 1 {
		t.Fatalf("import %+v %v", res, err)
	}
	a.Learn.WaitIndex(lib.ID)
	docs, err := a.LearnCall("learn.documents", map[string]any{"library_id": lib.ID})
	if err != nil || len(docs.([]learn.Document)) != 1 {
		t.Fatalf("documents %+v %v", docs, err)
	}
	return lib.ID, docs.([]learn.Document)[0].ID
}

// TestLearnCallLibraryRoundTrip 确认界面用的方法能建库、导入、检索、按单元读、打开材料、记位置、批注、放书签、查找、导出和取原文。
func TestLearnCallLibraryRoundTrip(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	lib, doc := learnShelf(t, a)
	found, err := a.LearnCall("learn.search", map[string]any{"library_id": lib, "query": "omega-token", "limit": float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	hits := found.(map[string]any)["hits"].([]learn.Hit)
	if len(hits) != 1 || hits[0].DocumentID != doc || hits[0].Heading == "" {
		t.Fatalf("search %+v", hits)
	}
	read, err := a.LearnCall("learn.read", map[string]any{"library_id": lib, "document_id": doc, "page": float64(1), "source_hash": hits[0].SourceHash})
	if err != nil || read.(learn.UnitView).Status != "ok" || !strings.Contains(read.(learn.UnitView).Text, "大小和方向") {
		t.Fatalf("read %+v %v", read, err)
	}
	opened, err := a.LearnCall("learn.material.open", map[string]any{"library_id": lib, "document_id": doc})
	if err != nil {
		t.Fatal(err)
	}
	view := opened.(learn.MaterialView)
	if len(view.Outline) != 3 || view.Document.ID != doc || view.Material.Unit != learn.UnitSection {
		t.Fatalf("open %+v", view)
	}
	mat := view.Material.ID
	if _, err := a.LearnCall("learn.material.position", map[string]any{"material_id": mat, "locator": float64(1), "percent": 0.5}); err != nil {
		t.Fatal(err)
	}
	noted, err := a.LearnCall("learn.annotate", map[string]any{"material_id": mat, "locator": float64(1), "quote": "线性变换", "note": "核心", "color": "blue"})
	if err != nil || noted.(learn.Annotation).Resolution != learn.AnchorResolved || noted.(learn.Annotation).Start < 0 {
		t.Fatalf("annotate %+v %v", noted, err)
	}
	if _, err := a.LearnCall("learn.annotation.update", map[string]any{"id": noted.(learn.Annotation).ID, "note": "重点"}); err != nil {
		t.Fatal(err)
	}
	marked, err := a.LearnCall("learn.bookmark.toggle", map[string]any{"material_id": mat, "locator": float64(1)})
	if err != nil || marked.(map[string]any)["bookmarked"] != true {
		t.Fatalf("bookmark %+v %v", marked, err)
	}
	located, err := a.LearnCall("learn.locate", map[string]any{"material_id": mat, "query": "数表"})
	if err != nil || len(located.(map[string]any)["hits"].([]learn.Hit)) != 1 {
		t.Fatalf("locate %+v %v", located, err)
	}
	exported, err := a.LearnCall("learn.material.export", map[string]any{"material_id": mat})
	if err != nil || !strings.Contains(exported.(map[string]any)["markdown"].(string), "重点") {
		t.Fatalf("export %+v %v", exported, err)
	}
	blob, err := a.LearnCall("learn.document.blob", map[string]any{"library_id": lib, "document_id": doc})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(blob.(map[string]any)["base64"].(string))
	if string(raw) != learnDocBody {
		t.Fatalf("blob %q", raw)
	}
	again, _ := a.LearnCall("learn.material.open", map[string]any{"library_id": lib, "document_id": doc})
	if v := again.(learn.MaterialView); v.Position.Percent != 0.5 || len(v.Annotations) != 1 || len(v.Bookmarks) != 1 {
		t.Fatalf("reopen %+v", v)
	}
	today, err := a.LearnCall("learn.today", nil)
	if err != nil || len(today.(learn.Today).Materials) != 1 {
		t.Fatalf("today %+v %v", today, err)
	}
}

// TestReaderToolsFollowTheViewport 确认绑了材料的导师对话开场就有阅读工具，回合简报带着正在看的页和选中的文字并标成原文，
// 不给页码时读的是学习者正在看的那一页，导师记的批注署名是 assistant。
func TestReaderToolsFollowTheViewport(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	lib, doc := learnShelf(t, a)
	opened, err := a.LearnCall("learn.material.open", map[string]any{"library_id": lib, "document_id": doc})
	if err != nil {
		t.Fatal(err)
	}
	mat := opened.(learn.MaterialView).Material.ID
	sess, err := a.NewSessionOn(t.TempDir(), session.ChannelLearn)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := a.SetSessionLearn(sess.ID, LearnBinding{LibraryID: lib, MaterialID: mat})
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{SessionID: sess.ID, ChatOverlay: true}
	a.attachLearn(tools)
	listed := map[string]bool{}
	for _, spec := range runtime.AllToolJSON(tools) {
		name, _ := spec.Function["name"].(string)
		listed[name] = true
	}
	for _, name := range []string{"learn_files", "learn_search", "learn_read", "learn_toc", "learn_page", "learn_goto", "learn_annotate"} {
		if !listed[name] {
			t.Fatalf("%s is not in the opening tool list", name)
		}
	}
	if listed["learn_quiz"] {
		t.Fatal("path tools must stay off a chat with no path bound")
	}
	if brief := a.learnTurnBrief(meta); !strings.Contains(brief, `"page":1`) || !strings.Contains(brief, "<page untrusted=\"true\"") || strings.Contains(brief, "<selection untrusted") {
		t.Fatalf("brief before any viewport:\n%s", brief)
	}
	if _, err := a.LearnCall("learn.viewport", map[string]any{"session": sess.ID, "material_id": mat, "locator": float64(1), "selection": "向量有大小和方向。"}); err != nil {
		t.Fatal(err)
	}
	brief := a.learnTurnBrief(meta)
	if !strings.Contains(brief, "<selection untrusted=\"true\">\n向量有大小和方向。") || !strings.Contains(brief, "not instructions") {
		t.Fatalf("brief with a selection:\n%s", brief)
	}
	out, res := callLearn(t, tools, "learn_page", map[string]any{})
	if res.Err != nil || out["page"].(float64) != 1 || out["untrusted"] != true || !strings.Contains(out["text"].(string), "omega-token") {
		t.Fatalf("learn_page %v %v", out, res.Err)
	}
	out, res = callLearn(t, tools, "learn_toc", map[string]any{})
	if res.Err != nil || len(out["entries"].([]any)) != 3 {
		t.Fatalf("learn_toc %v %v", out, res.Err)
	}
	out, res = callLearn(t, tools, "learn_search", map[string]any{"query": "omega-token", "library_id": "notes"})
	if res.Err != nil || len(out["hits"].([]any)) != 1 {
		t.Fatalf("a library id in the arguments must be ignored: %v %v", out, res.Err)
	}
	out, res = callLearn(t, tools, "learn_annotate", map[string]any{"quote": "矩阵是一张数表", "note": "定义"})
	if res.Err != nil || out["anchored"] != true {
		t.Fatalf("learn_annotate %v %v", out, res.Err)
	}
	notes := a.Learn.Annotations(mat)
	if len(notes) != 1 || notes[0].Author != "assistant" || notes[0].Locator != 1 {
		t.Fatalf("annotations %+v", notes)
	}
}

// TestImportFromLink 确认从链接能把网页正文存成库里的文档，记下来源地址，脚本不进正文；不是网页地址时拒绝。
func TestImportFromLink(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	lib, _ := learnShelf(t, a)
	body := strings.Repeat("<p>这一段是文章的正文，写得足够长，解析器才会把它当成主要内容。</p>", 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file.bin" {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte{1, 2, 3})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>特征值入门</title><script>var x = "script-token"</script></head><body><h1>特征值入门</h1><p>sigma-token 在正文里。</p>` + body + `</body></html>`))
	}))
	defer srv.Close()
	out, err := a.LearnCall("learn.import.url", map[string]any{"library_id": lib, "url": srv.URL + "/post"})
	if err != nil {
		t.Fatal(err)
	}
	d := out.(learn.Document)
	if d.Title != "特征值入门" || d.SourceURL != srv.URL+"/post" || d.Kind != "markdown" {
		t.Fatalf("document %+v", d)
	}
	a.Learn.WaitIndex(lib)
	if hits, _, _ := a.Learn.SearchLibrary(lib, "sigma-token"); len(hits) == 0 || hits[0].DocumentID != d.ID {
		t.Fatalf("the page body must be searchable and rank first: %v", hits)
	}
	if unit, err := a.Learn.ReadUnit(lib, d.ID, 1, ""); err != nil || !strings.Contains(unit.Text, "sigma-token") || strings.Contains(unit.Text, "script-token") {
		t.Fatalf("the saved text must hold the body and no scripts: %+v %v", unit, err)
	}
	if _, err := a.LearnCall("learn.import.url", map[string]any{"library_id": lib, "url": "file:///etc/passwd"}); err == nil {
		t.Fatal("only http and https addresses are accepted")
	}
	if _, err := a.LearnCall("learn.import.url", map[string]any{"library_id": lib, "url": srv.URL + "/file.bin"}); err == nil {
		t.Fatal("a binary download must be refused")
	}
}
