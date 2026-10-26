// 本文件拥有给界面看的 View，以及从活着的页面同步地址、标题、正文和 JPEG 帧。
// 空闲的 View 不启动 Chrome。lane 空白时写成 isolated。不调用 runtime.Run。

package browser

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// View 是界面上的浏览器状态。Live 表示 CDP 还在。Screenshot 是 data:image/jpeg;base64 帧。Log 最多带最近 24 条。
type View struct {
	URL         string           `json:"url"`
	Title       string           `json:"title"`
	Lane        string           `json:"lane"`
	Profile     string           `json:"profile"`
	ProfileName string           `json:"profileName,omitempty"`
	Recording   bool             `json:"recording,omitempty"`
	TabCount    int              `json:"tabCount,omitempty"`
	Headed      bool             `json:"headed"`
	Live        bool             `json:"live"`
	Text        string           `json:"text"`
	Screenshot  string           `json:"screenshot,omitempty"`
	Preview     string           `json:"preview,omitempty"`
	Log         []map[string]any `json:"log"`
}

// View 返回当前状态的副本。宿主为空时 Lane 是 isolated，不启动浏览器。帧超过 900 毫秒且没有同步在进行时，在后台再调一次 syncView。
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
	name := h.profileName
	if name == "" {
		name = "default"
	}
	recording := false
	if h.rec != nil {
		h.rec.mu.Lock()
		recording = h.rec.active
		h.rec.mu.Unlock()
	}
	return View{
		URL:         h.url,
		Title:       title,
		Lane:        lane,
		Profile:     h.profile,
		ProfileName: name,
		Recording:   recording,
		TabCount:    h.tabCount,
		Headed:      h.headed,
		Live:        h.cdp != nil,
		Text:        text,
		Screenshot:  h.frame,
		Preview:     h.preview,
		Log:         append([]map[string]any(nil), h.log[start:]...),
	}
}

// syncView 从当前 CDP 读取 location.href、document.title 和正文前 4000 字，并抓一张 JPEG。href 表明连接已断时 dropDeadCDP。
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
	href, hrefErr := h.evalOnConn("location.href")
	title, _ := h.evalOnConn("document.title")
	text, _ := h.evalOnConn(`document.body ? (document.body.innerText || "").slice(0, 4000) : ""`)
	shot := h.captureJPEG()
	if isCDPGone(hrefErr) {
		h.dropDeadCDP()
	}
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

// captureJPEG 用质量 42 的 JPEG 截一张图，返回 data URL。失败、解码失败或连接已断时返回空字符串，连接已断时同时 dropDeadCDP。
func (h *Host) captureJPEG() string {
	if h.cdp == nil {
		return ""
	}
	raw, err := h.cdp.call("Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 42})
	if err != nil {
		if isCDPGone(err) {
			h.dropDeadCDP()
		}
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

// cleanJS 去掉空白。空、<nil>、undefined、null 都返回空字符串。
// s 去掉空白。空、<nil>、undefined 或 null 时返回空字符串。
func cleanJS(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" || s == "undefined" || s == "null" {
		return ""
	}
	return s
}

// hostOf 从地址里取出主机部分。没有 :// 时原样返回去掉空白的字符串。
// raw 去掉空白。含 :// 时取其后、到 /、? 或 # 之前的一段。没有 :// 时返回去掉空白后的原文。
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
