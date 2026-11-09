// probe.go 拥有探测：轻探只做 initialize 然后关掉；全探再在 buddies/.probe/<id> 里建一个会话、记下模型后关掉。
// 超时 20 秒，最多同时 2 个。结果写进状态记录：要求登录记成 needs_auth，缺必填密钥记成 needs_config，其余失败记成出错或刚断开。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// probeTimeout 是一次探测的上限。
const probeTimeout = 20 * time.Second

// Probe 立刻对 id 做一次全探，返回探测后的视图。
func (m *Manager) Probe(ctx context.Context, id string) (View, error) {
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	m.mu.Unlock()
	if !ok {
		return View{}, ErrNotFound
	}
	if e.Builtin {
		return m.view(id), nil
	}
	err := m.probe(ctx, id, true)
	return m.view(id), err
}

// probe 探测一项并更新记录。没装时什么都不做。
func (m *Manager) probe(ctx context.Context, id string, full bool) error {
	settings := m.opts.Settings()
	item := settings.Item(id)
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	d := m.det[id]
	busy := m.jobs[id] != nil || m.removing[id]
	if !busy {
		m.probing[id]++
	}
	m.mu.Unlock()
	if busy {
		return nil
	}
	defer func() {
		m.mu.Lock()
		if m.probing[id]--; m.probing[id] <= 0 {
			delete(m.probing, id)
		}
		m.mu.Unlock()
	}()
	if !ok || e.Builtin || (d.Managed == nil && d.SystemPath == "") {
		return nil
	}
	if missing := m.missingRequired(e, item); len(missing) > 0 {
		m.update(id, func(r *record) {
			r.State = StateNeedsConfig
			r.Detail = strings.Join(missing, ", ")
			r.CheckedAt = m.opts.Now()
		})
		return nil
	}
	l, secrets, err := m.launchFor(e, item, d)
	if err != nil {
		m.fail(id, full, err.Error(), secrets)
		return err
	}
	select {
	case m.probeSem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-m.probeSem }()
	if full {
		m.update(id, func(r *record) {
			if r.State == "" || r.State == StateNotInstalled || r.State == StateError || r.State == StateOffline || r.State == StateNeedsConfig {
				r.State = StateProbing
			}
		})
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	dir := m.probeDir(id)
	opts := l.Options()
	opts.Dir = dir
	opts.Version = m.opts.Version
	opts.InitTimeout = probeTimeout
	c, err := m.opts.Dial(ctx, opts)
	if err != nil {
		detail := err.Error()
		m.writeProbeLog(id, detail, secrets)
		m.fail(id, full, detail, secrets)
		return err
	}
	defer func() {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelClose()
		done := make(chan struct{})
		go func() { _ = c.Close(); close(done) }()
		select {
		case <-done:
		case <-closeCtx.Done():
		}
		if tail := c.Stderr().Tail(4000); tail != "" {
			m.writeProbeLog(id, tail, secrets)
		}
	}()
	init := c.Init()
	now := m.opts.Now()
	if !full {
		var state State
		m.mu.Lock()
		if r := m.recs[id]; r != nil {
			state = r.State
		}
		m.mu.Unlock()
		if state == StateOnline || state == StateRecentlyLost {
			m.update(id, func(r *record) {
				r.State = StateOnline
				r.Detail = ""
				r.LostAt = time.Time{}
				r.CheckedAt = now
				r.Probe = mergeInfo(r.Probe, infoFrom(init, nil, false, now))
			})
			return nil
		}
		full = true
	}
	sess, err := c.NewSession(ctx, acp.NewSessionRequest{Cwd: dir, McpServers: []acp.McpServer{}})
	if err != nil {
		if errors.Is(err, acp.ErrAuthRequired) {
			m.update(id, func(r *record) {
				r.State = StateNeedsAuth
				r.Detail = ""
				r.CheckedAt = now
				r.Probe = mergeInfo(r.Probe, infoFrom(init, nil, false, now))
			})
			return nil
		}
		detail := err.Error()
		if tail := c.Stderr().Tail(600); tail != "" {
			detail += ": " + tail
		}
		m.fail(id, true, detail, secrets)
		return err
	}
	if bool(init.AgentCapabilities.SessionCapabilities.Close) {
		_ = c.CloseSession(ctx, sess.SessionID)
	}
	m.update(id, func(r *record) {
		r.State = StateOnline
		r.Detail = ""
		r.LostAt = time.Time{}
		r.CheckedAt = now
		r.Probe = infoFrom(init, &sess, true, now)
	})
	return nil
}

// mergeInfo 用新的握手信息覆盖旧的，旧的模型列表留着（轻探拿不到模型）。
func mergeInfo(old, fresh *probeInfo) *probeInfo {
	if old != nil && len(fresh.Models) == 0 {
		fresh.Models = old.Models
		fresh.CurrentModel = old.CurrentModel
		fresh.Options = old.Options
		fresh.Full = old.Full
	}
	return fresh
}

// fail 记一次探测失败。轻探失败走刚断开的计时；全探失败记成出错。
func (m *Manager) fail(id string, full bool, detail string, secrets []string) {
	detail = shortDetail(redactText(detail, secrets...))
	now := m.opts.Now()
	m.update(id, func(r *record) {
		if full {
			r.State = StateError
			r.Detail = detail
			r.CheckedAt = now
			return
		}
		lostLocked(r, now, detail)
	})
}

// shortDetail 把详情收成一行，最多 300 个字符。
func shortDetail(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 300 {
		return string(r[:300]) + "…"
	}
	return s
}

// writeProbeLog 把探测时的 stderr 尾巴追加到 buddies/<id>/logs/probe.log，已打码，超过 256 KB 时只留后一半。
func (m *Manager) writeProbeLog(id, text string, secrets []string) {
	if m.opts.Root == "" || strings.TrimSpace(text) == "" {
		return
	}
	m.logMu.Lock()
	defer m.logMu.Unlock()
	dir := filepath.Join(m.installDir(id), "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, "probe.log")
	line := fmt.Sprintf("[%s] %s\n", m.opts.Now().UTC().Format(time.RFC3339), redactText(text, secrets...))
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 256<<10 {
		_ = os.WriteFile(path, raw[len(raw)/2:], 0o600)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

// LogTail 返回一项的安装日志和探测日志的最后 lines 行，已打码。
func (m *Manager) LogTail(id string, lines int) (string, error) {
	m.mu.Lock()
	_, ok := m.entryLocked(id)
	m.mu.Unlock()
	if !ok {
		return "", ErrNotFound
	}
	if lines <= 0 || lines > 2000 {
		lines = 200
	}
	var parts []string
	for _, name := range []string{"install.log", "probe.log"} {
		raw, err := os.ReadFile(filepath.Join(m.installDir(id), "logs", name))
		if err != nil {
			continue
		}
		all := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
		if len(all) > lines {
			all = all[len(all)-lines:]
		}
		parts = append(parts, "== "+name+" ==\n"+strings.Join(all, "\n"))
	}
	return redactText(strings.Join(parts, "\n\n")), nil
}

// probeUnknown 对已装但还没探过、或上次停在检查中的项做一次全探。
func (m *Manager) probeUnknown(ctx context.Context) {
	m.mu.Lock()
	var ids []string
	for _, e := range m.entries {
		if e.Builtin || !m.installedLocked(e) {
			continue
		}
		r := m.recs[e.ID]
		if r == nil || r.Probe == nil || r.State == "" || r.State == StateProbing || r.State == StateNotInstalled || r.State == StateInstalling {
			ids = append(ids, e.ID)
		}
	}
	m.mu.Unlock()
	slices.Sort(ids)
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		id := id
		m.goSafe(func() { _ = m.probe(ctx, id, true) })
	}
}
