package diaglog

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONSchemaAndRedact(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(Options{Dir: dir, Process: ProcessCLI, Version: "test", Level: slog.LevelInfo, Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Logger().Info("boot", "component", "boot", "api_key", "sk-live-secretvalue", "authorization", "Bearer sk-live-secretvalue")
	recs, err := l.Tail(10, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("expected a log line")
	}
	raw := recs[len(recs)-1].Raw
	if strings.Contains(raw, "sk-live") || strings.Contains(raw, "secretvalue") {
		t.Fatalf("secret leaked: %s", raw)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"ts", "level", "msg", "process", "pid", "goos", "version"} {
		if m[k] == nil || m[k] == "" {
			t.Fatalf("missing %s in %s", k, raw)
		}
	}
	if m["component"] != "boot" {
		t.Fatalf("component=%v", m["component"])
	}
}

func TestWorkerStdoutUnpolluted(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	dir := t.TempDir()
	l, err := Open(Options{Dir: dir, Process: ProcessWorker, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	l.Logger().Info("worker boot", "component", "boot")
	_ = l.Close()
	_ = w.Close()
	b, _ := io.ReadAll(r)
	if len(bytes.TrimSpace(b)) != 0 {
		t.Fatalf("stdout polluted: %q", b)
	}
	body, err := os.ReadFile(filepath.Join(dir, "worker.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "worker boot") {
		t.Fatalf("missing log: %s", body)
	}
}

func TestRedactPatterns(t *testing.T) {
	in := "Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz token=x OPENAI_API_KEY=abc -----BEGIN KEY-----\nabc\n-----END KEY-----"
	out := Redact(in)
	if strings.Contains(out, "sk-abcdefgh") || strings.Contains(out, "OPENAI_API_KEY=abc") || strings.Contains(out, "BEGIN KEY-----\nabc") {
		t.Fatalf("%s", out)
	}
}

func TestRotator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "yoyo.log")
	rot, err := OpenRotator(path, RotatorOpts{MaxSize: 64, MaxFiles: 3, MaxAgeDays: 14})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := rot.Write([]byte("abcdefghijklmnopqrstuvwxyz0123456789\n")); err != nil {
			t.Fatal(err)
		}
	}
	_ = rot.Close()
	ents, _ := os.ReadDir(dir)
	var gz int
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".gz") {
			gz++
		}
	}
	if gz == 0 {
		t.Fatalf("expected rotated gz, got %v", names(ents))
	}
}

func TestDebugCategoryFilter(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(Options{Dir: dir, Process: ProcessCLI, Level: slog.LevelInfo, DebugCats: []string{"rpc"}, Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Logger().Debug("hidden", "component", "mcp", "cat", "mcp")
	l.Logger().InfoContext(With(nil, Fields{Cat: "rpc", Component: "rpc"}), "rpc ok")
	l.Logger().DebugContext(With(nil, Fields{Cat: "rpc", Component: "rpc"}), "rpc debug")
	body, _ := os.ReadFile(l.Path())
	s := string(body)
	if strings.Contains(s, "hidden") {
		t.Fatalf("mcp debug leaked: %s", s)
	}
	if !strings.Contains(s, "rpc debug") {
		t.Fatalf("rpc debug missing: %s", s)
	}
}

func TestBundleOmitsVaultAndHeldOut(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(Options{Dir: dir, Process: ProcessCLI, Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	l.Logger().Info("ok", "component", "boot")
	_ = l.Close()
	_ = os.WriteFile(filepath.Join(dir, "vault.json"), []byte(`{"default":"sk-live-supersecret"}`), 0o600)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	_ = os.WriteFile(cfg, []byte("search_key: super-search-secret\nmodel: x\n"), 0o644)
	dest := filepath.Join(t.TempDir(), "bundle.zip")
	path, err := WriteBundle(BundleOpts{
		Dir:        dir,
		Dest:       dest,
		Doctor:     map[string]any{"ok": true, "api_key": "sk-live-supersecret"},
		ConfigPath: cfg,
		Version:    "test",
		HeldOutIDs: []string{"so-01-uniform"},
		Isolation:  map[string]any{"kind": "none"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ZipContains(path, "sk-live-supersecret") || ZipContains(path, "super-search-secret") {
		t.Fatal("secrets in bundle")
	}
	if ZipContains(path, "held-out body") {
		t.Fatal("held-out body in bundle")
	}
}

func names(ents []os.DirEntry) []string {
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

func TestConsoleMirrorsGUI(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()
	l, err := Open(Options{Dir: dir, Process: ProcessGUI, Version: "test", Console: &buf})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Logger().Info("boot", "component", "boot")
	s := buf.String()
	if !strings.Contains(s, "boot") {
		t.Fatalf("console silent: %q", s)
	}
	if strings.Contains(s, "sk-") {
		t.Fatalf("secret on console: %s", s)
	}
	body, _ := os.ReadFile(l.Path())
	if !strings.Contains(string(body), `"msg":"boot"`) {
		t.Fatalf("file missing json: %s", body)
	}
}
