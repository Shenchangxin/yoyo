// uninstall.go 拥有卸载：托管安装直接删目录；系统 npm 安装跑 npm rm -g；其余给出手动卸载说明。
// 有工单正在用它、或正在安装时拒绝。删目录前等正在跑的探测停下，卸载中不再起新的探测。设置和 vault 里的密钥留着，重新装回来还能用。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Uninstall 卸载 id，返回卸载后的视图。
func (m *Manager) Uninstall(ctx context.Context, id string) (View, error) {
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	d := m.det[id]
	busy := m.jobs[id] != nil
	npm := m.sysNPM
	m.mu.Unlock()
	if !ok || e.Builtin {
		return View{}, ErrNotFound
	}
	if busy {
		return View{}, errors.New(m.say("正在安装，先取消再卸载", "installing now; cancel it first"))
	}
	if n := m.working()[id]; n > 0 {
		return View{}, fmt.Errorf(m.say("它正在做 %d 个任务，停掉后再卸载", "it is working on %d tasks; stop them first"), n)
	}
	switch {
	case e.Custom:
		if m.opts.SaveSettings != nil {
			err := m.opts.SaveSettings(func(s *Settings) {
				s.Custom = slices.DeleteFunc(s.Custom, func(c CustomAgent) bool { return c.ID == id })
				delete(s.Items, id)
			})
			if err != nil {
				return View{}, err
			}
		}
		m.mu.Lock()
		m.rebuildLocked()
		delete(m.det, id)
		delete(m.recs, id)
		m.mu.Unlock()
		m.saveRecords()
		return View{ID: id, State: StateNotInstalled}, nil
	case d.Managed != nil && (m.opts.Settings().Item(id).Mode != "system" || d.SystemPath == ""):
		if err := m.beginRemove(ctx, id); err != nil {
			return View{}, err
		}
		defer m.endRemove(id)
		if err := removeAllRetry(ctx, m.installDir(id)); err != nil {
			return View{}, fmt.Errorf("%s: %w", m.say("有文件正被占用，关掉用它的程序后再试", "some files are in use; close whatever is using them and try again"), err)
		}
		_ = removeAllRetry(ctx, filepath.Join(m.opts.Root, ".probe", id))
	case d.SystemPath != "":
		pkg, _ := systemPackage(e)
		if pkg == "" || npm == "" {
			return View{}, fmt.Errorf(m.say("这是装在系统里的版本（%s），用当初安装它的方式卸载", "this is a system install (%s); uninstall it the way it was installed"), d.SystemPath)
		}
		if _, err := m.opts.Run(ctx, Cmd{Launch: m.systemNPM(npm, "rm", "-g", pkg)}); err != nil {
			return View{}, fmt.Errorf("npm rm -g %s: %w", pkg, err)
		}
		m.paths.reset()
		if again := m.detectFast(e); again.SystemPath != "" {
			return View{}, fmt.Errorf(m.say("npm 卸载后仍能找到 %s，它可能不是用 npm 装的", "%s is still on PATH; it was probably not installed with npm"), again.SystemPath)
		}
	default:
		return View{}, ErrNotInstalled
	}
	m.detectOne(ctx, e)
	m.update(id, func(r *record) {
		*r = record{State: StateNotInstalled}
	})
	return m.view(id), nil
}

// removeWait 是卸载等正在跑的探测停下的上限：探测超时加上关进程的时间。
const removeWait = probeTimeout + 5*time.Second

// beginRemove 标记 id 正在卸载，并等它正在跑的探测结束（探测会占着 agent 的文件和目录）。
func (m *Manager) beginRemove(ctx context.Context, id string) error {
	m.mu.Lock()
	if m.removing[id] {
		m.mu.Unlock()
		return errors.New(m.say("已经在卸载", "already uninstalling"))
	}
	m.removing[id] = true
	m.mu.Unlock()
	deadline := time.Now().Add(removeWait)
	for {
		m.mu.Lock()
		n := m.probing[id]
		m.mu.Unlock()
		if n == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			m.endRemove(id)
			return errors.New(m.say("它还在检测中，稍后再卸载", "still checking it; try again in a moment"))
		}
		select {
		case <-ctx.Done():
			m.endRemove(id)
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// endRemove 清掉卸载标记。
func (m *Manager) endRemove(id string) {
	m.mu.Lock()
	delete(m.removing, id)
	m.mu.Unlock()
}

// removeAllRetry 删掉目录。Windows 上刚写完的文件常被杀毒软件或刚退出的进程占着一小会儿，失败时每半秒重试，最多 10 次。
func removeAllRetry(ctx context.Context, dir string) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = os.RemoveAll(dir); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(500 * time.Millisecond):
		}
	}
	return err
}
