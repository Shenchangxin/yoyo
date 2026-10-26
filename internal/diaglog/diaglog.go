// Package diaglog 写本进程的 JSON 日志、崩溃文本和支持包。
// 单文件滚动阈值默认是 32 << 20 字节。备份个数默认 14。备份按修改时间保留 14 个 24 小时，当前正在写的日志文件不按天数删除。
// worker 默认不镜像到控制台，避免写进给 JSON-RPC 用的 stdout。
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

// Process 是写日志的进程种类，用来选择文件名和控制台。
type Process string

const (
	// ProcessGUI 是桌面进程，值为 gui。FileName 对它返回 yoyo.log。
	ProcessGUI Process = "gui"
	// ProcessWorker 是工作进程，值为 worker。FileName 对它返回 worker.log。
	ProcessWorker Process = "worker"
	// ProcessCLI 是命令行进程，值为 cli。FileName 对它返回 cli.log。
	ProcessCLI Process = "cli"
)

const (
	// EnvLogDir 是日志目录环境变量 YOYO_LOG_DIR。非空时 DefaultLogDir 直接返回它。
	EnvLogDir = "YOYO_LOG_DIR"
	// EnvDebug 是调试分类环境变量 YOYO_DEBUG。Open 在 DebugCats 为空时用 ParseDebug 读它。
	EnvDebug = "YOYO_DEBUG"
	// EnvLogLevel 是级别环境变量 YOYO_LOG_LEVEL。Open 在 Level 为 0 时用 parseLevel 读它。
	EnvLogLevel = "YOYO_LOG_LEVEL"
	// EnvProcess 是进程种类环境变量 YOYO_PROCESS。DetectProcess 在 YOYO_WORKER 不是 1 时读它。
	EnvProcess = "YOYO_PROCESS"
	// EnvHome 是主目录环境变量 YOYO_HOME。DefaultLogDir 在参数和 YOYO_LOG_DIR 都空时读它。
	EnvHome = "YOYO_HOME"
	// EnvWorker 是工作进程环境变量 YOYO_WORKER。值为 1 时 DetectProcess 返回 worker。
	EnvWorker = "YOYO_WORKER"
	// EnvLogStderr 是控制台开关环境变量 YOYO_LOG_STDERR。值为 0、false、off 或 no 时不用 stderr。
	EnvLogStderr = "YOYO_LOG_STDERR"
)

const (
	// DefaultMaxSize 是单文件滚动大小的默认值，值为 32 << 20。MaxSize 小于等于 0 时用它。
	DefaultMaxSize = 32 << 20
	// DefaultFiles 是备份个数上限的默认值，值为 14。序号大于它的备份会被删掉。
	DefaultFiles = 14
	// DefaultAgeDays 是备份保留天数的默认值，值为 14。比较的是修改时间，早于现在减去这么多个 24 小时就删除。
	DefaultAgeDays = 14
)

// Options 是 Open 的参数。零值字段在 Open 里补上默认。
type Options struct {
	Dir        string
	Process    Process
	Version    string
	Level      slog.Level
	DebugCats  []string
	MaxSize    int64
	MaxFiles   int
	MaxAgeDays int
	// Console 镜像人可读的行。nil 时 GUI 和 CLI 用 stderr，worker 不用控制台，因为 stdout 留给 JSON-RPC。设成 io.Discard 则关闭。
	Console io.Writer
}

// Logger 是当前进程的滚动 JSON 日志，并记住调试分类和级别。
type Logger struct {
	mu            sync.Mutex
	opt           Options
	rot           *Rotator
	handler       slog.Handler
	slogger       *slog.Logger
	level         *slog.LevelVar
	cats          atomic.Value // 小写分类名到是否启用，类型是 map[string]bool
	stopRetention chan struct{}
	closeOnce     sync.Once
}

// def 是进程里的默认 Logger。Open 会把它换掉。
var def atomic.Pointer[Logger]

// DetectProcess 判断当前进程种类。YOYO_WORKER 为 1 时返回 worker。
// 否则读 YOYO_PROCESS，去掉空白并转成小写，只接受 gui、worker、cli。其余返回空字符串。
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

// DefaultLogDir 决定日志目录。YOYO_LOG_DIR 非空时用它。
// 否则用 homeRoot，homeRoot 为空再读 YOYO_HOME。仍为空则用用户主目录下的 .yoyo。最后接上 logs。
// 用户主目录也取不到时，结果是相对路径 logs。
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

// FileName 按进程返回日志文件名。worker 是 worker.log，cli 是 cli.log，其余含 gui 和空字符串都是 yoyo.log。
// p 为 ProcessWorker 时返回 worker.log，为 ProcessCLI 时返回 cli.log，其余返回 yoyo.log。
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

// Open 打开进程日志并设成 slog 和标准 log 的默认输出。
// 空目录用 DefaultLogDir。空进程先 DetectProcess，仍空则用 gui。空版本用 version.Version。
// MaxSize、MaxFiles、MaxAgeDays 小于等于 0 时分别用 DefaultMaxSize、DefaultFiles、DefaultAgeDays。
// Level 为 0 时用 parseLevel 读 YOYO_LOG_LEVEL。DebugCats 为空时用 ParseDebug 读 YOYO_DEBUG。
// 会创建 crash 和 mcp 子目录，权限 0o755。文件日志是 JSON。有控制台时再加文本处理器。
// 标准 log 的标志设为 0。有控制台时标准 log 同时写滚动文件和脱敏后的控制台，否则只写滚动文件。
// 返回前启动按小时调用 Cleanup 的循环，并立刻 Cleanup 一次。
// opt 的 Dir 为空时用 DefaultLogDir。Process 为空时先 DetectProcess，仍空则用 ProcessGUI。Version 为空时用 version.Version。MaxSize、MaxFiles、MaxAgeDays 小于等于 0 时换成对应默认。Level 为 0 时用 parseLevel 读环境变量。DebugCats 长度为 0 时用 ParseDebug 读环境变量。
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

// consoleWriter 决定人可读日志写到哪里。Console 已设置时，io.Discard 返回 nil，否则用它。
// Console 为 nil 且进程是 worker 时返回 nil。YOYO_LOG_STDERR 为 0、false、off、no 时返回 nil。其余返回 stderr。
// opt 的 Console 非 nil 时，等于 io.Discard 则返回 nil，否则返回它。Console 为 nil 且 Process 是 ProcessWorker 时返回 nil。
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

// redactWriter 在写入前脱敏，返回给调用方的字节数仍是原始长度。
type redactWriter struct{ w io.Writer }

// Write 把脱敏后的字节写进底层。底层失败时返回 0 和该错误。成功时返回 len(p)，即使脱敏后变短。
func (r redactWriter) Write(p []byte) (int, error) {
	b := RedactBytes(p)
	if _, err := r.w.Write(b); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Boot 在需要时打开进程日志，并写一行 component 为 boot 的 INFO。
// process 为空时先 DetectProcess，仍空则用 gui。默认 Logger 已经是同一进程时直接返回，不再写 boot。
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

// retentionLoop 每小时调用一次 Cleanup，直到 stopRetention 被关闭。l 或通道为 nil 时直接返回。
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

// SetDefault 把 l 存成进程默认日志。传入 nil 会清掉默认值。
func SetDefault(l *Logger) {
	def.Store(l)
}

// Default 返回进程默认日志。还没有 Open 时返回 nil。
func Default() *Logger {
	return def.Load()
}

// Apply 更新已打开日志的级别、调试分类和选项里的版本字符串。l 为 nil 时直接返回。
// Level 为 0 时不改级别。DebugCats 为 nil 时不改分类，空切片会清掉分类。版本字符串为空时不改。
// 这里只改 opt.Version，不改 Open 时已经挂到 slog 上的 version 属性。
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

// setCats 把分类收成小写集合。空字符串丢掉。结果存进 cats。
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

// debugOK 判断该分类的调试行能否写出。l 为 nil 时返回假。
// 当前级别小于等于 Debug 时全部放行。分类表里有 * 时放行。否则看小写后的 cat 是否在表里。
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

// Logger 返回底层 slog。l 为 nil 时返回 slog.Default。
func (l *Logger) Logger() *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	return l.slogger
}

// Handler 返回带上下文的处理器。l 为 nil 时返回 slog.Default 的处理器。
func (l *Logger) Handler() slog.Handler {
	if l == nil {
		return slog.Default().Handler()
	}
	return l.handler
}

// WailsLogger 返回带 component=wails 的 slog。l 为 nil 时用 slog.Default 再加这个属性。
func (l *Logger) WailsLogger() *slog.Logger {
	if l == nil {
		return slog.Default().With("component", "wails")
	}
	return l.slogger.With("component", "wails")
}

// Dir 返回日志目录。l 为 nil 时返回空字符串。
func (l *Logger) Dir() string {
	if l == nil {
		return ""
	}
	return l.opt.Dir
}

// Path 返回当前日志文件路径。l 或滚动器为 nil 时返回空字符串。
func (l *Logger) Path() string {
	if l == nil || l.rot == nil {
		return ""
	}
	return l.rot.Path()
}

// Process 返回打开时的进程种类。l 为 nil 时返回空字符串。
func (l *Logger) Process() Process {
	if l == nil {
		return ""
	}
	return l.opt.Process
}

// MCPStderrPath 返回 mcp 子目录里 name.stderr.log 的路径。name 经 sanitizeName。l 为 nil 时返回空字符串。
func (l *Logger) MCPStderrPath(name string) string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "mcp", sanitizeName(name)+".stderr.log")
}

// RendererPath 返回目录里的 renderer.jsonl。l 为 nil 时返回空字符串。
func (l *Logger) RendererPath() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "renderer.jsonl")
}

// CrashDir 返回目录里的 crash。l 为 nil 时返回空字符串。
func (l *Logger) CrashDir() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.opt.Dir, "crash")
}

// Close 关掉按小时清理的循环，并关闭滚动文件。重复调用只关一次通道。l 为 nil 时返回 nil。
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

// Writer 返回滚动文件，供外部把字节写进同一条日志。l 为 nil 时返回 io.Discard。
func (l *Logger) Writer() io.Writer {
	if l == nil {
		return io.Discard
	}
	return l.rot
}

// Info 用默认日志写一条 INFO。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.Info。
// args 在 Default 非空时传给 slogger.Info。
func Info(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Info(msg, args...)
	}
}

// Warn 用默认日志写一条 WARN。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.Warn。
// args 在 Default 非空时传给 slogger.Warn。
func Warn(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Warn(msg, args...)
	}
}

// Error 用默认日志写一条 ERROR。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.Error。
// args 在 Default 非空时传给 slogger.Error。
func Error(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Error(msg, args...)
	}
}

// Debug 用默认日志写一条 DEBUG。还没有默认日志时丢掉。是否落盘还看级别和分类。
// msg 在 Default 非空时传给 slogger.Debug。
// args 在 Default 非空时传给 slogger.Debug。
func Debug(msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.Debug(msg, args...)
	}
}

// InfoContext 用默认日志和 ctx 写一条 INFO。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.InfoContext。
// args 在 Default 非空时传给 slogger.InfoContext。
func InfoContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.InfoContext(ctx, msg, args...)
	}
}

// WarnContext 用默认日志和 ctx 写一条 WARN。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.WarnContext。
// args 在 Default 非空时传给 slogger.WarnContext。
func WarnContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.WarnContext(ctx, msg, args...)
	}
}

// ErrorContext 用默认日志和 ctx 写一条 ERROR。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.ErrorContext。
// args 在 Default 非空时传给 slogger.ErrorContext。
func ErrorContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.ErrorContext(ctx, msg, args...)
	}
}

// DebugContext 用默认日志和 ctx 写一条 DEBUG。还没有默认日志时丢掉。
// msg 在 Default 非空时传给 slogger.DebugContext。
// args 在 Default 非空时传给 slogger.DebugContext。
func DebugContext(ctx context.Context, msg string, args ...any) {
	if l := Default(); l != nil {
		l.slogger.DebugContext(ctx, msg, args...)
	}
}

// parseLevel 把文本收成 slog 级别。debug 是 Debug，warn 和 warning 是 Warn，error 是 Error，其余含空字符串都是 Info。
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

// ParseDebug 按逗号拆调试分类，去掉空白并转成小写。空段丢掉。整段空白时返回 nil。
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

// sanitizeName 把名字收成文件名片段。空白变成 unknown。
// 只保留字母、数字、下划线和连字符，其他字符换成下划线。收完仍为空时返回 unknown。
// name 去掉首尾空白，为空时返回 unknown。只保留字母、数字、下划线和连字符，其他字符换成下划线。结果仍为空时返回 unknown。
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
