package diaglog

import (
	"context"
	"io"
	stdlog "log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/version"
)

type Process string

const (
	ProcessGUI    Process = "gui"
	ProcessWorker Process = "worker"
	ProcessCLI    Process = "cli"
)

const (
	EnvLogDir   = "YOYO_LOG_DIR"
	EnvDebug    = "YOYO_DEBUG"
	EnvLogLevel = "YOYO_LOG_LEVEL"
	EnvProcess  = "YOYO_PROCESS"
	EnvHome      = "YOYO_HOME"
	EnvWorker    = "YOYO_WORKER"
	EnvLogStderr = "YOYO_LOG_STDERR"
)

const (
	DefaultMaxSize = 32 << 20
	DefaultFiles   = 14
	DefaultAgeDays = 14
)

type Options struct {
	Dir        string
	Process    Process
	Version    string
	Level      slog.Level
	DebugCats  []string
	MaxSize    int64
	MaxFiles   int
	MaxAgeDays int
	// Console mirrors human-readable lines. nil uses stderr for GUI/CLI
	// and discard for worker (stdout is JSON-RPC). io.Discard disables.
	Console io.Writer
}

type Logger struct {
	mu            sync.Mutex
	opt           Options
	rot           *Rotator
	handler       slog.Handler
	slogger       *slog.Logger
	level         *slog.LevelVar
	cats          atomic.Value // map[string]bool
	stopRetention chan struct{}
	closeOnce     sync.Once
}

var def atomic.Pointer[Logger]

func DetectProcess() Process {
	if os.Getenv(EnvWorker) == "1" {
		return ProcessWorker
	}
	switch Process(strings.ToLower(strings.TrimSpace(os.Getenv(EnvProcess)))) {
	case ProcessGUI, ProcessWorker, ProcessCLI:
		return Process(strings.ToLower(strings.TrimSpace(os.Getenv(EnvProcess))))
	default:
		return ""
	}
}

func DefaultLogDir(homeRoot string) string {
	if v := strings.TrimSpace(os.Getenv(EnvLogDir)); v != "" {
		return v
	}
	if homeRoot == "" {
		homeRoot = strings.TrimSpace(os.Getenv(EnvHome))
	}
	if homeRoot == "" {
		h, err := os.UserHomeDir()
		if err == nil {
			homeRoot = filepath.Join(h, ".yoyo")
		}
	}
	return filepath.Join(homeRoot, "logs")
}

func FileName(p Process) string {
	switch p {
	case ProcessWorker:
		return "worker.log"
	case ProcessCLI:
		return "cli.log"
	default:
		return "yoyo.log"
	}
}

func Open(opt Options) (*Logger, error) {
	if opt.Dir == "" {
		opt.Dir = DefaultLogDir("")
	}
	if opt.Process == "" {
		opt.Process = DetectProcess()
	}
	if opt.Process == "" {
		opt.Process = ProcessGUI
	}
	if opt.Version == "" {
		opt.Version = version.Version
	}
	if opt.MaxSize <= 0 {
		opt.MaxSize = DefaultMaxSize
	}
	if opt.MaxFiles <= 0 {
		opt.MaxFiles = DefaultFiles
	}
	if opt.MaxAgeDays <= 0 {
		opt.MaxAgeDays = DefaultAgeDays
	}
	if opt.Level == 0 {
		opt.Level = parseLevel(os.Getenv(EnvLogLevel))
	}
	if len(opt.DebugCats) == 0 {
		opt.DebugCats = ParseDebug(os.Getenv(EnvDebug))
	}
	if err := os.MkdirAll(filepath.Join(opt.Dir, "crash"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(opt.Dir, "mcp"), 0o755); err != nil {
		return nil, err
	}
	rot, err := OpenRotator(filepath.Join(opt.Dir, FileName(opt.Process)), RotatorOpts{
		MaxSize:    opt.MaxSize,
		MaxFiles:   opt.MaxFiles,
		MaxAgeDays: opt.MaxAgeDays,
	})
	if err != nil {
		return nil, err
	}
	level := &slog.LevelVar{}
	level.Set(opt.Level)
	l := &Logger{opt: opt, rot: rot, level: level}
	l.setCats(opt.DebugCats)
	fileH := slog.NewJSONHandler(rot, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	})
	next := slog.Handler(fileH)
	cons := consoleWriter(opt)
	if cons != nil {
		textH := slog.NewTextHandler(cons, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: replaceConsoleAttr,
		})
		next = slog.NewMultiHandler(fileH, textH)
	}
	h := &ctxHandler{next: next, log: l}
	l.handler = h
	l.slogger = slog.New(h).With(
		"process", string(opt.Process),
		"pid", os.Getpid(),
		"goos", runtime.GOOS,
		"goarch", runtime.GOARCH,
		"version", opt.Version,
	)
	SetDefault(l)
	slog.SetDefault(l.slogger)
	stdlog.SetFlags(0)
	if cons != nil {
		stdlog.SetOutput(io.MultiWriter(rot, redactWriter{cons}))
	} else {
		stdlog.SetOutput(rot)
	}
	l.stopRetention = make(chan struct{})
	go l.retentionLoop()
	_ = rot.Cleanup()
	return l, nil
}

func consoleWriter(opt Options) io.Writer {
	if opt.Console != nil {
		if opt.Console == io.Discard {
			return nil
		}
		return opt.Console
	}
	if opt.Process == ProcessWorker {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvLogStderr))) {
	case "0", "false", "off", "no":
		return nil
	}
	return os.Stderr
}

type redactWriter struct{ w io.Writer }

func (r redactWriter) Write(p []byte) (int, error) {
	b := RedactBytes(p)
	if _, err := r.w.Write(b); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Boot opens the process logger if needed and writes a component=boot INFO line.
func Boot(process Process) (*Logger, error) {
	if process == "" {
		process = DetectProcess()
	}
	if process == "" {
		process = ProcessGUI
	}
	if def := Default(); def != nil && def.Process() == process {
		return def, nil
	}
	l, err := Open(Options{Process: process, Version: version.Version})
	if err != nil {
		return nil, err
	}
	Info("boot", "component", "boot")
	return l, nil
}

func (l *Logger) retentionLoop() {
	if l == nil || l.stopRetention == nil {
		return
	}
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-l.stopRetention:
			return
		case <-t.C:
			if l.rot != nil {
				_ = l.rot.Cleanup()
			}
		}
	}
}

func SetDefault(l *Logger) {
	def.Store(l)
}

func Default() *Logger {
	return def.Load()
}

func (l *Logger) Apply(opt Options) {
	if l == nil {
		return
	}
	if opt.Level != 0 {
		l.level.Set(opt.Level)
	}
	if opt.DebugCats != nil {
		l.setCats(opt.DebugCats)
	}
	l.mu.Lock()
	if opt.Version != "" {
		l.opt.Version = opt.Version
	}
	l.mu.Unlock()
}

func (l *Logger) setCats(cats []string) {
	m := map[string]bool{}
	for _, c := range cats {
		c = strings.ToLower(strings.TrimSpace(c))
		if c != "" {
			m[c] = true
		}
	}
	l.cats.Store(m)
}

func (l *Logger) debugOK(cat string) bool {
	if l == nil {
		return false
	}
	if l.level.Level() <= slog.LevelDebug {
		return true
	}
	m, _ := l.cats.Load().(map[string]bool)
	if m == nil {
		return false
	}
	if m["*"] {
		return true
	}
	return m[strings.ToLower(strings.TrimSpace(cat))]
}

func (l *Logger) Logger() *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	return l.slogger
}

func (l *Logger) Handler() slog.Handler {
	if l == nil {
		return slog.Default().Handler()
	}
	return l.handler
}

func (l *Logger) WailsLogger() *slog.Logger {
	if l == nil {
		return slog.Default().With("component", "wails")
	}
	return l.slogger.With("component", "wails")
}

func (l *Logger) Dir() string {
	if l == nil {
		return ""
	}
	return l.opt.Dir
}

func (l *Logger) Path() string {
	if l == nil || l.rot == nil {
		return ""
	}
	return l.rot.Path()
}

func (l *Logger) Process() Process {
	if l == nil {
		return ""
	}
	return l.opt.Process
}

func (l *Logger) MCPStderrPath(name string) string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "mcp", sanitizeName(name)+".stderr.log")
}

func (l *Logger) RendererPath() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "renderer.jsonl")
}

func (l *Logger) CrashDir() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "crash")
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		if l.stopRetention != nil {
			close(l.stopRetention)
		}
	})
	if l.rot == nil {
		return nil
	}
	return l.rot.Close()
}

func (l *Logger) Writer() io.Writer {
	if l == nil {
		return io.Discard
	}
	return l.rot
}

func Info(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Error(msg, args...)
	}
}

func Debug(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Debug(msg, args...)
	}
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.InfoContext(ctx, msg, args...)
	}
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.WarnContext(ctx, msg, args...)
	}
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.ErrorContext(ctx, msg, args...)
	}
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.DebugContext(ctx, msg, args...)
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func ParseDebug(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	if s == "" {
		return "unknown"
	}
	return s
}
