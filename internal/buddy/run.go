// run.go 拥有默认的命令执行：隐藏窗口、放进进程组，ctx 取消时整组结束，输出合并后返回（只留最后 1 MB）。
// 取版本、跑 npm、校验 node 都经 Options.Run；测试注入假的。

package buddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/procgroup"
)

// maxRunOutput 是一条命令留下的输出上限。
const maxRunOutput = 1 << 20

// defaultRun 跑一条命令直到结束。退出码非 0 时返回错误，输出照样返回。
func defaultRun(ctx context.Context, c Cmd) ([]byte, error) {
	opts := c.Launch.Options()
	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	acp.SetCmdLine(cmd, opts.CmdLine)
	out := &tailBuffer{max: maxRunOutput}
	var w io.Writer = out
	if c.Output != nil {
		w = io.MultiWriter(out, c.Output)
	}
	sw := &syncWriter{w: w}
	cmd.Stdout = sw
	cmd.Stderr = sw
	p, err := procgroup.Start(cmd, procgroup.Options{Hide: true})
	if err != nil {
		return nil, err
	}
	select {
	case <-p.Done():
	case <-ctx.Done():
		_ = p.Kill()
		<-p.Done()
		return out.Bytes(), ctx.Err()
	}
	if code := p.ExitCode(); code != 0 {
		return out.Bytes(), fmt.Errorf("exit status %d", code)
	}
	return out.Bytes(), nil
}

// syncWriter 让 stdout 和 stderr 两个拷贝协程安全地写同一处。
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write 持锁写入。
func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// tailBuffer 只留最后 max 个字节。
type tailBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

// Write 追加并裁掉超出的开头。
func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf.Write(p)
	if extra := t.buf.Len() - t.max; extra > 0 {
		t.buf.Next(extra)
	}
	return len(p), nil
}

// Bytes 返回留下的内容的副本。
func (t *tailBuffer) Bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return bytes.Clone(t.buf.Bytes())
}
