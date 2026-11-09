// health.go 拥有健康检查：启动 20 秒后开始，对已装且启用、没在服务工单的项每 10 分钟轻探一次；
// 每 30 秒把超过 5 分钟的「刚断开」落成离线。正在服务工单的项视为在线，不去打扰。

package buddy

import (
	"context"
	"time"
)

// healthLoop 跑到 ctx 结束。
func (m *Manager) healthLoop(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(m.opts.HealthDelay):
	}
	settle := time.NewTicker(30 * time.Second)
	defer settle.Stop()
	check := time.NewTimer(0)
	defer check.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-settle.C:
			m.Settle()
		case <-check.C:
			m.CheckHealth(ctx)
			check.Reset(m.opts.HealthEvery)
		}
	}
}

// Settle 把超过宽限的「刚断开」落成离线，变了的项发事件。
func (m *Manager) Settle() {
	now := m.opts.Now()
	var changed []string
	m.mu.Lock()
	for id, r := range m.recs {
		if settleLocked(r, now) {
			changed = append(changed, id)
		}
	}
	m.mu.Unlock()
	if len(changed) == 0 {
		return
	}
	m.saveRecords()
	for _, id := range changed {
		m.publish(id)
	}
}

// CheckHealth 轻探每个已装、启用、没在服务工单的项，最多同时 2 个（probeSem 控制）。
func (m *Manager) CheckHealth(ctx context.Context) {
	working := m.working()
	settings := m.opts.Settings()
	m.mu.Lock()
	var ids []string
	for _, e := range m.entries {
		if e.Builtin || !m.installedLocked(e) || settings.Item(e.ID).Disabled || working[e.ID] > 0 || m.jobs[e.ID] != nil {
			continue
		}
		ids = append(ids, e.ID)
	}
	m.mu.Unlock()
	done := make(chan struct{}, len(ids))
	for _, id := range ids {
		id := id
		m.goSafe(func() {
			_ = m.probe(ctx, id, false)
			done <- struct{}{}
		})
	}
	for range ids {
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
	}
}

// Refresh 在后台重新检测全部项并全探已装的；force 时先刷新 ACP 注册表。立刻返回，结果走事件。
func (m *Manager) Refresh(force bool) {
	m.Start()
	m.mu.Lock()
	m.kicked = true
	m.mu.Unlock()
	m.paths.reset()
	m.goSafe(func() {
		if force {
			_ = m.refreshRegistry(m.ctx, true)
		}
		m.detectAll(m.ctx)
		m.probeAll(m.ctx)
	})
}

// probeAll 对已装且启用的项各全探一次。
func (m *Manager) probeAll(ctx context.Context) {
	settings := m.opts.Settings()
	m.mu.Lock()
	var ids []string
	for _, e := range m.entries {
		if !e.Builtin && m.installedLocked(e) && !settings.Item(e.ID).Disabled {
			ids = append(ids, e.ID)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		id := id
		m.goSafe(func() { _ = m.probe(ctx, id, true) })
	}
}
