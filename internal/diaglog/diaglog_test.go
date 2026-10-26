// 本文件锁住 JSON 字段和脱敏、worker 不写 stdout、滚动产生 gz、调试分类、支持包不含密钥，以及 GUI 控制台镜像。
// 不连接外部服务。由 go test 调用。

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

// TestJSONSchemaAndRedact 要求日志行含 ts、level、msg、process、pid、goos、version 和 component=boot，并且不含 sk-live 或 secretvalue。
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

// TestWorkerStdoutUnpolluted 要求 worker 的 Open 不往 stdout 写字节，并且 worker.log 里有 worker boot。
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

// TestRedactPatterns 要求 Bearer 密钥、OPENAI_API_KEY 的值和 PEM 正文都被抹掉。
func TestRedactPatterns(t *testing.T) {
	in := "Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz token=x OPENAI_API_KEY=abc -----BEGIN KEY-----\nabc\n-----END KEY-----"
	out := Redact(in)
	if strings.Contains(out, "sk-abcdefgh") || strings.Contains(out, "OPENAI_API_KEY=abc") || strings.Contains(out, "BEGIN KEY-----\nabc") {
		t.Fatalf("%s", out)
	}
}

// TestRotator 用 64 字节的 MaxSize 和 3 个备份写 20 行，要求目录里出现 .gz。
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

// TestDebugCategoryFilter 要求只放行 cat 为 rpc 的调试行，mcp 的 hidden 不落盘。
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

// TestBundleOmitsVaultAndHeldOut 要求支持包里没有 sk-live-supersecret、super-search-secret 和 held-out body。
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

// names 收集目录项名字，供失败信息打印。
func names(ents []os.DirEntry) []string {
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

// TestConsoleMirrorsGUI 要求 GUI 的控制台里有 boot，不含 sk-，并且文件里有 JSON 的 msg boot。
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
