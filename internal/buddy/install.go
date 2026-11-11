// install.go 拥有安装任务：同一项只允许一个、全局最多 2 个并行；进度、安装日志（已打码）和取消。
// 成功后重新检测再全探；失败或取消时删掉半成品，没装过的回到未安装。具体装法在 install_npm.go、install_binary.go、install_system.go 和 uv.go。

package buddy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// installTimeout 是一次安装的总上限。
const installTimeout = 15 * time.Minute

// job 是一个进行中的安装。
type job struct {
	mode     string
	cancel   context.CancelFunc
	progress Progress
}

// Install 开始安装 id。mode 取 managed（默认）或 system。立刻返回安装中的视图，进度走事件。
func (m *Manager) Install(id, mode string) (View, error) {
	if mode == "" {
		mode = "managed"
	}
	if mode != "managed" && mode != "system" {
		return View{}, fmt.Errorf("%s: %s", m.say("安装方式只能是 managed 或 system", "mode must be managed or system"), mode)
	}
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	if !ok || e.Builtin {
		m.mu.Unlock()
		return View{}, ErrNotFound
	}
	if e.Custom {
		m.mu.Unlock()
		return View{}, errors.New(m.say("自定义 agent 不需要安装", "custom agents are not installed by yoyo"))
	}
	if m.jobs[id] != nil {
		m.mu.Unlock()
		return View{}, errors.New(m.say("已经在装了", "already installing"))
	}
	if mode == "managed" && !installable(e, m.plat) {
		m.mu.Unlock()
		return View{}, errors.New(m.say("此系统没有可用的托管安装", "no managed install for this system"))
	}
	if mode == "system" {
		if sys := m.systemInstallView(e); !sys.Available {
			m.mu.Unlock()
			return View{}, errors.New(sys.Reason)
		}
	}
	ctx, cancel := context.WithTimeout(m.ctx, installTimeout)
	m.jobs[id] = &job{mode: mode, cancel: cancel, progress: Progress{Phase: "queued"}}
	r := m.recordLocked(id)
	r.Detail = ""
	m.mu.Unlock()
	m.publish(id)
	m.goSafe(func() {
		defer cancel()
		m.runInstall(ctx, e, mode)
	})
	return m.view(id), nil
}

// Cancel 取消 id 正在进行的安装。
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	j := m.jobs[id]
	m.mu.Unlock()
	if j == nil {
		return errors.New(m.say("没有在装", "not installing"))
	}
	j.cancel()
	return nil
}

// runInstall 排队拿到名额后按 mode 安装，最后收尾。
func (m *Manager) runInstall(ctx context.Context, e Entry, mode string) {
	log := m.openInstallLog(e.ID, mode)
	defer log.Close()
	select {
	case m.installSem <- struct{}{}:
	case <-ctx.Done():
		m.finishInstall(e, ctx.Err(), log)
		return
	}
	defer func() { <-m.installSem }()
	prog := func(p Progress) { m.setProgress(e.ID, p) }
	var err error
	if mode == "system" {
		err = m.installSystem(ctx, e, prog, log)
	} else {
		err = m.installManaged(ctx, e, prog, log)
	}
	m.finishInstall(e, err, log)
}

// installManaged 按来源优先级装托管版：本平台有二进制就用二进制，其次 npx（托管 Node），最后 uvx（托管 uv）。
func (m *Manager) installManaged(ctx context.Context, e Entry, prog func(Progress), log io.Writer) error {
	prev, _ := m.readInstall(e.ID)
	var rec *InstallRecord
	var err error
	switch {
	case hasTarget(e, m.plat):
		t, _, _ := pickTarget(e.Dist.Binary, m.plat)
		rec, err = m.installBinary(ctx, e, t, prog, log)
	case e.Dist.NPX != nil:
		rec, err = m.installNPM(ctx, e, *e.Dist.NPX, prog, log)
	case e.Dist.UVX != nil:
		rec, err = m.installUVX(ctx, e, *e.Dist.UVX, prog, log)
	default:
		err = errors.New(m.say("此系统没有可用的托管安装", "no managed install for this system"))
	}
	if err != nil {
		return err
	}
	prog(Progress{Phase: "verify", Percent: 95})
	rec.SizeBytes = dirSize(rec.Dir)
	rec.InstalledAt = m.opts.Now().UTC()
	if err := m.writeInstall(e.ID, rec); err != nil {
		return err
	}
	if prev != nil && prev.Dir != "" && filepath.Clean(prev.Dir) != filepath.Clean(rec.Dir) && within(m.installDir(e.ID), prev.Dir) {
		_ = os.RemoveAll(prev.Dir)
	}
	return nil
}

// hasTarget 判断本平台有没有二进制归档。
func hasTarget(e Entry, p Platform) bool {
	_, _, ok := pickTarget(e.Dist.Binary, p)
	return ok
}

// finishInstall 收尾：写完日志最后一行就关掉它，再重新检测，然后在同一把锁里去掉任务并记下结果，界面不会看到「没在装也没结果」的中间态；
// 成功就全探，失败或取消记下原因。日志先关，否则紧接着的卸载在 Windows 上删不掉它。
func (m *Manager) finishInstall(e Entry, err error, log *installLog) {
	detail := ""
	if err != nil {
		detail = m.say("安装失败：", "Install failed: ") + shortDetail(redactText(err.Error()))
		if errors.Is(err, context.Canceled) {
			detail = m.say("已取消安装", "Install cancelled")
		}
		fmt.Fprintf(log, "%s\n", detail)
	} else {
		fmt.Fprintln(log, "done")
	}
	_ = log.Close()
	m.detectOne(m.ctx, e)
	m.mu.Lock()
	delete(m.jobs, e.ID)
	r := m.recordLocked(e.ID)
	if err != nil {
		r.Detail = detail
		if r.State == StateInstalling || r.State == "" {
			r.State = StateNotInstalled
		}
	} else {
		r.State = StateProbing
		r.Detail = ""
	}
	m.mu.Unlock()
	m.saveRecords()
	m.publish(e.ID)
	if err == nil {
		m.adoptYoyoChat(e.ID)
		_ = m.probe(m.ctx, e.ID, true)
	}
}

// setProgress 更新安装进度并发事件。
func (m *Manager) setProgress(id string, p Progress) {
	m.mu.Lock()
	j := m.jobs[id]
	if j != nil {
		if p.Line == "" && p.Phase == j.progress.Phase {
			p.Line = j.progress.Line
		}
		j.progress = p
	}
	m.mu.Unlock()
	if j != nil {
		m.publish(id)
	}
}

// installLog 是安装日志：逐行打码后写进 buddies/<id>/logs/install.log，最后一行同步到进度上。
type installLog struct {
	mu      sync.Mutex
	f       *os.File
	partial bytes.Buffer
	onLine  func(string)
}

// openInstallLog 打开安装日志，写下开头一行。打不开时退回到只更新进度。
func (m *Manager) openInstallLog(id, mode string) *installLog {
	l := &installLog{onLine: func(line string) {
		m.mu.Lock()
		j := m.jobs[id]
		if j != nil {
			j.progress.Line = line
		}
		m.mu.Unlock()
		if j != nil {
			m.publish(id)
		}
	}}
	if m.opts.Root != "" {
		dir := filepath.Join(m.installDir(id), "logs")
		if os.MkdirAll(dir, 0o755) == nil {
			l.f, _ = os.OpenFile(filepath.Join(dir, "install.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		}
	}
	fmt.Fprintf(l, "[%s] install %s (%s)\n", m.opts.Now().UTC().Format(time.RFC3339), id, mode)
	return l
}

// Write 按行打码后落盘，并把最后一个非空行交给进度。
func (l *installLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.partial.Write(p)
	for {
		line, err := l.partial.ReadString('\n')
		if err != nil {
			l.partial.Reset()
			l.partial.WriteString(line)
			break
		}
		l.emit(strings.TrimRight(line, "\r\n"))
	}
	return len(p), nil
}

// emit 写一行。调用方持有 l.mu。
func (l *installLog) emit(line string) {
	line = redactText(line)
	if l.f != nil {
		_, _ = l.f.WriteString(line + "\n")
	}
	if strings.TrimSpace(line) != "" && l.onLine != nil {
		l.onLine(shortDetail(line))
	}
}

// Close 写出最后半行并关掉文件。可以重复调用；关掉之后的写入只更新进度。
func (l *installLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.partial.Len() > 0 {
		l.emit(l.partial.String())
		l.partial.Reset()
	}
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// dirSize 统计目录占用的字节数。
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// within 判断 path 在不在 root 里面。
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}
