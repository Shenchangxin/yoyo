//go:build !windows

// procgroup_unix.go 拥有 Unix 上的进程组：子进程用 Setpgid 自成一组，Kill 先给整组发 SIGTERM，两秒后再发 SIGKILL。
// Linux 上另加 Pdeathsig（见 pdeath_linux.go），yoyo 意外退出时子进程也收到信号。

package procgroup

import (
	"os/exec"
	"syscall"
	"time"
)

// group 记住进程组号。
type group struct {
	pgid int
}

// prepare 让子进程自成一组。opts 的窗口选项在 Unix 上没有意义。
func prepare(cmd *exec.Cmd, _ Options) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	setDeathSignal(cmd.SysProcAttr)
}

// startGroup 启动 cmd。组号就是主进程编号。
func startGroup(cmd *exec.Cmd) (*group, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &group{pgid: cmd.Process.Pid}, nil
}

// kill 先给整组发 SIGTERM；主进程两秒内没退出就发 SIGKILL。组里还有别的进程时同样收到。
func (g *group) kill(done <-chan struct{}) error {
	if g.pgid <= 0 {
		return nil
	}
	err := syscall.Kill(-g.pgid, syscall.SIGTERM)
	if err == syscall.ESRCH {
		return nil
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if e := syscall.Kill(-g.pgid, syscall.SIGKILL); e != nil && e != syscall.ESRCH {
		return e
	}
	return nil
}

// exited 在 Unix 上什么都不做；留下的组员由 Kill 处理。
func (g *group) exited() {}
