// state.go 拥有每个 agent 的状态记录：状态机迁移、探测结果，以及落到 buddies/state.json 的持久化。
// 怎么把记录推导成界面上的一项在 manager.go 的 view；健康检查的计时在 health.go。

package buddy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// lostGrace 是刚断开到算离线的时长。
const lostGrace = 5 * time.Minute

// record 是一个 agent 的持久状态。
type record struct {
	State     State      `json:"state,omitempty"`
	Detail    string     `json:"detail,omitempty"`
	CheckedAt time.Time  `json:"checkedAt,omitempty"`
	LostAt    time.Time  `json:"lostAt,omitempty"`
	Probe     *probeInfo `json:"probe,omitempty"`
	// SysPath、SysMod 和 SysVersion 缓存系统版本号，文件没变就不再跑 --version。
	SysPath    string    `json:"sysPath,omitempty"`
	SysMod     time.Time `json:"sysMod,omitempty"`
	SysVersion string    `json:"sysVersion,omitempty"`
}

// probeInfo 是最近一次成功握手看到的东西。
type probeInfo struct {
	At           time.Time             `json:"at"`
	Full         bool                  `json:"full,omitempty"`
	AgentName    string                `json:"agentName,omitempty"`
	AgentVersion string                `json:"agentVersion,omitempty"`
	Caps         acp.AgentCapabilities `json:"caps"`
	Auth         []acp.AuthMethod      `json:"auth,omitempty"`
	Models       []acp.ModelInfo       `json:"models,omitempty"`
	CurrentModel string                `json:"currentModel,omitempty"`
	// Options 是 session/new 给的可选设置。没有 models 时，模型列表从 category 为 model 的那一项里取。
	Options []acp.SessionConfigOption `json:"options,omitempty"`
}

// loadRecords 读 state.json；读不到或坏了就从空表开始。
func loadRecords(path string) map[string]*record {
	out := map[string]*record{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	if json.Unmarshal(raw, &out) != nil {
		return map[string]*record{}
	}
	for id, r := range out {
		if r == nil {
			delete(out, id)
		}
	}
	return out
}

// saveRecords 先写临时文件再改名，写到一半断电也不会留下半个 JSON。
func saveRecords(path string, recs map[string]*record) error {
	raw, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// recordLocked 取 id 的记录，没有时建一个。调用方持有 m.mu。
func (m *Manager) recordLocked(id string) *record {
	r := m.recs[id]
	if r == nil {
		r = &record{}
		m.recs[id] = r
	}
	return r
}

// settleLocked 把超过宽限的「刚断开」落成离线。返回有没有变。调用方持有 m.mu。
func settleLocked(r *record, now time.Time) bool {
	if r.State == StateRecentlyLost && !r.LostAt.IsZero() && now.Sub(r.LostAt) >= lostGrace {
		r.State = StateOffline
		return true
	}
	return false
}

// lostLocked 记一次轻探失败：在线的变成刚断开并开始计时；已经断开的按宽限落成离线；其余状态记成出错。调用方持有 m.mu。
func lostLocked(r *record, now time.Time, detail string) {
	r.CheckedAt = now
	r.Detail = detail
	switch r.State {
	case StateOnline:
		r.State = StateRecentlyLost
		r.LostAt = now
	case StateRecentlyLost:
		settleLocked(r, now)
	case StateOffline:
	default:
		r.State = StateError
	}
}

// infoFrom 从握手结果整理出探测信息。
func infoFrom(init acp.InitializeResponse, sess *acp.NewSessionResponse, full bool, now time.Time) *probeInfo {
	info := &probeInfo{At: now, Full: full, Caps: init.AgentCapabilities, Auth: init.AuthMethods}
	if init.AgentInfo != nil {
		info.AgentName = firstNonEmpty(init.AgentInfo.Title, init.AgentInfo.Name)
		info.AgentVersion = init.AgentInfo.Version
	}
	if sess == nil {
		return info
	}
	info.Options = sess.ConfigOptions
	if sess.Models != nil {
		info.Models = sess.Models.AvailableModels
		info.CurrentModel = sess.Models.CurrentModelID
		return info
	}
	for _, opt := range sess.ConfigOptions {
		if opt.Category != "model" {
			continue
		}
		for _, v := range opt.Values() {
			info.Models = append(info.Models, acp.ModelInfo{ModelID: v.Value, Name: v.Name, Description: v.Description})
		}
		info.CurrentModel = opt.CurrentValue
		break
	}
	return info
}
