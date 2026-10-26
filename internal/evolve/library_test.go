// 锁住经验三字段缺一不落盘、领域目录互相隔离、held 编号和评分词不入库，以及只有 confounded 时不可迁移。

package evolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHasExperienceRequiresThreeFields 要求缺 Evidence 时为假，三字段都在时为真。
func TestHasExperienceRequiresThreeFields(t *testing.T) {
	if HasExperience(Experience{Cue: "c", Lesson: "l"}) {
		t.Fatal("missing evidence")
	}
	if !HasExperience(Experience{Cue: "c", Lesson: "l", Evidence: "e"}) {
		t.Fatal("complete")
	}
}

// TestDomainSlugIsolatesPaths 要求只取最后一段，路径穿越变成不含点点的名字，空路径为 libraryUnknown。
func TestDomainSlugIsolatesPaths(t *testing.T) {
	if DomainSlug(filepath.Join("tmp", "alpha")) != "alpha" {
		t.Fatalf("%s", DomainSlug(filepath.Join("tmp", "alpha")))
	}
	if DomainSlug(filepath.Join("tmp", "a", "b")) != "b" {
		t.Fatalf("%s", DomainSlug(filepath.Join("tmp", "a", "b")))
	}
	if strings.Contains(DomainSlug(`..\..\etc`), "..") {
		t.Fatal("dotdot")
	}
	if DomainSlug("") != libraryUnknown {
		t.Fatal("empty")
	}
}

// TestLibraryNOENTRYDoesNotWrite 要求缺 Evidence 时 Add 返回假且不建 INDEX.md。
func TestLibraryNOENTRYDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	lib := &Library{Root: root, Domain: "alpha"}
	ok, err := lib.Add("t1", "hash1", "why", Experience{Cue: "c", Lesson: "l"}, nil)
	if err != nil || ok {
		t.Fatalf("no_entry %v %v", ok, err)
	}
	if _, err := os.Stat(lib.IndexPath()); !os.IsNotExist(err) {
		t.Fatal("index")
	}
}

// TestLibraryAddAndDomainIsolation 要求 alpha 能读到自己的索引，beta 的 Context 为空且没有 INDEX.md。
func TestLibraryAddAndDomainIsolation(t *testing.T) {
	root := t.TempDir()
	a := &Library{Root: root, Domain: "alpha"}
	b := &Library{Root: root, Domain: "beta"}
	ok, err := a.Add("task-a", "hash-a", "contrast", Experience{
		Cue:      "When a named artifact is missing after the first pass",
		Lesson:   "Add a verification instruction that writes the artifact before stopping",
		Evidence: "inv1 delivered relative to base",
	}, nil)
	if err != nil || !ok {
		t.Fatalf("add %v %v", ok, err)
	}
	if a.Context() == "" || !strings.Contains(a.Context(), "named artifact") {
		t.Fatalf("index %s", a.Context())
	}
	if b.Context() != "" {
		t.Fatalf("beta leaked %s", b.Context())
	}
	if _, err := os.Stat(filepath.Join(b.Dir(), "INDEX.md")); !os.IsNotExist(err) {
		t.Fatal("beta dir")
	}
}

// TestLibraryHeldOutIsNOENTRY 要求经验正文含 held 编号时 Add 返回假。
func TestLibraryHeldOutIsNOENTRY(t *testing.T) {
	root := t.TempDir()
	lib := &Library{Root: root, Domain: "alpha"}
	ok, err := lib.Add("t1", "h", "why", Experience{
		Cue:      "when write-answer appears",
		Lesson:   "do not copy write-answer",
		Evidence: "row 1 cites write-answer",
	}, map[string]bool{"write-answer": true})
	if err != nil || ok {
		t.Fatalf("held-out must no_entry: %v %v", ok, err)
	}
}

// TestLibraryViewListsCue 要求 View 列出该领域的一条，且 Cue 非空。
func TestLibraryViewListsCue(t *testing.T) {
	root := t.TempDir()
	lib := &Library{Root: root, Domain: "alpha"}
	if _, err := lib.Add("task-a", "hash-a", "why", Experience{
		Cue:      "When a named artifact is missing after the first pass",
		Lesson:   "Add a verification instruction that writes the artifact before stopping",
		Evidence: "inv1 delivered relative to base",
	}, nil); err != nil {
		t.Fatal(err)
	}
	view := lib.View(nil)
	if view.Domain != "alpha" || len(view.Entries) != 1 || view.Entries[0].Cue == "" {
		t.Fatalf("%+v", view)
	}
}

// TestExperienceConfoundedIsNotTransferable 要求只有 confounded 时不可迁移，有 helped 时可以。
func TestExperienceConfoundedIsNotTransferable(t *testing.T) {
	exp := Experience{
		Cue: "c", Lesson: "l", Evidence: "e",
		Verdicts: []EditVerdict{{Component: "instruction", Verdict: "confounded"}},
	}
	if experienceTransferable(exp) {
		t.Fatal("confounded")
	}
	exp.Verdicts = []EditVerdict{{Component: "instruction", Verdict: "helped"}}
	if !experienceTransferable(exp) {
		t.Fatal("helped")
	}
}

// TestLibraryGraderTokenIsNOENTRY 要求正文含评分禁用词时 Add 返回假。
func TestLibraryGraderTokenIsNOENTRY(t *testing.T) {
	root := t.TempDir()
	lib := &Library{Root: root, Domain: "alpha"}
	ok, err := lib.Add("t1", "h", "why", Experience{
		Cue:      "when the grader pass_rate rises",
		Lesson:   "copy the rubric",
		Evidence: "held-out improved",
	}, nil)
	if err != nil || ok {
		t.Fatalf("grader: %v %v", ok, err)
	}
}
