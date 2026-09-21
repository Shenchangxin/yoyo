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
)

// SpawnWorker starts this executable as a JSON-RPC stdio worker so the GUI
// process never runs the agent loop.
func SpawnWorker(exe, home, evals string) (*Service, error) {
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return nil, err
		}
	}
	cmd := exec.Command(exe)
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
		_ = cmd.Process.Kill()
		if logf != nil {
			_ = logf.Close()
		}
		return nil, fmt.Errorf("worker health timeout")
	}
	return &Service{RPC: cli, cmd: cmd, workerLog: logf}, nil
}

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

func OpenIsolated(home, evals string) (*Service, *app.App, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	svc, err := SpawnWorker(exe, home, evals)
	if err == nil {
		return svc, nil, nil
	}
	core, err2 := app.Open(home, evals)
	if err2 != nil {
		return nil, nil, fmt.Errorf("worker: %v; local: %w", err, err2)
	}
	return NewService(core), core, nil
}
