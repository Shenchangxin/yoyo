// 动作阶梯的执行侧：已有可用连接器时拦住 browser_open 和 computer_act，没有虚拟显示器时拦住 screenshot_region。
// 不打开浏览器，也不发送邮件。工具分发在调用前执行 ladderBlock。

package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ladderBlock 在已有可用连接器时拒绝用 computer_act 或 browser_open 去点同一服务。没有连接器则放行。screenshot_region 在没有虚拟显示器时拒绝，并说明不截主屏。
// name 为 computer_act、browser_open 或 screenshot_region 时才继续检查。
// raw 在 computer_act 时转小写交给 connectorForGUI，在 browser_open 时交给 urlHost。
func (t *WorkspaceTools) ladderBlock(name, raw string) error {
	if t == nil || t.Connectors == nil {
		return nil
	}
	switch name {
	case "computer_act":
		app := strings.ToLower(raw)
		if hit := t.connectorForGUI(app); hit != "" {
			return fmt.Errorf("action ladder: use connector_read/send for %s; computer_act is last resort", hit)
		}
	case "browser_open":
		if host := urlHost(raw); host != "" {
			if hit := t.connectorForHost(host); hit != "" {
				return fmt.Errorf("action ladder: live connector %s covers %s; use connector_read/send instead of clicking the site", hit, host)
			}
		}
	case "screenshot_region":
		if t.Computer == nil {
			return fmt.Errorf("screenshot_region captures the virtual display only; primary screen is never captured")
		}
	}
	return nil
}

// connectorForGUI 在应用名像邮件或 Slack，且对应账户在允许列表里时，返回该账户 ID。
// app 转小写后与邮件、浏览器或 slack 词比较。
func (t *WorkspaceTools) connectorForGUI(app string) string {
	blob := strings.ToLower(app)
	for _, a := range t.Connectors.List() {
		if !t.connectorAllowed(a.ID) {
			continue
		}
		p := strings.ToLower(a.Provider)
		switch {
		case (p == "gmail" || p == "outlook") && containsAny(blob, "mail", "outlook", "thunderbird", "gmail", "chrome", "edge"):
			if containsAny(blob, "mail", "outlook", "gmail", "thunderbird") {
				return a.ID
			}
		case p == "slack" && strings.Contains(blob, "slack"):
			return a.ID
		}
	}
	return ""
}

// connectorForHost 按主机名匹配 gmail、outlook 或 slack 的已允许账户。
// host 转小写后与 gmail、outlook 或 slack 的主机片段比较。
func (t *WorkspaceTools) connectorForHost(host string) string {
	host = strings.ToLower(host)
	for _, a := range t.Connectors.List() {
		if !t.connectorAllowed(a.ID) {
			continue
		}
		p := strings.ToLower(a.Provider)
		switch {
		case p == "gmail" && (strings.Contains(host, "mail.google.") || strings.Contains(host, "gmail.")):
			return a.ID
		case p == "outlook" && (strings.Contains(host, "outlook.live.") || strings.Contains(host, "outlook.office") || strings.Contains(host, "office.com")):
			return a.ID
		case p == "slack" && strings.Contains(host, "slack.com"):
			return a.ID
		}
	}
	return ""
}

// connectorAllowed 在允许列表为空、含 * 或含该 ID 时返回真。ID 为空也返回真。
// id 为空时返回 true，否则与允许列表逐项比较。
func (t *WorkspaceTools) connectorAllowed(id string) bool {
	if t == nil || id == "" {
		return true
	}
	allow := t.AllowedConnectors
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if a == id || a == "*" {
			return true
		}
	}
	return false
}

// urlHost 从 URL、JSON 的 url 字段，或已知邮件与 Slack 主机子串里取出主机。都没有则返回空串。
// raw 去掉空白后先按 URL 取 Host，失败再按 JSON 的 url 字段，再按已知邮件和 Slack 主机子串。
func urlHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	var m struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(raw), &m) == nil && m.URL != "" {
		if u, err := url.Parse(m.URL); err == nil {
			return u.Host
		}
	}
	low := strings.ToLower(raw)
	switch {
	case strings.Contains(low, "mail.google.") || strings.Contains(low, "gmail."):
		return "mail.google.com"
	case strings.Contains(low, "outlook.live.") || strings.Contains(low, "outlook.office") || strings.Contains(low, "office.com"):
		return "outlook.office.com"
	case strings.Contains(low, "slack.com"):
		return "slack.com"
	}
	return ""
}
