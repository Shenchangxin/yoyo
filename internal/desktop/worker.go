package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
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
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
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
			return nil, err
		}
	case <-time.After(8 * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("worker health timeout")
	}
	return &Service{RPC: cli, cmd: cmd}, nil
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
