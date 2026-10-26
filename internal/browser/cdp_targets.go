// 本文件从调试端口的 /json/list 或 /json 列出 CDP 目标，并挑一个 page 的 WebSocket。
// 想要的地址对得上时优先；否则要 http、https 或 file，再退到 about，最后退到任意 page。不调用 runtime.Run。

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// cdpTarget 是 /json/list 里的一条。WS 是 webSocketDebuggerUrl。
type cdpTarget struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	WS    string `json:"webSocketDebuggerUrl"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

// pageWS 返回一个 page 目标的 WebSocket，不指定想要的地址。
// addr 传给 pageWSPrefer，第二个参数是空字符串。
func pageWS(addr string) (string, error) {
	return pageWSPrefer(addr, "")
}

// pageWSPrefer 在 page 目标里挑 WebSocket。want 非空且不是空白页时，URL 相同、有该前缀或 sameDocumentURL 对上就用它。否则优先 http、https、file，再 about，最后任意 page。只要指定页却没有时返回 wanted page。
// addr 传给 listCDPTargets。
func pageWSPrefer(addr, want string) (string, error) {
	tabs, err := listCDPTargets(addr)
	if err != nil {
		return "", err
	}
	want = strings.TrimSpace(want)
	wantPage := want != "" && !isBlankURL(want)
	var httpish, about, fallback string
	for _, tab := range tabs {
		if !strings.EqualFold(tab.Type, "page") || tab.WS == "" {
			continue
		}
		if fallback == "" {
			fallback = tab.WS
		}
		if wantPage && (tab.URL == want || strings.HasPrefix(tab.URL, want) || sameDocumentURL(tab.URL, want)) {
			return tab.WS, nil
		}
		low := strings.ToLower(strings.TrimSpace(tab.URL))
		if strings.HasPrefix(low, "http:") || strings.HasPrefix(low, "https:") || strings.HasPrefix(low, "file:") {
			if httpish == "" {
				httpish = tab.WS
			}
			continue
		}
		if about == "" && (low == "" || strings.HasPrefix(low, "about:")) {
			about = tab.WS
		}
	}
	if httpish != "" {
		return httpish, nil
	}
	if wantPage {
		return "", fmt.Errorf("browser: wanted page %s", want)
	}
	if about != "" {
		return about, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("browser: no page target")
}

// waitDebugPageWS 在截止时间前等一个 page 的 WebSocket。want 非空时先用最多 1.5 秒专等那个地址，然后再退到 debugWS。
// addr 传给 pageWSPrefer 和 debugWS。
// deadline 限定整段等待。专等 want 的上限是现在起 1500 毫秒，晚于 deadline 时改用 deadline。
func waitDebugPageWS(addr, want string, deadline time.Time) (string, error) {
	preferUntil := time.Now().Add(1500 * time.Millisecond)
	if preferUntil.After(deadline) {
		preferUntil = deadline
	}
	var last error
	if strings.TrimSpace(want) != "" && !isBlankURL(want) {
		for time.Now().Before(preferUntil) {
			ws, err := pageWSPrefer(addr, want)
			if err == nil && ws != "" {
				return ws, nil
			}
			last = err
			time.Sleep(80 * time.Millisecond)
		}
	}
	for time.Now().Before(deadline) {
		ws, err := debugWS(addr)
		if err == nil && ws != "" {
			return ws, nil
		}
		last = err
		time.Sleep(80 * time.Millisecond)
	}
	if last != nil {
		return "", last
	}
	return "", fmt.Errorf("browser: no page target")
}

// listCDPTargets 先试 /json/list，失败再试 /json。
// addr 依次配 /json/list 和 /json 传给 fetchCDPTargets。
func listCDPTargets(addr string) ([]cdpTarget, error) {
	var last error
	for _, path := range []string{"/json/list", "/json"} {
		tabs, err := fetchCDPTargets(addr, path)
		if err == nil {
			return tabs, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("browser: no CDP targets")
	}
	return nil, last
}

// fetchCDPTargets 用 800 毫秒超时 GET 指定路径并解码目标列表。状态码大于等于 300 时返回错误。
// addr 拼进请求地址，形式是 http:// 加 addr 加 path。
// path 接在 addr 后面。状态码大于等于 300 时错误文案带上 path 和状态文本。
func fetchCDPTargets(addr, path string) ([]cdpTarget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("browser: %s %s", path, resp.Status)
	}
	var tabs []cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return nil, err
	}
	return tabs, nil
}

// chromeDebugUp 在 150 毫秒内请求 /json/version，状态码小于 300 则认为调试端口在听。
// addr 拼成 http:// 加 addr 加 /json/version，在 150 毫秒内 GET。状态码小于 300 时返回真。
func chromeDebugUp(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/json/version", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 300
}

// debugWS 先要一个 page 的 WebSocket，没有再退到 /json/version 里的浏览器级地址。
// addr 先传给 pageWS。没有得到非空 WebSocket 时再传给 versionWS。
func debugWS(addr string) (string, error) {
	if ws, err := pageWS(addr); err == nil && ws != "" {
		return ws, nil
	}
	return versionWS(addr)
}
