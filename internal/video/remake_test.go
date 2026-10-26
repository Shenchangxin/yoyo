// remake_test.go 检查翻拍草稿保留参考哈希，以及替换画面时丢掉旧片段字节。

package video

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// TestRemakeDraftKeepsTheReference 确认草稿的种类、参考哈希、源哈希和标题被写进剪辑工程。
func TestRemakeDraftKeepsTheReference(t *testing.T) {
	e := testEngine(t)
	hash, err := e.PutBytes([]byte("reference-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.RemakeDraft(hash, "街拍", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != "remake" || p.ReferenceHash != hash || p.SourceHash != hash {
		t.Fatalf("draft %+v", p)
	}
	if p.Title != "街拍" {
		t.Fatalf("title %q", p.Title)
	}
	got, err := e.GetCutProject(p.ID)
	if err != nil || got.Kind != "remake" || got.ReferenceHash != hash {
		t.Fatalf("stored %+v %v", got, err)
	}
}

// TestRemakeDraftRejectsAnEmptyReference 确认空白参考哈希返回 reference hash required。
func TestRemakeDraftRejectsAnEmptyReference(t *testing.T) {
	e := testEngine(t)
	if _, err := e.RemakeDraft("  ", "", ""); err == nil || !strings.Contains(err.Error(), "reference hash required") {
		t.Fatal(err)
	}
}

// TestRemakeReplaceKeepsTheReferenceAndDropsClipBytes 确认参考哈希不变，源哈希换成新画面，原片段哈希不再是 from-reference，起止仍是 0 到 4。
func TestRemakeReplaceKeepsTheReferenceAndDropsClipBytes(t *testing.T) {
	e := testEngine(t)
	ref, err := e.PutBytes([]byte("reference-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	mine, err := e.PutBytes([]byte("my-picture"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CreateCutProject(CutProject{
		Title:         "翻拍",
		Kind:          "remake",
		ReferenceHash: ref,
		SourceHash:    ref,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Duration = 12
	p.EditList.Cuts = []cut.Clip{{ID: "a", Hash: "from-reference", Start: 0, End: 4}}
	p, err = e.saveCut(p)
	if err != nil {
		t.Fatal(err)
	}
	next, err := e.RemakeReplace(p.ID, mine)
	if err != nil {
		t.Fatal(err)
	}
	if next.ReferenceHash != ref || next.SourceHash != mine {
		t.Fatalf("hashes ref=%s source=%s", next.ReferenceHash, next.SourceHash)
	}
	if len(next.EditList.Cuts) != 1 || next.EditList.Cuts[0].Hash == "from-reference" {
		t.Fatalf("windows %+v", next.EditList.Cuts)
	}
	if next.EditList.Cuts[0].Start != 0 || next.EditList.Cuts[0].End != 4 {
		t.Fatalf("times %+v", next.EditList.Cuts[0])
	}
}

// TestRemakeReplaceRejectsAnOrdinaryCut 确认普通剪辑工程替换时返回 only a remake draft。
func TestRemakeReplaceRejectsAnOrdinaryCut(t *testing.T) {
	e := testEngine(t)
	hash, err := e.PutBytes([]byte("picture"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CreateCutProject(CutProject{Title: "成片", SourceHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemakeReplace(p.ID, hash); err == nil || !strings.Contains(err.Error(), "only a remake draft") {
		t.Fatal(err)
	}
}
