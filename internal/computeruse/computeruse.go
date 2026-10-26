// Package computeruse 把操作记到目录里的 recording.json。Open 把 display 设为 virtual，并调用 ensureVirtual。本包不定义能力级别。
// 调用方把 computer_use 当作单独一级；这一级能不能记成 Always 由 capability 决定，这里不审批。
// Windows 上 CreateDesktopW 成功后 display 为 virtual-desktop:YoyoVirtual。截图在桌面句柄为 0 时返回错误，不调用抓屏。click、type、key 在句柄为 0 时仍调用 user32。
// 非 Windows 注入要求环境变量 YOYO_CU_DISPLAY，未设置则返回错误。启动进程时没有把桌面名字写进进程参数。
package computeruse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event 是一次操作记录，含动作、应用、display 字符串，以及是否注入成功。
type Event struct {
	TS       time.Time `json:"ts"`
	Op       string    `json:"op"`
	App      string    `json:"app"`
	Detail   string    `json:"detail"`
	Display  string    `json:"display"`
	Injected bool      `json:"injected,omitempty"`
	Shot     string    `json:"shot,omitempty"`
	Note     string    `json:"note,omitempty"`
}

// Host 保存允许的应用名、事件和显示名。display 初值是 virtual。hwnd 是 Windows 虚拟桌面句柄，其他平台保持 0。
type Host struct {
	mu      sync.Mutex
	dir     string
	allow   map[string]bool
	events  []Event
	display string
	hwnd    uintptr
}

// Open 按 0o700 创建目录，display 设为 virtual，读入 allow.json，再调用 ensureVirtual。
func Open(dir string) (*Host, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	h := &Host{dir: dir, allow: map[string]bool{}, display: "virtual"}
	_ = h.loadAllow()
	h.ensureVirtual()
	return h, nil
}

// Allow 把应用名转成小写后放进允许表，并写回 allow.json。
func (h *Host) Allow(app string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allow[strings.ToLower(app)] = true
	_ = h.flushAllow()
}

// Allowed 返回当前允许的应用名，顺序不稳定。
func (h *Host) Allowed() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for k := range h.allow {
		out = append(out, k)
	}
	return out
}

// Act 先 ensureVirtual。应用名非空、且允许表里没有该名字也没有 * 时返回错误，文案写明虚拟显示、不碰操作者桌面。否则记一条事件；注入失败时错误放进 Note，记录仍写入 recording.json，权限 0o600。
func (h *Host) Act(app, op, detail string) (Event, error) {
	h.ensureVirtual()
	app = strings.ToLower(strings.TrimSpace(app))
	h.mu.Lock()
	defer h.mu.Unlock()
	if app != "" && !h.allow[app] && !h.allow["*"] {
		return Event{}, fmt.Errorf("computer_use: app %q not on allowlist (virtual display only; never the operator desktop)", app)
	}
	ev := Event{TS: time.Now().UTC(), Op: op, App: app, Detail: detail, Display: h.display}
	if shot, err := h.inject(op, app, detail); err != nil {
		ev.Note = err.Error()
	} else {
		ev.Injected = op != "record"
		ev.Shot = shot
	}
	h.events = append(h.events, ev)
	raw, _ := json.MarshalIndent(h.events, "", "  ")
	_ = os.WriteFile(filepath.Join(h.dir, "recording.json"), raw, 0o600)
	return ev, nil
}

// Recording 返回事件切片的副本。
func (h *Host) Recording() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Event(nil), h.events...)
}

// loadAllow 从 allow.json 读入小写应用名。文件或 JSON 无效时返回错误，允许表保持原样。
func (h *Host) loadAllow() error {
	b, err := os.ReadFile(filepath.Join(h.dir, "allow.json"))
	if err != nil {
		return err
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	for _, a := range list {
		h.allow[strings.ToLower(a)] = true
	}
	return nil
}

// flushAllow 把允许表写成 allow.json，权限 0o644。
func (h *Host) flushAllow() error {
	var list []string
	for a := range h.allow {
		list = append(list, a)
	}
	b, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(filepath.Join(h.dir, "allow.json"), b, 0o644)
}
