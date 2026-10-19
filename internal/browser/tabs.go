package browser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TabInfo is one page target the isolated Chrome currently has open.
type TabInfo struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Active bool   `json:"active"`
}

func (h *Host) debugAddr() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	port := h.port
	h.mu.Unlock()
	if port <= 0 {
		return ""
	}
	return "127.0.0.1:" + strconv.Itoa(port)
}

func (h *Host) currentTargetID() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.targetID
}

func targetIDFromWS(ws string) string {
	ws = strings.TrimSpace(ws)
	if i := strings.LastIndex(ws, "/"); i >= 0 && i+1 < len(ws) {
		return ws[i+1:]
	}
	return ""
}

func (h *Host) ListTabs() ([]TabInfo, error) {
	if h == nil {
		return nil, fmt.Errorf("browser: no host")
	}
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return nil, err
	}
	addr := h.debugAddr()
	if addr == "" {
		return nil, fmt.Errorf("browser: no debug port")
	}
	tabs, err := listCDPTargets(addr)
	if err != nil {
		return nil, err
	}
	active := h.currentTargetID()
	out := make([]TabInfo, 0, len(tabs))
	for _, tab := range tabs {
		if !strings.EqualFold(tab.Type, "page") {
			continue
		}
		id := strings.TrimSpace(tab.ID)
		if id == "" {
			id = targetIDFromWS(tab.WS)
		}
		out = append(out, TabInfo{
			ID:     id,
			URL:    tab.URL,
			Title:  tab.Title,
			Active: id != "" && id == active,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("browser: no page tabs")
	}
	h.mu.Lock()
	h.tabCount = len(out)
	h.mu.Unlock()
	return out, nil
}

func (h *Host) SwitchTab(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("browser: switch_tab needs a tab id")
	}
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return err
	}
	addr := h.debugAddr()
	tabs, err := listCDPTargets(addr)
	if err != nil {
		return err
	}
	var hit cdpTarget
	for _, tab := range tabs {
		if !strings.EqualFold(tab.Type, "page") {
			continue
		}
		tid := strings.TrimSpace(tab.ID)
		if tid == "" {
			tid = targetIDFromWS(tab.WS)
		}
		if tid == id || strings.HasPrefix(tid, id) || strings.HasPrefix(id, tid) {
			hit = tab
			hit.ID = tid
			break
		}
	}
	if hit.WS == "" {
		return fmt.Errorf("browser: no tab matching %s", id)
	}
	h.invalidateRefs()
	if err := h.dialWS(hit.WS); err != nil {
		return err
	}
	h.mu.Lock()
	h.targetID = hit.ID
	if hit.URL != "" {
		h.url = hit.URL
	}
	h.mu.Unlock()
	h.syncView()
	h.refreshActions()
	h.record("tab", "switch "+hit.ID)
	return nil
}

func (h *Host) NewTab(raw string) error {
	if err := h.ensureCDP(); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "about:blank"
	}
	res, err := h.callCDP("Target.createTarget", map[string]any{"url": raw})
	if err != nil {
		return err
	}
	var out struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(res, &out)
	deadline := time.Now().Add(3 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if out.TargetID != "" {
			if err := h.SwitchTab(out.TargetID); err == nil {
				return nil
			} else {
				last = err
			}
		} else {
			tabs, err := h.ListTabs()
			if err == nil {
				for _, tab := range tabs {
					if tab.URL == raw || strings.HasPrefix(tab.URL, raw) {
						return h.SwitchTab(tab.ID)
					}
				}
			} else {
				last = err
			}
		}
		time.Sleep(80 * time.Millisecond)
	}
	if last != nil {
		return last
	}
	return fmt.Errorf("browser: created tab but could not switch to it")
}

func (h *Host) CloseTab(id string) error {
	if err := h.ensureCDP(); err != nil {
		return err
	}
	tabs, err := h.ListTabs()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = h.currentTargetID()
	}
	var target TabInfo
	var fallback string
	for _, tab := range tabs {
		if tab.ID == id || strings.HasPrefix(tab.ID, id) {
			target = tab
			continue
		}
		if fallback == "" {
			fallback = tab.ID
		}
	}
	if target.ID == "" {
		return fmt.Errorf("browser: no tab matching %s", id)
	}
	if fallback != "" && target.ID == h.currentTargetID() {
		_ = h.SwitchTab(fallback)
	}
	if _, err := h.callCDP("Target.closeTarget", map[string]any{"targetId": target.ID}); err != nil {
		return err
	}
	h.record("tab", "close "+target.ID)
	if fallback != "" {
		return h.SwitchTab(fallback)
	}
	h.invalidateRefs()
	return nil
}
