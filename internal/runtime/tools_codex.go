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

const webFetchUA = "Yoyo/0.3 (+https://github.com/Shenchangxin/yoyo)"

func (t *WorkspaceTools) webFetch(rawURL string) ToolResult {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ToolResult{Err: fmt.Errorf("web_fetch: need http(s) URL")}
	}
	if blockedHost(u.Hostname()) {
		return ToolResult{Err: fmt.Errorf("web_fetch: host is not allowed")}
	}
	if err := t.check(capability.Network, "web_fetch", "", rawURL); err != nil {
		return ToolResult{Err: err}
	}
	ctx := t.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	status, finalURL, ct, body, err := httpGet(ctx, u.String())
	if err != nil {
		return ToolResult{Err: err}
	}
	if alt := retryHTTPOn406(u, status); alt != nil {
		if s2, u2, ct2, b2, e2 := httpGet(ctx, alt.String()); e2 == nil && s2 != 406 {
			status, finalURL, ct, body = s2, u2, ct2, b2
		}
	}
	text := body
	if compact, ok := formatFeed(body); ok {
		text = compact
	} else if isXMLFeed(ct, body) {
		// Keep tags so ids/titles stay recoverable after elision.
	} else if !strings.Contains(strings.ToLower(ct), "json") {
		text = stripTags(text)
	}
	capped, _ := capText(text, 12_000)
	return ToolResult{Content: fmt.Sprintf("HTTP %d %s\n\n%s", status, finalURL, capped)}
}

func httpGet(ctx context.Context, rawURL string) (status int, finalURL, contentType, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	req.Header.Set("User-Agent", webFetchUA)
	req.Header.Set("Accept", "application/atom+xml, application/xml, text/html;q=0.9, */*;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), "", err
	}
	return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), string(raw), nil
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
