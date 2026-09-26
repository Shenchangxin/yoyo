package browser

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type View struct {
	URL        string           `json:"url"`
	Title      string           `json:"title"`
	Lane       string           `json:"lane"`
	Profile    string           `json:"profile"`
	Headed     bool             `json:"headed"`
	Live       bool             `json:"live"`
	Text       string           `json:"text"`
	Screenshot string           `json:"screenshot,omitempty"`
	Log        []map[string]any `json:"log"`
}

func (h *Host) View() View {
	if h == nil {
		return View{Lane: "isolated"}
	}
	h.mu.Lock()
	live := h.cdp != nil
	stale := live && (h.frameAt.IsZero() || time.Since(h.frameAt) > 900*time.Millisecond)
	kick := stale && !h.syncing
	if kick {
		h.syncing = true
	}
	h.mu.Unlock()
	if kick {
		go func() {
			h.syncView()
			h.mu.Lock()
			h.syncing = false
			h.mu.Unlock()
		}()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	lane := h.lane
	if lane == "" {
		lane = "isolated"
	}
	title := strings.TrimSpace(h.title)
	if title == "" || title == "<nil>" {
		title = hostOf(h.url)
	}
	text := h.snap
	if len(text) > 4000 {
		text = text[:4000]
	}
	n := len(h.log)
	start := 0
	if n > 24 {
		start = n - 24
	}
	return View{
		URL:        h.url,
		Title:      title,
		Lane:       lane,
		Profile:    h.profile,
		Headed:     h.headed,
		Live:       h.cdp != nil,
		Text:       text,
		Screenshot: h.frame,
		Log:        append([]map[string]any(nil), h.log[start:]...),
	}
}

func (h *Host) syncView() {
	if h == nil {
		return
	}
	h.mu.Lock()
	cdp := h.cdp
	h.mu.Unlock()
	if cdp == nil {
		return
	}
	href, _ := h.evalOnConn("location.href")
	title, _ := h.evalOnConn("document.title")
	text, _ := h.evalOnConn(`document.body ? (document.body.innerText || "").slice(0, 4000) : ""`)
	shot := h.captureJPEG()
	h.mu.Lock()
	if clean := cleanJS(href); clean != "" {
		h.url = clean
	}
	if clean := cleanJS(title); clean != "" {
		h.title = clean
	}
	if clean := cleanJS(text); clean != "" {
		h.snap = clean
	}
	if shot != "" {
		h.frame = shot
		h.frameAt = time.Now()
	}
	h.mu.Unlock()
}

func (h *Host) captureJPEG() string {
	if h.cdp == nil {
		return ""
	}
	raw, err := h.cdp.call("Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 42})
	if err != nil {
		return ""
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.Data == "" {
		return ""
	}
	if _, err := base64.StdEncoding.DecodeString(out.Data); err != nil {
		return ""
	}
	return "data:image/jpeg;base64," + out.Data
}

func cleanJS(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" || s == "undefined" || s == "null" {
		return ""
	}
	return s
}

func hostOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "://"); i >= 0 {
		rest := raw[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return raw
}
