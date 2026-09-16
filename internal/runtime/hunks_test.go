package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDiffHunks(t *testing.T) {
	diff := `diff --git a/foo.txt b/foo.txt
index 111..222 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 keep
-old
+new
 still
@@ -10,2 +11,2 @@
-a
+b
`
	hs := ParseDiffHunks(diff)
	if len(hs) != 2 {
		t.Fatalf("hunks=%d %+v", len(hs), hs)
	}
	if hs[0].File != "foo.txt" || hs[0].ID != "foo.txt:1" {
		t.Fatalf("%+v", hs[0])
	}
	if !strings.Contains(hs[0].Body, "-old") || !strings.Contains(hs[0].Body, "+new") {
		t.Fatalf("body %q", hs[0].Body)
	}
	patch, err := PatchFromHunks(diff, []string{"foo.txt:1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(patch, "-a") {
		t.Fatalf("second hunk leaked: %s", patch)
	}
	if !strings.Contains(patch, "+new") {
		t.Fatalf("missing selected hunk: %s", patch)
	}
}

func TestApplyHunksGit(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=yoyo", "GIT_AUTHOR_EMAIL=yoyo@local", "GIT_COMMITTER_NAME=yoyo", "GIT_COMMITTER_EMAIL=yoyo@local")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "yoyo@local")
	run("config", "user.name", "yoyo")
	p := filepath.Join(dir, "foo.txt")
	if err := os.WriteFile(p, []byte("keep\nold\nstill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "foo.txt")
	run("commit", "-m", "seed")
	if err := os.WriteFile(p, []byte("keep\nnew\nstill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diff, err := exec.Command("git", "-C", dir, "diff", "--no-color").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	hs := ParseDiffHunks(string(diff))
	if len(hs) != 1 {
		t.Fatalf("hunks=%d %s", len(hs), diff)
	}
	// restore then apply the hunk
	if err := os.WriteFile(p, []byte("keep\nold\nstill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyHunks(dir, string(diff), []string{hs[0].ID}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if strings.ReplaceAll(string(got), "\r\n", "\n") != "keep\nnew\nstill\n" {
		t.Fatalf("%q", got)
	}
	if err := ReverseApplyHunks(dir, string(diff), []string{hs[0].ID}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(p)
	if strings.ReplaceAll(string(got), "\r\n", "\n") != "keep\nold\nstill\n" {
		t.Fatalf("undo %q", got)
	}
}
