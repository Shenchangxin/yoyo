package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

func planLanguageMismatch(voice string, steps []string) error {
	if !looksCJK(voice) {
		return nil
	}
	for _, s := range steps {
		if looksCJK(s) {
			return nil
		}
	}
	return fmt.Errorf("update_plan: 操作者用中文下达任务，步骤正文必须用中文")
}

func (t *WorkspaceTools) updatePlan(argsJSON string) ToolResult {
	var p struct {
		Explanation string `json:"explanation"`
		Plan        []struct {
			Step   string `json:"step"`
			Status string `json:"status"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &p); err != nil {
		return ToolResult{Err: fmt.Errorf("update_plan: %w", err)}
	}
	if len(p.Plan) == 0 {
		return ToolResult{Err: fmt.Errorf("update_plan: empty plan")}
	}
	if t != nil && t.ChatOverlay {
		steps := make([]string, 0, len(p.Plan))
		for _, s := range p.Plan {
			steps = append(steps, s.Step)
		}
		if err := planLanguageMismatch(t.OperatorVoice, steps); err != nil {
			return ToolResult{Err: err}
		}
	}
	var b strings.Builder
	if p.Explanation != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Explanation)
	}
	for i, step := range p.Plan {
		st := strings.TrimSpace(step.Status)
		if st == "" {
			st = "pending"
		}
		fmt.Fprintf(&b, "%d. [%s] %s\n", i+1, st, step.Step)
	}
	t.mu.Lock()
	t.PlanText = b.String()
	plan := t.PlanText
	t.mu.Unlock()
	return ToolResult{Content: plan}
}

func (t *WorkspaceTools) wait(seconds int) ToolResult {
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 30 {
		seconds = 30
	}
	ctx := t.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ToolResult{Err: ctx.Err()}
	case <-timer.C:
		return ToolResult{Content: fmt.Sprintf("waited %ds", seconds)}
	}
}

func (t *WorkspaceTools) listSkills() ToolResult {
	if len(t.Skills) == 0 {
		return ToolResult{Content: "no skills loaded"}
	}
	names := make([]string, 0, len(t.Skills))
	for name := range t.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return ToolResult{Content: strings.Join(names, "\n")}
}

func (t *WorkspaceTools) viewImage(rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "view_image", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	st, err := os.Stat(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
	default:
		return ToolResult{Err: fmt.Errorf("view_image: unsupported type %s", ext)}
	}
	head, err := os.ReadFile(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	if len(head) > 64 {
		head = head[:64]
	}
	res := ToolResult{Content: fmt.Sprintf("image %s (%d bytes, %s). pixels attached as a multimodal part.\nmagic=%x", rel, st.Size(), ext, head[:min(8, len(head))])}
	if url, err := encodeDataURL(p); err == nil {
		res.Parts = []ContentPart{{Type: "image_url", ImageURL: url, MIME: "image/" + strings.TrimPrefix(ext, ".")}}
	}
	return res
}

func (t *WorkspaceTools) webFetch(rawURL string) ToolResult {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ToolResult{Err: fmt.Errorf("web_fetch: need http(s) URL")}
	}
	if blockedHost(u.Hostname()) {
		return ToolResult{Err: fmt.Errorf("web_fetch: host is not allowed")}
	}
	if t.Caps != nil {
		req := capability.Request{
			Level:     capability.Network,
			Action:    "web_fetch",
			Path:      rawURL,
			Command:   rawURL,
			SessionID: t.SessionID,
			Workspace: t.Workspace,
			ForceAsk:  true,
		}
		var err error
		if t.Ctx != nil {
			err = t.Caps.CheckCtx(t.Ctx, req)
		} else {
			err = t.Caps.Check(req)
		}
		if err != nil {
			return ToolResult{Err: err}
		}
	}
	ctx := t.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ToolResult{Err: err}
	}
	req.Header.Set("User-Agent", "Yoyo/0.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ToolResult{Err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return ToolResult{Err: err}
	}
	text := string(body)
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "json") {
		text = stripTags(text)
	}
	capped, _ := capText(text, 12_000)
	return ToolResult{Content: fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, rawURL, capped)}
}

func blockedHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || h == "metadata.google.internal" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return blockedIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return true
		}
	}
	return false
}

func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func stripTags(s string) string {
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
