// GitHub 原始地址和 API 在 jsDelivr 与 gh-proxy 之间切换。

package skillmarket

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	jsdelivrCDN   = "https://cdn.jsdelivr.net/gh/"
	jsdelivrCN    = "https://cdn.jsdmirror.com/gh/"
	ghProxyPrefix = "https://gh-proxy.com/"
	jsdelivrData  = "https://data.jsdelivr.com/v1/packages/gh/"
)

var haSticky sync.Map // 主机名对应上次成功的 URL

// expandHA 给 raw.githubusercontent.com 加上 jsDelivr、jsdmirror 和 gh-proxy。api.github.com 加上 gh-proxy。其他地址只返回自己。上次成功的地址会被排到前面。
func expandHA(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return []string{raw}
	}
	host := strings.ToLower(u.Host)
	var alts []string
	switch host {
	case "raw.githubusercontent.com":
		if js := jsdelivrFromGitHubRaw(u); js != "" {
			alts = []string{raw, jsdelivrCDN + js, jsdelivrCN + js, ghProxyPrefix + raw}
		}
	case "api.github.com":
		alts = []string{raw, ghProxyPrefix + raw}
	}
	if len(alts) == 0 {
		return []string{raw}
	}
	return preferSticky(host, alts)
}

// jsdelivrFromGitHubRaw 把 raw 路径换成 owner/repo@ref/其余。段数少于 4 时返回空串。
func jsdelivrFromGitHubRaw(u *url.URL) string {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	repo := parts[0] + "/" + parts[1]
	ref := parts[2]
	rest := strings.Join(parts[3:], "/")
	if rest == "" {
		return ""
	}
	return repo + "@" + ref + "/" + rest
}

// jsdelivrTreeURL 返回 data.jsdelivr.com 上该仓库修订的扁平文件列表。
func jsdelivrTreeURL(repo, ref string) string {
	return jsdelivrData + repo + "@" + ref + "?structure=flat"
}

// preferSticky 若记住了这个主机上次成功的地址，就把它移到列表最前。
func preferSticky(family string, urls []string) []string {
	pref, ok := haSticky.Load(family)
	if !ok {
		return urls
	}
	want, _ := pref.(string)
	if want == "" {
		return urls
	}
	out := make([]string, 0, len(urls))
	seen := map[string]bool{}
	for _, u := range urls {
		if u == want {
			out = append(out, u)
			seen[u] = true
			break
		}
	}
	for _, u := range urls {
		if seen[u] {
			continue
		}
		out = append(out, u)
	}
	return out
}

// rememberHA 只为 raw.githubusercontent.com 和 api.github.com 记住这次成功的地址。
func rememberHA(original, used string) {
	u, err := url.Parse(original)
	if err != nil || u.Host == "" {
		return
	}
	host := strings.ToLower(u.Host)
	if host != "raw.githubusercontent.com" && host != "api.github.com" {
		return
	}
	haSticky.Store(host, used)
}

// defaultGet 按 expandHA 的顺序尝试。只有一个地址时超时 20 秒，多个时每个 8 秒。成功后记住该地址。全部失败且没有错误时返回 "empty url"。
func defaultGet(rawURL string) ([]byte, error) {
	urls := expandHA(rawURL)
	timeout := 20 * time.Second
	if len(urls) > 1 {
		timeout = 8 * time.Second
	}
	var last error
	for _, u := range urls {
		b, err := httpGet(u, timeout)
		if err == nil {
			rememberHA(rawURL, u)
			return b, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("empty url")
	}
	return lastWrap(rawURL, last)
}

// lastWrap 原样返回最后一个错误。
func lastWrap(raw string, err error) ([]byte, error) {
	return nil, err
}

// hostOf 返回 URL 的主机。解析失败时返回 "skill market"。
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "skill market"
	}
	return u.Host
}

// httpGet 以 Yoyo-Workstation 发起 GET。GitHub 地址在有令牌时带 Bearer。直连 api.github.com 时 Accept 用 application/vnd.github+json。重定向超过 8 次，或跳到非 https，都会失败。状态码大于等于 300 返回 "skill market: " 加状态。正文最多读 MaxBody。
func httpGet(rawURL string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Yoyo-Workstation")
	if tok := githubAuthToken(); tok != "" && githubish(rawURL) {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if strings.Contains(rawURL, "api.github.com") && !strings.Contains(rawURL, "gh-proxy.com") {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	} else {
		req.Header.Set("Accept", "text/plain, text/markdown, application/json, */*")
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 8 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("refusing non-https redirect")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skill market: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxBody))
}

// githubish 在地址含 api.github.com 或 raw.githubusercontent.com 时返回 true。
func githubish(raw string) bool {
	return strings.Contains(raw, "api.github.com") || strings.Contains(raw, "raw.githubusercontent.com")
}

// uniqueBases 去掉空白和重复的基址，并去掉末尾斜杠。
func uniqueBases(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, b := range in {
		b = strings.TrimRight(strings.TrimSpace(b), "/")
		if b == "" || seen[b] {
			continue
		}
		seen[b] = true
		out = append(out, b)
	}
	return out
}
