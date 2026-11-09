// fs 的测试：外部 agent 经 ACP 读写文件时的目录限制、只读目录和 CheckWrite。

package acpworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// fsDirs 是一张工单的目录：工单目录、可写的共享目录、只读的前置目录，以及它们的上一级。
type fsDirs struct{ base, dir, shared, before string }

func fsWorker(t *testing.T, check func(context.Context, string) error) (*Worker, fsDirs) {
	t.Helper()
	base := t.TempDir()
	d := fsDirs{base: base, dir: filepath.Join(base, "work"), shared: filepath.Join(base, "shared"), before: filepath.Join(base, "before")}
	for _, p := range []string{d.dir, d.shared, d.before} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	spec := superagent.WorkerSpec{TicketID: "T-1", Dir: d.dir, Roots: []string{d.shared}, ReadRoots: []string{d.before}}
	return open(Config{Spec: spec, CheckWrite: check}, newPool(maxLive)), d
}

func TestFSWritesStayInTicketDirs(t *testing.T) {
	w, d := fsWorker(t, nil)
	ctx := context.Background()
	for _, p := range []string{filepath.Join(d.dir, "a.txt"), filepath.Join(d.dir, "sub", "b.txt"), filepath.Join(d.shared, "NOTES.md")} {
		if err := w.writeFile(ctx, acp.WriteTextFileRequest{Path: p, Content: "hi"}); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		if body, err := os.ReadFile(p); err != nil || string(body) != "hi" {
			t.Fatalf("read back %s = %q, %v", p, body, err)
		}
	}
	for _, p := range []string{
		filepath.Join(d.before, "x.txt"),
		filepath.Join(d.base, "outside.txt"),
		filepath.Join(d.dir, "..", "escape.txt"),
		"relative.txt",
		"",
	} {
		if err := w.writeFile(ctx, acp.WriteTextFileRequest{Path: p, Content: "no"}); err == nil {
			t.Fatalf("write %q should be refused", p)
		}
	}
	for _, p := range []string{filepath.Join(d.base, "outside.txt"), filepath.Join(d.base, "escape.txt"), filepath.Join(d.before, "x.txt")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s exists after a refused write", p)
		}
	}
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".yoyo-write-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestFSReadsCoverReadRootsOnly(t *testing.T) {
	w, d := fsWorker(t, nil)
	ctx := context.Background()
	plan := filepath.Join(d.before, "plan.md")
	if err := os.WriteFile(plan, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.base, "secret.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: plan}); err != nil || got != "one\ntwo\nthree\n" {
		t.Fatalf("read plan = %q, %v", got, err)
	}
	line, limit := 2, 1
	if got, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: plan, Line: &line, Limit: &limit}); err != nil || got != "two\n" {
		t.Fatalf("read line 2 = %q, %v", got, err)
	}
	for _, p := range []string{filepath.Join(d.base, "secret.txt"), d.dir, "plan.md"} {
		if _, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: p}); err == nil {
			t.Fatalf("read %q should fail", p)
		}
	}
	big := filepath.Join(d.dir, "big.log")
	if err := os.WriteFile(big, make([]byte, maxReadBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.readFile(ctx, acp.ReadTextFileRequest{Path: big}); err == nil {
		t.Fatal("a file over 2 MB should not be read")
	}
}

func TestFSCheckWriteVetoes(t *testing.T) {
	denied := errors.New("broker said no")
	var asked []string
	w, d := fsWorker(t, func(_ context.Context, path string) error {
		asked = append(asked, path)
		return denied
	})
	p := filepath.Join(d.dir, "a.txt")
	if err := w.writeFile(context.Background(), acp.WriteTextFileRequest{Path: p, Content: "x"}); !errors.Is(err, denied) {
		t.Fatalf("write err = %v, want the broker's", err)
	}
	if len(asked) != 1 || asked[0] != p {
		t.Fatalf("CheckWrite saw %v", asked)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("file written despite the veto")
	}
}

func TestSliceLines(t *testing.T) {
	text := "a\nb\nc"
	n := func(v int) *int { return &v }
	for _, tc := range []struct {
		line, limit *int
		want        string
	}{
		{nil, nil, text},
		{n(2), nil, "b\nc"},
		{n(1), n(2), "a\nb\n"},
		{n(9), nil, ""},
		{nil, n(0), ""},
	} {
		if got := sliceLines(text, tc.line, tc.limit); got != tc.want {
			t.Fatalf("sliceLines(%v, %v) = %q, want %q", tc.line, tc.limit, got, tc.want)
		}
	}
}
