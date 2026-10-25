package video

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

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

func TestRemakeDraftRejectsAnEmptyReference(t *testing.T) {
	e := testEngine(t)
	if _, err := e.RemakeDraft("  ", "", ""); err == nil || !strings.Contains(err.Error(), "reference hash required") {
		t.Fatal(err)
	}
}

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
