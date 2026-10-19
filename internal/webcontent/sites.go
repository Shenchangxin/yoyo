package webcontent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	reYouTubeID  = regexp.MustCompile(`(?i)(?:v=|/embed/|/shorts/|youtu\.be/)([\w-]{11})`)
	reBilibiliBV = regexp.MustCompile(`(?i)\b(BV[0-9A-Za-z]+)`)
	reV2EXTopic  = regexp.MustCompile(`(?i)(?:v2ex\.com/t/|/t/)(\d+)`)
	reGitHubRepo = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/#?]+)`)

	YouTubeOEmbedBase    = "https://www.youtube.com/oembed"
	YouTubeTimedTextBase = "https://www.youtube.com/api/timedtext"
	BilibiliViewBase     = "https://api.bilibili.com/x/web-interface/view"
	V2EXAPIBase          = "https://www.v2ex.com/api"
	GitHubAPIBase        = "https://api.github.com"
)

// SiteText returns structured text for well-known public URLs.
// ok is false when this package has no extractor (caller should HTTP GET).
func SiteText(ctx context.Context, client *http.Client, rawURL string) (text string, ok bool, err error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return "", false, nil
	}
	host := strings.ToLower(u.Hostname())
	var (
		t    string
		err2 error
	)
	switch {
	case host == "youtu.be" || strings.HasSuffix(host, "youtube.com"):
		t, err2 = youtubeText(ctx, client, rawURL)
	case host == "bilibili.com" || strings.HasSuffix(host, ".bilibili.com"):
		t, err2 = bilibiliText(ctx, client, rawURL)
	case host == "v2ex.com" || strings.HasSuffix(host, ".v2ex.com"):
		t, err2 = v2exText(ctx, client, rawURL)
	case host == "github.com":
		t, err2 = githubRepoText(ctx, client, rawURL)
	default:
		return "", false, nil
	}
	if err2 != nil || strings.TrimSpace(t) == "" {
		return "", false, nil
	}
	return t, true, nil
}

func youtubeText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	id := youtubeID(rawURL)
	if id == "" {
		return "", fmt.Errorf("youtube: no video id")
	}
	var b strings.Builder
	oembed := YouTubeOEmbedBase + "?format=json&url=" + url.QueryEscape("https://www.youtube.com/watch?v="+id)
	if status, _, _, body, err := get(ctx, client, oembed, "application/json"); err == nil && status < 300 {
		var meta map[string]any
		if json.Unmarshal([]byte(body), &meta) == nil {
			title := firstString(meta, "title")
			author := firstString(meta, "author_name")
			if title != "" {
				fmt.Fprintf(&b, "YouTube: %s\n", title)
			}
			if author != "" {
				fmt.Fprintf(&b, "channel: %s\n", author)
			}
		}
	}
	fmt.Fprintf(&b, "url: https://www.youtube.com/watch?v=%s\n", id)
	if caps := youtubeCaptions(ctx, client, id); caps != "" {
		b.WriteString("\ntranscript:\n")
		b.WriteString(caps)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("youtube: empty")
	}
	return out, nil
}

func youtubeID(rawURL string) string {
	if m := reYouTubeID.FindStringSubmatch(rawURL); len(m) > 1 {
		return m[1]
	}
	return ""
}

func youtubeCaptions(ctx context.Context, client *http.Client, id string) string {
	listURL := YouTubeTimedTextBase + "?type=list&v=" + url.QueryEscape(id)
	status, _, _, body, err := get(ctx, client, listURL, "application/xml, text/xml")
	if err != nil || status >= 300 || !strings.Contains(body, "lang_code") {
		return fetchTimedText(ctx, client, id, "en")
	}
	lang := "en"
	if i := strings.Index(strings.ToLower(body), `lang_code="`); i >= 0 {
		rest := body[i+11:]
		if j := strings.Index(rest, `"`); j > 0 && j <= 12 {
			lang = rest[:j]
		}
	}
	if strings.Contains(strings.ToLower(body), `lang_code="zh"`) {
		lang = "zh"
	} else if strings.Contains(strings.ToLower(body), `lang_code="zh-hans"`) {
		lang = "zh-Hans"
	}
	return fetchTimedText(ctx, client, id, lang)
}

func fetchTimedText(ctx context.Context, client *http.Client, id, lang string) string {
	u := YouTubeTimedTextBase + "?v=" + url.QueryEscape(id) + "&lang=" + url.QueryEscape(lang)
	status, _, _, body, err := get(ctx, client, u, "application/xml, text/xml")
	if err != nil || status >= 300 {
		return ""
	}
	text := collapseSpace(stripTags(body))
	if len([]rune(text)) > 8000 {
		text = string([]rune(text)[:8000]) + "…"
	}
	return text
}

func bilibiliText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	bv := ""
	if m := reBilibiliBV.FindStringSubmatch(rawURL); len(m) > 1 {
		bv = m[1]
	}
	if bv == "" {
		return "", fmt.Errorf("bilibili: no bvid")
	}
	api := BilibiliViewBase + "?bvid=" + url.QueryEscape(bv)
	status, _, _, body, err := get(ctx, client, api, "application/json")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("bilibili %d", status)
	}
	var wrap struct {
		Code int `json:"code"`
		Data struct {
			Title string `json:"title"`
			Desc  string `json:"desc"`
			Owner struct {
				Name string `json:"name"`
			} `json:"owner"`
			Stat struct {
				View    int `json:"view"`
				Danmaku int `json:"danmaku"`
			} `json:"stat"`
			BVID string `json:"bvid"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(body), &wrap) != nil || wrap.Code != 0 {
		return "", fmt.Errorf("bilibili: bad json")
	}
	d := wrap.Data
	var b strings.Builder
	fmt.Fprintf(&b, "Bilibili: %s\n", strings.TrimSpace(d.Title))
	if d.Owner.Name != "" {
		fmt.Fprintf(&b, "up: %s\n", d.Owner.Name)
	}
	fmt.Fprintf(&b, "url: https://www.bilibili.com/video/%s\n", d.BVID)
	if d.Desc != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(d.Desc))
	}
	return strings.TrimSpace(b.String()), nil
}

func v2exText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, "/api/topics/hot.json") || strings.HasSuffix(strings.TrimRight(lower, "/"), "v2ex.com") {
		return v2exJSON(ctx, client, strings.TrimRight(V2EXAPIBase, "/")+"/topics/hot.json", "V2EX hot")
	}
	id := ""
	if m := reV2EXTopic.FindStringSubmatch(rawURL); len(m) > 1 {
		id = m[1]
	}
	if id == "" {
		return "", fmt.Errorf("v2ex: no topic id")
	}
	return v2exJSON(ctx, client, strings.TrimRight(V2EXAPIBase, "/")+"/topics/show.json?id="+url.QueryEscape(id), "V2EX topic "+id)
}

func v2exJSON(ctx context.Context, client *http.Client, api, label string) (string, error) {
	status, _, _, body, err := get(ctx, client, api, "application/json")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("v2ex %d", status)
	}
	var arr []map[string]any
	if json.Unmarshal([]byte(body), &arr) != nil || len(arr) == 0 {
		return "", fmt.Errorf("v2ex: bad json")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", label)
	limit := len(arr)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		m := arr[i]
		title := firstString(m, "title")
		content := firstString(m, "content", "content_rendered")
		link := firstString(m, "url")
		fmt.Fprintf(&b, "\n%d. %s\n", i+1, title)
		if link != "" {
			fmt.Fprintf(&b, "   %s\n", link)
		}
		if content != "" {
			content = collapseSpace(stripTags(content))
			if len([]rune(content)) > 400 {
				content = string([]rune(content)[:400]) + "…"
			}
			fmt.Fprintf(&b, "   %s\n", content)
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func githubRepoText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	m := reGitHubRepo.FindStringSubmatch(rawURL)
	if len(m) < 3 {
		return "", fmt.Errorf("github: not a repo url")
	}
	owner, repo := m[1], m[2]
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parts := strings.Split(strings.Trim(u.Path, "/"), "/"); len(parts) != 2 {
		return "", fmt.Errorf("github: not a repo root")
	}
	switch strings.ToLower(repo) {
	case "issues", "pulls", "actions", "projects", "settings":
		return "", fmt.Errorf("github: not a repo url")
	}
	api := strings.TrimRight(GitHubAPIBase, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
	status, _, _, body, err := get(ctx, client, api, "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("github %d", status)
	}
	var meta map[string]any
	if json.Unmarshal([]byte(body), &meta) != nil {
		return "", fmt.Errorf("github: bad json")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "GitHub: %s\n", firstString(meta, "full_name"))
	if d := firstString(meta, "description"); d != "" {
		fmt.Fprintf(&b, "%s\n", d)
	}
	if h := firstString(meta, "html_url"); h != "" {
		fmt.Fprintf(&b, "url: %s\n", h)
	}
	if l := firstString(meta, "language"); l != "" {
		fmt.Fprintf(&b, "language: %s\n", l)
	}
	if s, ok := meta["stargazers_count"].(float64); ok {
		fmt.Fprintf(&b, "stars: %.0f\n", s)
	}
	return strings.TrimSpace(b.String()), nil
}
