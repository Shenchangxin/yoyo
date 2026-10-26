package desktop

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// SpawnWorker 再启动 exe。exe 为空时用当前可执行文件。子进程环境在现有环境上追加 YOYO_WORKER=1，home 或 evals 非空时再追加 YOYO_HOME、YOYO_EVALS。
// stdin 和 stdout 用管道接上，stdout 是 JSON-RPC 协议。stderr 能打开日志就写入工作者日志，否则丢弃。osutil.NoConsole 不给子进程弹控制台。
// 启动后调用 health，最多等 8 秒。超时或失败会杀掉子进程、关掉日志并返回错误。成功则返回带 LineClient 的 Service，App 为空。
func SpawnWorker(exe, home, evals string) (*Service, error) {
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return nil, err
		}
	}
	cmd := exec.Command(exe)
	// 子进程是同一个可执行文件。YOYO_WORKER=1 让它不建窗口，stdout 只出 JSON-RPC。
	cmd.Env = append(os.Environ(), "YOYO_WORKER=1")
	if home != "" {
		cmd.Env = append(cmd.Env, "YOYO_HOME="+home)
	}
	if evals != "" {
		cmd.Env = append(cmd.Env, "YOYO_EVALS="+evals)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	osutil.NoConsole(cmd)
	logf := openWorkerLog(home)
	if logf != nil {
		cmd.Stderr = logf
	}
	if err := cmd.Start(); err != nil {
		if logf != nil {
			_ = logf.Close()
		}
		return nil, err
	}
	cli := api.NewLineClient(stdout, stdin)
	done := make(chan error, 1)
	go func() {
		_, err := cli.Call("health", nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			_ = cmd.Process.Kill()
			if logf != nil {
				_ = logf.Close()
			}
			return nil, err
		}
	case <-time.After(8 * time.Second):
		// 8 秒内 health 没回来就杀掉子进程。OpenIsolated 收到这个错误后改在本进程打开 App。这不是沙箱检查。
		_ = cmd.Process.Kill()
		if logf != nil {
			_ = logf.Close()
		}
		return nil, fmt.Errorf("worker health timeout")
	}
	return &Service{RPC: cli, cmd: cmd, workerLog: logf}, nil
}

// openWorkerLog 在诊断日志目录下追加打开工作者日志，权限 0600。目录或文件创建失败时返回 nil。
// home 传给 diaglog.DefaultLogDir。
func openWorkerLog(home string) *os.File {
	dir := diaglog.DefaultLogDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(dir, diaglog.FileName(diaglog.ProcessWorker)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	return f
}

// OpenIsolated 用当前可执行文件调用 SpawnWorker，从而再启动同一程序并带上 YOYO_WORKER=1。
// SpawnWorker 成功时返回该 Service，核心留在子进程，这里的 App 为 nil。失败则在本进程 app.Open 并 NewService，退回同进程。两步都失败时，错误同时带上工作者错误和本地错误。
// 这不是安全沙箱，工作者仍握有 vault 和工具。
// home 传给 SpawnWorker。失败时传给 app.Open。
// evals 传给 SpawnWorker。失败时传给 app.Open。
func OpenIsolated(home, evals string) (*Service, *app.App, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	svc, err := SpawnWorker(exe, home, evals)
	if err == nil {
		return svc, nil, nil
	}
	// 工作者没起来就在本进程打开 App。窗口和循环回到同一个进程。
	core, err2 := app.Open(home, evals)
	if err2 != nil {
		return nil, nil, fmt.Errorf("worker: %v; local: %w", err, err2)
	}
	return NewService(core), core, nil
}
