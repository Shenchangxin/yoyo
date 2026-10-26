// 本文件把 panic 写成 crash 目录里的文本，并在进程日志里记一条 error。
// 文件名是 panic- 加 UTC 时间 20060102T150405Z 再加 .txt，权限 0o600。

package diaglog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// WriteCrash 把 msg、rec 和当前堆栈写到崩溃目录。l 为 nil 时返回空字符串。
// 目录权限 0o755。正文里的 msg 经 Redact。同时写一条 component 为 runtime 的 error，带上 crash_path。返回写出的路径。
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

// WriteCrash 用默认日志写崩溃文件。还没有默认日志时返回空字符串。
func WriteCrash(msg string, rec any) string {
	if l := Default(); l != nil {
		return l.WriteCrash(msg, rec)
	}
	return ""
}
