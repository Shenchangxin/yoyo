package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connector"
)

const ActionLadderPin = `
## Action ladder
1. Prefer connector_read / connector_draft / connector_send when a live account exists for that service.
2. Use the isolated browser (browser_open) when there is no connector. Default profile is isolated; attached login is opt-in.
3. computer_act is last resort on the virtual display, never the operator desktop.
Do not screenshot or click a service that already has a live connector.
Scheduled jobs and computer_act require this machine to stay awake. Closing the lid stops the agent.
`

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

func connectorProviders(list []connector.Account) []string {
	var out []string
	for _, a := range list {
		out = append(out, a.Provider)
	}
	return out
}
