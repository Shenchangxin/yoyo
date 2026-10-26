// 计划、等待、技能清单、看图和 web_fetch。主机拒绝与 HTTP 正文上限在本文件。检索顺序在 webnet.go。

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
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

// updatePlan 解析 explanation 和 plan。空计划返回错误。空状态写成 pending。正文写入 PlanText，并 WritePlanFile。
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
	ws, sid := t.Workspace, t.SessionID
	t.mu.Unlock()
	WritePlanFile(ws, sid, plan)
	return ToolResult{Content: plan}
}

// wait 把秒数夹到 1 到 30。用工具上下文等待，上下文取消则返回其错误。
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

// listSkills 优先用 SkillMeta 的名字，否则用 Skills 的键。按 SkillCatalogBudget 截断，零值用 DefaultSkillCatalogBudget。描述超过 skillCatalogDescCap 个 rune 会截断。超预算的条目计为省略。
func (t *WorkspaceTools) listSkills() ToolResult {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	if t != nil && t.SkillMeta != nil {
		for _, s := range t.SkillMeta {
			if s.Name != "" {
				add(s.Name)
			}
		}
	}
	if len(names) == 0 && t != nil {
		for name := range t.Skills {
			add(name)
		}
	}
	if len(names) == 0 {
		return ToolResult{Content: "no skills loaded"}
	}
	sort.Strings(names)
	if t == nil || t.SkillMeta == nil {
		return ToolResult{Content: strings.Join(names, "\n")}
	}
	budget := DefaultSkillCatalogBudget
	if t.SkillCatalogBudget > 0 {
		budget = t.SkillCatalogBudget
	}
	var b strings.Builder
	b.WriteString("Skills catalog (also pinned). load_skill by name.\n")
	used := 0
	omitted := 0
	for _, name := range names {
		desc := ""
		if s, ok := t.SkillMeta[name]; ok {
			desc = strings.TrimSpace(s.Description)
		}
		line := name
		if desc != "" {
			if utf8.RuneCountInString(desc) > skillCatalogDescCap {
				desc = capRunes(desc, skillCatalogDescCap)
			}
			line = name + ": " + desc
		}
		line += "\n"
		if used+len(line) > budget && used > 0 {
			omitted++
			continue
		}
		b.WriteString(line)
		used += len(line)
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "… %d more omitted. Name one with load_skill or @skill.\n", omitted)
	}
	return ToolResult{Content: strings.TrimSpace(b.String())}
}

// viewImage 经 resolve 和 ReadWorkspace 审批后读取图片。扩展名只接受 png、jpg、jpeg、gif、webp、bmp。能编码时把 data URL 放进 Parts。
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

// webFetchUA 是 web_fetch 的 User-Agent。
const webFetchUA = "Yoyo/0.3 (+https://github.com/Shenchangxin/yoyo)"

// webFetch 只接受带主机的 http 或 https。先用 blockedHost 拒绝主机，再做 Network 审批，超时 20 秒。SiteText 成功则截到 12000 个 rune。否则 httpGet。状态 406 时 retryHTTPOn406 得到的地址会再取一次。XML 源保留标签；非 JSON 且非源则去掉标签。shouldJinaReader 为真时再试 Jina。正文截到 12000 个 rune。
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
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if text, ok, _ := webcontent.SiteText(ctx, webHTTP, rawURL); ok {
		capped, _ := capText(text, 12_000)
		return ToolResult{Content: "site " + rawURL + "\n\n" + capped}
	}
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
	feed := false
	if compact, ok := formatFeed(body); ok {
		return ToolResult{Content: fmt.Sprintf("HTTP %d %s\n\n%s", status, finalURL, compact)}
	} else if isXMLFeed(ct, body) {
		feed = true
		// 保留标签，省略之后 id 和标题仍能认出来。
	} else if !strings.Contains(strings.ToLower(ct), "json") {
		text = stripTags(text)
	}
	if !feed && shouldJinaReader(u, status, ct, body, text) {
		if md, err := webcontent.ReadJina(ctx, webHTTP, builtinJinaReader, rawURL); err == nil && strings.TrimSpace(md) != "" {
			capped, _ := capText(md, 12_000)
			return ToolResult{Content: fmt.Sprintf("HTTP %d %s (jina)\n\n%s", status, finalURL, capped)}
		}
	}
	capped, _ := capText(text, 12_000)
	return ToolResult{Content: fmt.Sprintf("HTTP %d %s\n\n%s", status, finalURL, capped)}
}

// jinaSkipLoopback 为真时，localhost、以 .localhost 结尾的名字和环回 IP 不走 Jina。测试可把它改成假。
var jinaSkipLoopback = true

// shouldJinaReader 在 u 为空、环回（当 jinaSkipLoopback 为真）、主机名含 jina.ai 或 duckduckgo.com、或内容类型含 json 时返回假。状态 403、429、503 返回真。其余看 LooksAntibot 或 ThinExtract。
func shouldJinaReader(u *url.URL, status int, ct, raw, stripped string) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if jinaSkipLoopback {
		if host == "localhost" || strings.HasSuffix(host, ".localhost") {
			return false
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return false
		}
	}
	if strings.Contains(host, "jina.ai") || strings.Contains(host, "duckduckgo.com") {
		return false
	}
	if strings.Contains(strings.ToLower(ct), "json") {
		return false
	}
	if status == 403 || status == 429 || status == 503 {
		return true
	}
	return webcontent.LooksAntibot(raw) || webcontent.ThinExtract(stripped, raw)
}

// httpGet 用 http.DefaultClient 发 GET，没有设置 CheckRedirect，跟随重定向时不再检查 blockedHost。正文经 io.LimitReader 截到 256<<10，即 256KiB。
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

// blockedHost 判断 web_fetch 是否拒绝该主机。metadata.google.internal 直接拒绝。localhost 以及以 .localhost 结尾的名字放行。主机名能解析成 IP 时交给 blockedIP：环回地址放行，私网、链路本地单播、链路本地多播和未指定地址拒绝。其余名字做 DNS 查询，查询失败则拒绝；任一解析结果被 blockedIP 拒绝则拒绝。
func blockedHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "metadata.google.internal" {
		return true
	}
	// localhost 和以 .localhost 结尾的名字在这里放行。环回 IP 由 blockedIP 放行。metadata.google.internal 已在上面拒绝。私网、链路本地和未指定地址由 blockedIP 拒绝。
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return false
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

// blockedIP 在 ip 为 nil 时拒绝。环回地址放行。私网、链路本地单播、链路本地多播和未指定地址拒绝。其余放行。
func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return false
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// stripTags 去掉尖括号标签，把标签换成空格，再把连续空白收成一个空格。
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
