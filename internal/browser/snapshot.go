// 本文件把 Snapshot 渲染成四档观察：minimal、interactive、full、structured。
// 空档位对远程 http(s) 用 interactive，对工作区页面用 full。不调用模型。

package browser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	// SnapshotMinimal 只写地址、标题和可交互元素个数，值是 minimal。
	SnapshotMinimal = "minimal"
	// SnapshotInteractive 写地址、标题和可见控件，不写正文，值是 interactive。
	SnapshotInteractive = "interactive"
	// SnapshotFull 写正文，并在有控件时附上可见操作，值是 full。
	SnapshotFull = "full"
	// SnapshotStructured 写成带 url、title、elements 的缩进 JSON，值是 structured。
	SnapshotStructured = "structured"
)

// NormalizeSnapshotLevel 选定一档。空档位对远程 http(s) 用 interactive，其余（含工作区 HTML）用 full，因为校验仍要页面正文。已写明的四档原样返回小写。
func NormalizeSnapshotLevel(level string, pageURL string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case SnapshotMinimal, SnapshotInteractive, SnapshotFull, SnapshotStructured:
		return strings.ToLower(strings.TrimSpace(level))
	}
	if isRemoteHTTP(pageURL) {
		return SnapshotInteractive
	}
	return SnapshotFull
}

// isRemoteHTTP 在方案是 http 或 https，且主机不是 127.0.0.1、localhost、::1 时返回真。
func isRemoteHTTP(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host != "127.0.0.1" && host != "localhost" && host != "::1"
}

// FormatSnapshot 按一档把 Snapshot 写成文字。live 为假时先写一行说明隔离 Chrome 没在跑、这是上次的快照。有 Errors 就附在后面，有 Profile 就另起一行 profile=。
func FormatSnapshot(snap Snapshot, level string, live bool) string {
	level = NormalizeSnapshotLevel(level, snap.URL)
	acts := snap.Elements
	if len(acts) == 0 && strings.TrimSpace(snap.Actions) != "" {
		// 控件已经格式化过，interactive 和 full 继续用这段文字。
	}
	var b strings.Builder
	if !live {
		b.WriteString("(isolated Chrome is not running; this is the last snapshot, not the live page)\n")
	}
	switch level {
	case SnapshotMinimal:
		fmt.Fprintf(&b, "Page: %s | %q\nInteractive elements: %d", snap.URL, snap.Title, countActions(snap))
	case SnapshotInteractive:
		fmt.Fprintf(&b, "Page: %s | %q\n", snap.URL, snap.Title)
		if strings.TrimSpace(snap.Actions) != "" {
			b.WriteString("visible actions (prefer ref= on browser_click):\n")
			b.WriteString(snap.Actions)
		} else {
			b.WriteString("no visible interactive elements on this page")
		}
	case SnapshotStructured:
		b.WriteString(formatStructuredSnapshot(snap))
	default:
		body := snap.Text
		if strings.TrimSpace(snap.Actions) != "" {
			if strings.TrimSpace(body) != "" {
				body += "\n\n"
			}
			body += "visible actions (prefer ref= on browser_click):\n" + snap.Actions
		}
		b.WriteString(body)
	}
	if strings.TrimSpace(snap.Errors) != "" {
		b.WriteString("\n\n")
		b.WriteString(snap.Errors)
	}
	if snap.Profile != "" {
		fmt.Fprintf(&b, "\nprofile=%s", snap.Profile)
	}
	return strings.TrimSpace(b.String())
}

// countActions 优先数 Elements。没有时数 Actions 里的非空行。
func countActions(snap Snapshot) int {
	if n := len(snap.Elements); n > 0 {
		return n
	}
	if strings.TrimSpace(snap.Actions) == "" {
		return 0
	}
	n := 0
	for _, line := range strings.Split(snap.Actions, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// formatStructuredSnapshot 把地址、标题和元素编成缩进 JSON。编码失败时退回 Actions 文本。
func formatStructuredSnapshot(snap Snapshot) string {
	els := snap.Elements
	if len(els) == 0 {
		els = nil
	}
	payload := map[string]any{
		"url":      snap.URL,
		"title":    snap.Title,
		"elements": els,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return snap.Actions
	}
	return string(raw)
}

// snapshotWithElements 在加锁快照上填上 ref 缓存里的全部元素，并把 Profile 换成配置名。
func (h *Host) snapshotWithElements() Snapshot {
	snap := h.snapshotLocked()
	snap.Elements = h.refs.all()
	if name := h.ProfileName(); name != "" {
		snap.Profile = name
	}
	return snap
}
