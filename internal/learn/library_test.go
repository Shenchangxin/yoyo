// 锁住文库和阅读的不变量：导入之后索引自己建好，一份坏文件不拖累别的文档，两个库互不相干，
// 留在原处的文件夹改了能跟上，嵌入只算一次且失败时退回词法检索，批注靠引文和前后文找回位置，找不到也不丢。

package learn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/pages"
)

// writeFile 在 dir 下写一个文件并返回它的路径。
func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// indexedLib 建一个 indexed 库，导入 files 里的每个文件（文件名到内容），并等索引建完。
func indexedLib(t *testing.T, s *Store, files map[string]string) Library {
	t.Helper()
	lib, err := s.CreateLibrary("讲义", "indexed", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var paths []string
	for name, body := range files {
		paths = append(paths, writeFile(t, dir, name, body))
	}
	if _, err := s.ImportPaths(lib.ID, paths); err != nil {
		t.Fatal(err)
	}
	s.WaitIndex(lib.ID)
	got, err := s.Library(lib.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// docByTitle 返回库里标题是 title 的文档。
func docByTitle(t *testing.T, s *Store, libraryID, title string) Document {
	t.Helper()
	docs, err := s.Documents(libraryID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if d.Title == title {
			return d
		}
	}
	t.Fatalf("no document titled %q in %+v", title, docs)
	return Document{}
}

const calculus = "# 微积分\n\n开场白。\n\n## 导数\n\n导数是瞬时变化率，也是切线的斜率。alpha-token 出现在这里。\n\n## 积分\n\n积分是曲线下的面积。\n"

// TestSearchEmptyAndCJK 确认笔记不用手动重建就能检索，检索不到时命中为空且文案固定，两个汉字的词能命中。
func TestSearchEmptyAndCJK(t *testing.T) {
	s := openTemp(t)
	pg, err := s.Create(pages.Page{Title: "材料", Content: "这是学习材料，讲检索。"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildSync(NotesLibrary); err != nil {
		t.Fatal(err)
	}
	hits, msg, err := s.SearchLibrary(NotesLibrary, "zzzz-no-such-token")
	if err != nil || len(hits) != 0 || msg != "检索完成，没有命中" {
		t.Fatalf("%v %q %v", hits, msg, err)
	}
	hits, msg, err = s.SearchLibrary(NotesLibrary, "学习")
	if err != nil || msg != "" || len(hits) == 0 || hits[0].DocumentID != pg.ID {
		t.Fatalf("%v %q %v", hits, msg, err)
	}
}

// TestImportIndexesByItself 确认导入之后不用按重建：库回到 ready，文档带着单元数和提纲，按正文和按标题都搜得到，命中带标题路径。
func TestImportIndexesByItself(t *testing.T) {
	s := openTemp(t)
	var kinds []string
	s.OnChange = func(c Change) {
		if c.Scope == "library" {
			kinds = append(kinds, c.Kind)
		}
	}
	lib := indexedLib(t, s, map[string]string{"calc.md": calculus})
	if lib.Status != LibraryReady || lib.PublishedGen == 0 || lib.Documents != 1 || lib.Progress != nil {
		t.Fatalf("library after import %+v", lib)
	}
	if joined := strings.Join(kinds, ","); !strings.Contains(joined, "indexing") || !strings.Contains(joined, "indexed") {
		t.Fatalf("change events %v, want indexing then indexed", kinds)
	}
	d := docByTitle(t, s, lib.ID, "calc")
	if d.Status != DocParsed || d.Kind != kindMarkdown || d.Unit != UnitSection || d.PageCount < 1 || d.CharCount == 0 {
		t.Fatalf("document %+v", d)
	}
	hits, msg, err := s.SearchLibrary(lib.ID, "alpha-token")
	if err != nil || msg != "" || len(hits) != 1 {
		t.Fatalf("search %v %q %v", hits, msg, err)
	}
	// 这篇很短，整篇是一个切块。标题路径按命中的词前面最近的标题修正，落在它真正所在的小节上。
	if h := hits[0]; h.DocumentID != d.ID || h.Heading != "微积分 > 导数" || h.Page != 1 || h.SourceHash != d.SHA256 || !strings.Contains(h.Excerpt, "alpha-token") {
		t.Fatalf("hit %+v", h)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "曲线下的面积"); len(hits) != 1 || hits[0].Heading != "微积分 > 积分" {
		t.Fatalf("a hit in a later subsection must name that subsection: %+v", hits)
	}
	// 标题也进了索引：正文里没有「积分」两个字以外的词，按章节名照样搜得到。
	if hits, _, _ := s.SearchLibrary(lib.ID, "积分"); len(hits) == 0 {
		t.Fatal("a heading word must be searchable")
	}
	unit, err := s.ReadUnit(lib.ID, d.ID, 1, hits[0].SourceHash)
	if err != nil || unit.Status != unitOK || unit.Pages != d.PageCount || !strings.Contains(unit.Text, "切线的斜率") {
		t.Fatalf("read %+v %v", unit, err)
	}
	if stale, err := s.ReadUnit(lib.ID, d.ID, 1, "deadbeef"); err != nil || stale.Status != "changed" || stale.Text != "" {
		t.Fatalf("a stale hash must report changed without text: %+v %v", stale, err)
	}
	if _, err := s.ReadUnit(lib.ID, d.ID, 99, ""); err == nil {
		t.Fatal("a locator past the end must fail")
	}
}

// TestBadFileDoesNotBlockOthers 确认一份解析不了的文件只把自己标成失败，库照样发布，别的文档照样能搜。
func TestBadFileDoesNotBlockOthers(t *testing.T) {
	s := openTemp(t)
	lib := indexedLib(t, s, map[string]string{"good.txt": "beta unique words here", "broken.pdf": "this is not a pdf"})
	if lib.Status != LibraryReady || lib.PublishedGen == 0 {
		t.Fatalf("library %+v", lib)
	}
	if bad := docByTitle(t, s, lib.ID, "broken"); bad.Status != DocFailed || bad.Error == "" {
		t.Fatalf("broken file %+v", bad)
	}
	if hits, _, err := s.SearchLibrary(lib.ID, "beta"); err != nil || len(hits) != 1 {
		t.Fatalf("the good file must stay searchable: %v %v", hits, err)
	}
	out, err := s.ImportPaths(lib.ID, []string{writeFile(t, t.TempDir(), "notes.xyz", "x"), filepath.Join(t.TempDir(), "missing.md")})
	if err != nil || out.Imported != 0 || len(out.Skipped) != 2 {
		t.Fatalf("unsupported and missing paths must be skipped with reasons: %+v %v", out, err)
	}
}

// TestTwoLibrariesStayApart 确认两个库各自建索引互不冲突，在一个库里搜不到另一个库的内容。
func TestTwoLibrariesStayApart(t *testing.T) {
	s := openTemp(t)
	a := indexedLib(t, s, map[string]string{"a.md": "gamma only in the first library"})
	b := indexedLib(t, s, map[string]string{"b.md": "delta only in the second library"})
	if a.Status != LibraryReady || b.Status != LibraryReady {
		t.Fatalf("both libraries must index: %+v %+v", a, b)
	}
	if hits, _, _ := s.SearchLibrary(a.ID, "gamma"); len(hits) != 1 {
		t.Fatalf("first library %v", hits)
	}
	if hits, _, _ := s.SearchLibrary(b.ID, "delta"); len(hits) != 1 {
		t.Fatalf("second library %v", hits)
	}
	if hits, _, _ := s.SearchLibrary(a.ID, "delta"); len(hits) != 0 {
		t.Fatalf("a search must not cross libraries: %v", hits)
	}
	if err := s.DeleteLibrary(a.ID); err != nil {
		t.Fatal(err)
	}
	if hits, _, _ := s.SearchLibrary(b.ID, "delta"); len(hits) != 1 {
		t.Fatal("deleting one library must not touch the other")
	}
	if err := s.DeleteLibrary(NotesLibrary); err == nil {
		t.Fatal("the notes library must stay")
	}
}

// TestSearchPrefersAllTerms 确认每个词都出现的切块排在只含其中一个词的切块前面，哪怕后者把那个词重复了很多遍。
func TestSearchPrefersAllTerms(t *testing.T) {
	s := openTemp(t)
	filler := strings.Repeat("unrelated filler sentence that makes this document long. ", 12)
	lib := indexedLib(t, s, map[string]string{
		"loud.md":  "token token token token token token token token",
		"quiet.md": filler + "sigma token appears once here.",
	})
	hits, _, err := s.SearchLibrary(lib.ID, "sigma token")
	if err != nil || len(hits) != 2 || hits[0].Title != "quiet" {
		t.Fatalf("the chunk with every term must rank first: %+v %v", hits, err)
	}
}

// TestRemoveAndReimportFollowContent 确认拿掉文档后搜不到它，换了内容再导入时新内容搜得到、旧内容搜不到。
func TestRemoveAndReimportFollowContent(t *testing.T) {
	s := openTemp(t)
	lib, _ := s.CreateLibrary("讲义", "indexed", "")
	dir := t.TempDir()
	path := writeFile(t, dir, "note.md", "epsilon first version")
	if n, err := s.Import(lib.ID, path); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	s.WaitIndex(lib.ID)
	if n, err := s.ImportPaths(lib.ID, []string{path}); err != nil || n.Imported != 0 {
		t.Fatalf("importing unchanged bytes again is not an import: %+v %v", n, err)
	}
	writeFile(t, dir, "note.md", "zeta second version")
	if n, err := s.Import(lib.ID, path); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	s.WaitIndex(lib.ID)
	if hits, _, _ := s.SearchLibrary(lib.ID, "epsilon"); len(hits) != 0 {
		t.Fatalf("old content still searchable: %v", hits)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "zeta"); len(hits) != 1 {
		t.Fatal("new content must be searchable")
	}
	d := docByTitle(t, s, lib.ID, "note")
	if err := s.RenameDocument(lib.ID, d.ID, "改过的名字"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveDocument(lib.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	s.WaitIndex(lib.ID)
	if hits, _, _ := s.SearchLibrary(lib.ID, "zeta"); len(hits) != 0 {
		t.Fatalf("a removed document must leave the index: %v", hits)
	}
	if docs, _ := s.Documents(lib.ID); len(docs) != 0 {
		t.Fatalf("documents %+v", docs)
	}
}

// TestLinkedLibraryFollowsFolder 确认 linked 库建好就扫文件夹，之后文件新增、改动和删除都在重建时跟上，原文不抄进 raw。
func TestLinkedLibraryFollowsFolder(t *testing.T) {
	s := openTemp(t)
	dir := t.TempDir()
	writeFile(t, dir, "one.md", "eta in the first file")
	writeFile(t, dir, ".hidden/skip.md", "hidden-token")
	lib, err := s.CreateLibrary("文件夹", "linked", dir)
	if err != nil {
		t.Fatal(err)
	}
	s.WaitIndex(lib.ID)
	if hits, _, _ := s.SearchLibrary(lib.ID, "eta"); len(hits) != 1 {
		t.Fatal("the folder must be indexed on creation")
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "hidden-token"); len(hits) != 0 {
		t.Fatal("hidden folders are skipped")
	}
	if entries, _ := os.ReadDir(filepath.Join(s.Root(), "raw")); len(entries) != 0 {
		t.Fatalf("a linked library must not copy files: %v", entries)
	}
	writeFile(t, dir, "one.md", "theta replaced the first file")
	writeFile(t, dir, "sub/two.txt", "iota in a new file")
	if err := s.RebuildSync(lib.ID); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]int{"eta": 0, "theta": 1, "iota": 1} {
		if hits, _, _ := s.SearchLibrary(lib.ID, query); len(hits) != want {
			t.Fatalf("after edits, %q has %d hits, want %d", query, len(hits), want)
		}
	}
	if err := os.Remove(filepath.Join(dir, "one.md")); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildSync(lib.ID); err != nil {
		t.Fatal(err)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "theta"); len(hits) != 0 {
		t.Fatal("a file deleted from the folder must leave the index")
	}
	if _, err := s.CreateLibrary("坏的", "linked", filepath.Join(dir, "nope")); err == nil {
		t.Fatal("a missing folder must be refused")
	}
}

// fakeEmbed 返回一个按关键词分桶的嵌入函数和一个计数器，计数器记下一共嵌入了多少段文字。
// 「feline」和「猫」落在同一个桶里，用来验证没有共同字面的查询能靠向量命中。
func fakeEmbed() (EmbedFunc, *int) {
	calls := 0
	buckets := [][]string{{"猫", "feline"}, {"狗", "canine"}, {"鱼", "aquatic"}}
	return func(model string, texts []string) ([][]float32, error) {
		out := make([][]float32, len(texts))
		for i, text := range texts {
			calls++
			vec := make([]float32, len(buckets)+1)
			vec[len(buckets)] = 0.01
			for b, words := range buckets {
				for _, w := range words {
					vec[b] += float32(strings.Count(strings.ToLower(text), w))
				}
			}
			out[i] = vec
		}
		return out, nil
	}, &calls
}

// TestEmbeddingsFuseAndCache 确认配了嵌入模型后没有共同字面的查询也能命中，同一段文字重建时不再嵌入第二次；
// 嵌入出错时这一代照样发布，只剩词法检索，并把原因留在库上。
func TestEmbeddingsFuseAndCache(t *testing.T) {
	s := openTemp(t)
	embed, calls := fakeEmbed()
	s.Embed = embed
	lib := indexedLib(t, s, map[string]string{"cat.md": "猫喜欢晒太阳。", "dog.md": "狗喜欢散步。", "fish.md": "鱼住在水里。"})
	if err := s.SetEmbedModel(lib.ID, "test-embed"); err != nil {
		t.Fatal(err)
	}
	s.WaitIndex(lib.ID)
	first := *calls
	if first != 3 {
		t.Fatalf("embedded %d texts, want 3 chunks", first)
	}
	hits, _, err := s.SearchLibrary(lib.ID, "feline")
	if err != nil || len(hits) == 0 || hits[0].Title != "cat" {
		t.Fatalf("a query with no shared words must hit through vectors: %+v %v", hits, err)
	}
	afterQuery := *calls
	if err := s.RebuildSync(lib.ID); err != nil {
		t.Fatal(err)
	}
	if *calls != afterQuery {
		t.Fatalf("a rebuild embedded %d more texts, want 0 from the cache", *calls-afterQuery)
	}
	s.Embed = func(string, []string) ([][]float32, error) { return nil, fmt.Errorf("endpoint down") }
	if _, err := s.ImportPaths(lib.ID, []string{writeFile(t, t.TempDir(), "bird.md", "鸟会飞。")}); err != nil {
		t.Fatal(err)
	}
	s.WaitIndex(lib.ID)
	got, _ := s.Library(lib.ID)
	if got.Status != LibraryReady || !strings.Contains(got.LastError, "endpoint down") {
		t.Fatalf("an embedding failure must still publish and say why: %+v", got)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "鸟会飞"); len(hits) != 1 {
		t.Fatal("keyword search must keep working without vectors")
	}
}

// TestLongDocumentIsCutIntoSections 确认长文按标题和段落切成多个单元，每个单元不超过上限，提纲指向标题所在的单元，网页里的脚本和导航不进正文。
func TestLongDocumentIsCutIntoSections(t *testing.T) {
	s := openTemp(t)
	var md strings.Builder
	md.WriteString("# 长文\n\n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&md, "## 第%d章\n\n", i)
		for j := 0; j < 12; j++ {
			fmt.Fprintf(&md, "这是第%d章的第%d段，讲一些要点，把篇幅撑到足够长，好让切节的规则起作用。\n\n", i, j+1)
		}
	}
	body := strings.Repeat("<p>这一段是网页的正文，写得足够长，好让解析器确信这里才是主要内容而不是导航。</p>", 8)
	page := `<html><head><title>页</title><script>var secret = "script-token";</script></head><body><nav>nav-token</nav><h1>网页标题</h1><p>kappa 在网页正文里。</p>` + body + `</body></html>`
	lib := indexedLib(t, s, map[string]string{"long.md": md.String(), "page.html": page})
	d := docByTitle(t, s, lib.ID, "long")
	if d.PageCount < 2 {
		t.Fatalf("a long document must be cut into several sections, got %d", d.PageCount)
	}
	view, err := s.OpenMaterial(lib.ID, d.ID)
	if err != nil || len(view.Outline) != 7 || view.Outline[0].Locator != 1 || view.Outline[6].Locator != d.PageCount {
		t.Fatalf("outline %+v %v", view.Outline, err)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "第3章的第7段"); len(hits) == 0 || hits[0].Heading != "长文 > 第3章" || hits[0].Page < 1 {
		t.Fatalf("a hit must carry its heading path: %+v", hits)
	}
	for locator := 1; locator <= d.PageCount; locator++ {
		unit, err := s.ReadUnit(lib.ID, d.ID, locator, "")
		if err != nil || len([]rune(unit.Text)) > sectionCap || strings.TrimSpace(unit.Text) == "" {
			t.Fatalf("section %d: %d runes %v", locator, len([]rune(unit.Text)), err)
		}
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "kappa"); len(hits) != 1 || hits[0].Heading != "网页标题" {
		t.Fatalf("html body %v", hits)
	}
	for _, hidden := range []string{"script-token", "nav-token"} {
		if hits, _, _ := s.SearchLibrary(lib.ID, hidden); len(hits) != 0 {
			t.Fatalf("%s must not be indexed", hidden)
		}
	}
}

// TestReaderKeepsPlaceMarksAndNotes 确认阅读台记住读到的位置，书签能放能拿，批注在引文挪了位置或换行不同时照样定位，
// 找不到引文也留着并标成未定位，还能在材料里逐字查找并把批注导出来。
func TestReaderKeepsPlaceMarksAndNotes(t *testing.T) {
	s := openTemp(t)
	lib := indexedLib(t, s, map[string]string{"calc.md": calculus})
	d := docByTitle(t, s, lib.ID, "calc")
	view, err := s.OpenMaterial(lib.ID, d.ID)
	if err != nil || view.Position.Locator != 1 || view.Material.Kind != kindMarkdown || len(view.Outline) != 3 || len(view.Annotations) != 0 {
		t.Fatalf("first open %+v %v", view, err)
	}
	mat := view.Material.ID
	if again, _ := s.OpenMaterial(lib.ID, d.ID); again.Material.ID != mat {
		t.Fatal("opening the same document twice must reuse the material")
	}
	if err := s.SavePosition(mat, 1, 0.4); err != nil {
		t.Fatal(err)
	}
	on, marks, err := s.ToggleBookmark(mat, 1, "回来看")
	if err != nil || !on || len(marks) != 1 || marks[0].Label != "回来看" {
		t.Fatalf("bookmark %v %+v %v", on, marks, err)
	}
	unit, _ := s.ReadUnit(lib.ID, d.ID, 1, "")
	quote := "切线的斜率"
	want := len([]rune(unit.Text[:strings.Index(unit.Text, quote)]))
	a, err := s.Annotate(AnnotateRequest{MaterialID: mat, Locator: 1, Quote: quote, Note: "几何意义", Start: 3, Color: "green"})
	if err != nil || a.Resolution != AnchorResolved || a.Start != want || a.End != want+len([]rune(quote)) || a.Kind != AnnotHighlight || a.Color != "green" || a.Prefix == "" {
		t.Fatalf("a wrong start hint must be corrected from the text: %+v %v (want start %d)", a, err, want)
	}
	loose, err := s.Annotate(AnnotateRequest{MaterialID: mat, Locator: 1, Quote: "导数是瞬时变化率，\n也是切线的斜率", Start: -1, Author: "assistant"})
	if err != nil || loose.Resolution != AnchorResolved || loose.Quote != "导数是瞬时变化率，也是切线的斜率" || loose.Author != "assistant" {
		t.Fatalf("a quote broken by a line break must still resolve: %+v %v", loose, err)
	}
	spaced, err := s.Annotate(AnnotateRequest{MaterialID: mat, Locator: 1, Quote: "alpha-token   出现在这里", Start: -1})
	if err != nil || spaced.Resolution != AnchorResolved || spaced.Quote != "alpha-token 出现在这里" {
		t.Fatalf("a quote that differs only in spacing must resolve to the source text: %+v %v", spaced, err)
	}
	lost, err := s.Annotate(AnnotateRequest{MaterialID: mat, Locator: 1, Quote: "原文里没有这句话", Note: "留着"})
	if err != nil || lost.Resolution != AnchorUnresolved || lost.Start != -1 {
		t.Fatalf("a quote that is not in the text must be kept unresolved: %+v %v", lost, err)
	}
	if _, err := s.Annotate(AnnotateRequest{MaterialID: mat}); err == nil {
		t.Fatal("an empty annotation must be refused")
	}
	note, color := "改过的笔记", "purple"
	edited, err := s.UpdateAnnotation(a.ID, &note, &color, nil)
	if err != nil || edited.Note != note || edited.Color != color || edited.Start != want {
		t.Fatalf("update %+v %v", edited, err)
	}
	bad := "red"
	if _, err := s.UpdateAnnotation(a.ID, nil, &bad, nil); err == nil {
		t.Fatal("an unknown color must be refused")
	}
	if err := s.DeleteAnnotation(lost.ID); err != nil {
		t.Fatal(err)
	}
	view, err = s.MaterialView(mat)
	if err != nil || view.Position.Percent != 0.4 || len(view.Bookmarks) != 1 || len(view.Annotations) != 3 || view.Material.Annotations != 3 {
		t.Fatalf("reopened view %+v %v", view, err)
	}
	if view.Annotations[0].Start > view.Annotations[1].Start {
		t.Fatalf("annotations must come in reading order: %+v", view.Annotations)
	}
	hits, msg, err := s.Locate(mat, "曲线下的面积")
	if err != nil || msg != "" || len(hits) != 1 || hits[0].Page != 1 || !strings.Contains(hits[0].Excerpt, "曲线下的面积") {
		t.Fatalf("locate %v %q %v", hits, msg, err)
	}
	if hits, msg, _ := s.Locate(mat, "zzz-nowhere"); len(hits) != 0 || msg == "" {
		t.Fatalf("locate miss %v %q", hits, msg)
	}
	name, body, err := s.ExportMaterial(mat)
	if err != nil || !strings.HasSuffix(name, ".md") || !strings.Contains(body, "> 切线的斜率") || !strings.Contains(body, note) || !strings.Contains(body, "回来看") {
		t.Fatalf("export %q %q %v", name, body, err)
	}
	if off, marks, _ := s.ToggleBookmark(mat, 1, ""); off || len(marks) != 0 {
		t.Fatal("toggling again must remove the bookmark")
	}
	recent := s.RecentMaterials(4)
	if len(recent) != 1 || recent[0].ID != mat || recent[0].Annotations != 3 {
		t.Fatalf("recent %+v", recent)
	}
	if err := s.RemoveDocument(lib.ID, d.ID); err != nil {
		t.Fatal(err)
	}
	if recent := s.RecentMaterials(4); len(recent) != 0 {
		t.Fatalf("a material whose document was removed is not listed: %+v", recent)
	}
}

// TestAnchorPrefersMatchingContext 确认同一句话出现两次时，批注落在前后文对得上的那一处；起点对得上时直接用起点。
func TestAnchorPrefersMatchingContext(t *testing.T) {
	text := "第一段说：这个结论成立。然后讨论别的。第二段也说：这个结论成立。到此结束。"
	quote := "这个结论成立"
	second := len([]rune(text[:strings.LastIndex(text, quote)]))
	start, end, ok := anchorQuote(text, quote, -1, "第二段也说：", "。到此结束")
	if !ok || start != second || end != second+len([]rune(quote)) {
		t.Fatalf("anchor %d-%d %v, want the second occurrence at %d", start, end, ok, second)
	}
	first := len([]rune(text[:strings.Index(text, quote)]))
	if start, _, ok := anchorQuote(text, quote, first, "", ""); !ok || start != first {
		t.Fatalf("an exact start hint must win: %d %v", start, ok)
	}
	if _, _, ok := anchorQuote(text, "没有这句", -1, "", ""); ok {
		t.Fatal("a missing quote must not anchor")
	}
}

// TestInterruptedIndexResumes 确认上次退出时还在建的索引在打开后标成待重建，接上之后建完；笔记库有新笔记时自己重建。
func TestInterruptedIndexResumes(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	lib := indexedLib(t, s, map[string]string{"a.md": "lambda survives a restart"})
	if _, err := s.db.Exec(`UPDATE libraries SET status = 'building' WHERE id = ?`, lib.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if got, _ := s.Library(lib.ID); got.Status != LibraryStale {
		t.Fatalf("an interrupted index must reopen as stale: %+v", got)
	}
	s.ResumeIndexing()
	s.WaitIndex(lib.ID)
	if got, _ := s.Library(lib.ID); got.Status != LibraryReady {
		t.Fatalf("after resuming %+v", got)
	}
	if hits, _, _ := s.SearchLibrary(lib.ID, "lambda"); len(hits) != 1 {
		t.Fatal("the index must survive a restart")
	}
	if _, err := s.Create(pages.Page{Title: "新笔记", Content: "mu 出现在一篇新笔记里。"}); err != nil {
		t.Fatal(err)
	}
	// 笔记一保存就排了重建，排队本身在后台，这里等它真的排上再等建完。
	for i := 0; i < 200; i++ {
		if hits, _, _ := s.SearchLibrary(NotesLibrary, "mu"); len(hits) == 1 {
			return
		}
		s.WaitIndex(NotesLibrary)
	}
	t.Fatal("a saved note must become searchable without a manual rebuild")
}
