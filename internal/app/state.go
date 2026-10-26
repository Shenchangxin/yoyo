// 工作台状态变更总线。转录 token 留在 Hub，这条总线只通知第二扇窗口重绘。投递不阻塞写入方。

package app

import "strings"

// StateChange 是一次工作台状态变更。Epoch 是递增计数，Topic 是重绘主题，Object 是相关对象 id。
type StateChange struct {
	Epoch  int64  `json:"epoch"`
	Topic  string `json:"topic"`
	Object string `json:"object"`
}

// Touch 记下一次变更并返回新的 Epoch。订阅通道满时丢掉这次投递，不阻塞写入方。App 为空返回 0。
// topic 写入 StateChange.Topic。
// object 写入 StateChange.Object。
func (a *App) Touch(topic, object string) int64 {
	if a == nil {
		return 0
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	a.stateEpoch++
	if a.stateSubs == nil {
		return a.stateEpoch
	}
	change := StateChange{Epoch: a.stateEpoch, Topic: topic, Object: object}
	for ch := range a.stateSubs {
		select {
		case ch <- change:
		default:
		}
	}
	return a.stateEpoch
}

// StateEpoch 返回当前变更计数。App 为空返回 0。
func (a *App) StateEpoch() int64 {
	if a == nil {
		return 0
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.stateEpoch
}

// SubscribeState 返回变更流和取消函数。通道容量 64。App 为空时返回已关闭的通道。调用方必须调用取消函数。
func (a *App) SubscribeState() (<-chan StateChange, func()) {
	ch := make(chan StateChange, 64)
	if a == nil {
		close(ch)
		return ch, func() {}
	}
	a.stateMu.Lock()
	if a.stateSubs == nil {
		a.stateSubs = map[chan StateChange]struct{}{}
	}
	a.stateSubs[ch] = struct{}{}
	a.stateMu.Unlock()
	return ch, func() {
		a.stateMu.Lock()
		delete(a.stateSubs, ch)
		a.stateMu.Unlock()
	}
}

// NoteRPC 在 JSON-RPC 方法会改状态时 Touch 一次。读方法和未知方法不产生变更。
// method 传给 RPCTopic。
func (a *App) NoteRPC(method string) {
	topic, ok := RPCTopic(method)
	if !ok || a == nil {
		return
	}
	a.Touch(topic, "")
}

// RPCTopic 返回会改状态的方法对应的重绘主题。读方法返回空串和 false，避免轮询唤醒每一扇窗口。
// method 先 TrimSpace，空串或 isRPCRead 为真时返回空串和 false，否则按前缀选主题。
func RPCTopic(method string) (string, bool) {
	method = strings.TrimSpace(method)
	if method == "" || isRPCRead(method) {
		return "", false
	}
	switch {
	case strings.HasPrefix(method, "thread."), strings.HasPrefix(method, "turn."), strings.HasPrefix(method, "session."), strings.HasPrefix(method, "approvals."), strings.HasPrefix(method, "approval."):
		return "session", true
	case strings.HasPrefix(method, "config."), strings.HasPrefix(method, "provider."), method == "key.set", method == "app.pause.set", strings.HasPrefix(method, "skins."):
		return "config", true
	case strings.HasPrefix(method, "packs."), strings.HasPrefix(method, "skills."):
		return "pack", true
	case strings.HasPrefix(method, "pages."), strings.HasPrefix(method, "profiles."):
		return "page", true
	case strings.HasPrefix(method, "video."), strings.HasPrefix(method, "drama."), strings.HasPrefix(method, "media."), strings.HasPrefix(method, "canvas."):
		return "video", true
	case strings.HasPrefix(method, "harness."), strings.HasPrefix(method, "evolve."), strings.HasPrefix(method, "eval."), strings.HasPrefix(method, "playbook."):
		return "harness", true
	case strings.HasPrefix(method, "idol."):
		return "idol", true
	case strings.HasPrefix(method, "schedule."), strings.HasPrefix(method, "personal."), strings.HasPrefix(method, "inbox."), strings.HasPrefix(method, "memory."), strings.HasPrefix(method, "projects."):
		return "schedule", true
	case strings.HasPrefix(method, "workspace."):
		return "workspace", true
	case strings.HasPrefix(method, "remote."):
		return "remote", true
	case strings.HasPrefix(method, "connections."), strings.HasPrefix(method, "connectors."), strings.HasPrefix(method, "mcp."):
		return "config", true
	default:
		return "app", true
	}
}

// isRPCRead 把 health、about、doctor 等固定读方法，以及以 .list、.get、.status、.catalog、.tail、.search、.diff、.preview、.lineage 结尾的方法当成读。
// method 与固定读方法比较，或看是否包含 .list，或以 .get、.status、.catalog、.tail、.search、.diff、.preview、.lineage 结尾。
func isRPCRead(method string) bool {
	switch method {
	case "health", "about", "doctor", "models.catalog", "key.status", "turn.running", "turn.running_ids", "turn.running_status",
		"isolation.report", "runtime.pwsh.status", "update.status", "phone.status", "archive.list", "plugins.list",
		"video.status", "video.modes", "video.templates", "video.styles", "video.styles.all":
		return true
	}
	if strings.Contains(method, ".list") || strings.HasSuffix(method, ".get") || strings.HasSuffix(method, ".status") ||
		strings.HasSuffix(method, ".catalog") || strings.HasSuffix(method, ".tail") || strings.HasSuffix(method, ".search") ||
		strings.HasSuffix(method, ".diff") || strings.HasSuffix(method, ".preview") || strings.HasSuffix(method, ".lineage") {
		return true
	}
	return false
}
