// Package procgroup 启动子进程，并能一次结束它和它派生的整棵进程树。
// Windows 上用 Job Object（yoyo 退出时句柄关闭，整棵树随之结束）；Unix 上用独立进程组。
// acp 客户端和 buddy 的登录窗口调用它；它不改 internal/isolation。
package procgroup

import (
	"os/exec"
	"sync"
)

// Options 是启动方式。
type Options struct {
	// Hide 在 Windows 上不弹出控制台窗口。
	Hide bool
	// NewConsole 在 Windows 上给它开一个新控制台，登录窗口用。优先于 Hide。
	NewConsole bool
}

// Proc 是一个已经启动、归在同一组里的进程。
type Proc struct {
	cmd     *exec.Cmd
	group   *group
	done    chan struct{}
	waitErr error
	once    sync.Once
}

// Start 按 opts 启动 cmd，并把它放进一个能整体结束的组。cmd 不能已经启动过。
func Start(cmd *exec.Cmd, opts Options) (*Proc, error) {
	prepare(cmd, opts)
	g, err := startGroup(cmd)
	if err != nil {
		return nil, err
	}
	p := &Proc{cmd: cmd, group: g, done: make(chan struct{})}
	go func() {
		p.waitErr = cmd.Wait()
		p.group.exited()
		close(p.done)
	}()
	return p, nil
}

// Pid 返回主进程编号。
func (p *Proc) Pid() int {
	if p == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Wait 阻塞到主进程退出，返回 exec.Cmd.Wait 的结果。可以多次调用。
func (p *Proc) Wait() error {
	<-p.done
	return p.waitErr
}

// Done 在主进程退出后关闭。
func (p *Proc) Done() <-chan struct{} { return p.done }

// ExitCode 返回主进程的退出码。还没退出时为 -1。
func (p *Proc) ExitCode() int {
	select {
	case <-p.done:
	default:
		return -1
	}
	if p.cmd.ProcessState == nil {
		return -1
	}
	return p.cmd.ProcessState.ExitCode()
}

// Kill 结束整棵进程树。重复调用只生效一次。
func (p *Proc) Kill() error {
	if p == nil {
		return nil
	}
	var err error
	p.once.Do(func() {
		err = p.group.kill(p.done)
	})
	return err
}
