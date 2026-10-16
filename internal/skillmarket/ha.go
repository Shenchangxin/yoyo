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

var haSticky sync.Map // family -> preferred URL prefix/host

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

func jsdelivrTreeURL(repo, ref string) string {
	return jsdelivrData + repo + "@" + ref + "?structure=flat"
}

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

func lastWrap(raw string, err error) ([]byte, error) {
	return nil, err
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "skill market"
	}
	return u.Host
}

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

func githubish(raw string) bool {
	return strings.Contains(raw, "api.github.com") || strings.Contains(raw, "raw.githubusercontent.com")
}

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

