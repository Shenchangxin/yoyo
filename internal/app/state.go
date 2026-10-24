package app

import "strings"

// StateChange is a mutation of workstation state. Transcript tokens stay on
// Hub; this bus is for everything a second window must repaint.
type StateChange struct {
	Epoch  int64  `json:"epoch"`
	Topic  string `json:"topic"`
	Object string `json:"object"`
}

// Touch records a mutation. Delivery never blocks the writer.
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

// StateEpoch is the latest mutation counter.
func (a *App) StateEpoch() int64 {
	if a == nil {
		return 0
	}
	a.stateMu.Lock()
	defer a.stateMu.Unlock()
	return a.stateEpoch
}

// SubscribeState returns a stream of mutations. The caller must unsub.
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

// NoteRPC marks a successful JSON-RPC as a state change when it writes.
func (a *App) NoteRPC(method string) {
	topic, ok := RPCTopic(method)
	if !ok || a == nil {
		return
	}
	a.Touch(topic, "")
}

// RPCTopic reports the repaint topic for a mutating method.
// Reads return "", false so polls do not wake every window.
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
