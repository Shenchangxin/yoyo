package diaglog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

func (l *Logger) WriteCrash(msg string, rec any) string {
	if l == nil {
		return ""
	}
	dir := l.CrashDir()
	_ = os.MkdirAll(dir, 0o755)
	name := "panic-" + time.Now().UTC().Format("20060102T150405Z") + ".txt"
	path := filepath.Join(dir, name)
	stack := debug.Stack()
	body := fmt.Sprintf("ts=%s\nprocess=%s\npid=%d\nmsg=%s\nrec=%v\n\n%s\n",
		time.Now().UTC().Format(time.RFC3339Nano), l.opt.Process, os.Getpid(), Redact(msg), rec, stack)
	_ = os.WriteFile(path, []byte(body), 0o600)
	l.slogger.Error("panic", "component", "runtime", "err", Redact(msg), "crash_path", path)
	return path
}

func WriteCrash(msg string, rec any) string {
	if l := Default(); l != nil {
		return l.WriteCrash(msg, rec)
	}
	return ""
}
