package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type cdpTarget struct {
	Type string `json:"type"`
	WS   string `json:"webSocketDebuggerUrl"`
	URL  string `json:"url"`
}

func pageWS(addr string) (string, error) {
	return pageWSPrefer(addr, "")
}

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
		if wantPage && (tab.URL == want || strings.HasPrefix(tab.URL, want)) {
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

func debugWS(addr string) (string, error) {
	if ws, err := pageWS(addr); err == nil && ws != "" {
		return ws, nil
	}
	return versionWS(addr)
}
