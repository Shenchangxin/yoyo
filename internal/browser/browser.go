package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	Profile string `json:"profile"`
}

type Host struct {
	mu      sync.Mutex
	dir     string
	url     string
	title   string
	profile string
	lane    string
	log     []map[string]any
	cmd     *exec.Cmd
	cdp     *cdpConn
	port    int
	headed  bool
	snap    string
	frame   string
	frameAt time.Time
	syncing bool
}

func Open(dir string) (*Host, error) {
	p := filepath.Join(dir, "profile")
	if err := os.MkdirAll(p, 0o700); err != nil {
		return nil, err
	}
	return &Host{dir: dir, profile: p}, nil
}

func (h *Host) Isolated() bool { return h != nil && h.profile != "" }

func (h *Host) Close() {
	h.mu.Lock()
	cmd := h.cmd
	cdp := h.cdp
	h.cmd = nil
	h.cdp = nil
	h.mu.Unlock()
	if cdp != nil {
		cdp.close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func (h *Host) OpenURL(raw string) (Snapshot, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Snapshot{}, fmt.Errorf("browser: need http(s) URL")
	}
	h.mu.Lock()
	h.url = u.String()
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Snapshot{}, err
	}
	req.Header.Set("User-Agent", "YoyoBrowser/0.3")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	text := strip(string(body))
	if len(text) > 8000 {
		text = text[:8000]
	}
	snap := Snapshot{URL: u.String(), Title: u.Host, Text: text, Profile: h.profile}
	if err := h.navigateCDP(u.String()); err == nil {
		time.Sleep(350 * time.Millisecond)
		h.syncView()
		h.mu.Lock()
		if h.title != "" {
			snap.Title = h.title
		}
		if h.snap != "" {
			snap.Text = h.snap
		}
		h.mu.Unlock()
	}
	h.record("open", raw)
	return snap, nil
}

func (h *Host) Click(sel string) error {
	h.record("click", sel)
	if err := h.clickCDP(sel); err != nil {
		return fmt.Errorf("browser: isolated profile click failed: %w", err)
	}
	h.syncView()
	return nil
}

func (h *Host) Type(sel, text string) error {
	h.record("type", sel)
	if err := h.typeCDP(sel, text); err != nil {
		return fmt.Errorf("browser: isolated profile type failed: %w", err)
	}
	h.syncView()
	return nil
}

func (h *Host) Fill(sel, text string) error { return h.Type(sel, text) }

func (h *Host) Download(raw, dest string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("browser: need http(s) URL")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "YoyoBrowser/0.3")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(resp.Body, 64<<20))
	h.record("download", raw)
	return err
}

func (h *Host) Log() []map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]map[string]any(nil), h.log...)
}

func (h *Host) record(op, detail string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = append(h.log, map[string]any{"op": op, "detail": detail, "ts": time.Now().UTC(), "url": h.url, "profile": h.profile})
	raw, _ := json.Marshal(h.log)
	_ = os.WriteFile(filepath.Join(h.dir, "journal.json"), raw, 0o600)
}

func strip(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch r {
		case '<':
			in = true
		case '>':
			in = false
			b.WriteByte(' ')
		default:
			if !in {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
