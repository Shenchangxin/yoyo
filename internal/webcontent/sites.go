// 本文件为 YouTube、Bilibili、V2EX 和 GitHub 仓库根抽文本。没有对应提取器时 SiteText 的 ok 为 false。字幕和帖子正文有单独的 rune 上限。

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
	// reYouTubeID 从 v=、/embed/、/shorts/ 或 youtu.be/ 后面抓 11 位视频 id。
	reYouTubeID = regexp.MustCompile(`(?i)(?:v=|/embed/|/shorts/|youtu\.be/)([\w-]{11})`)
	// reBilibiliBV 抓 BV 开头的视频号。
	reBilibiliBV = regexp.MustCompile(`(?i)\b(BV[0-9A-Za-z]+)`)
	// reV2EXTopic 从 v2ex.com/t/ 或 /t/ 后面抓数字主题 id。
	reV2EXTopic = regexp.MustCompile(`(?i)(?:v2ex\.com/t/|/t/)(\d+)`)
	// reGitHubRepo 抓 github.com/<owner>/<repo>，仓库名停在 /、#、? 之前。
	reGitHubRepo = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/#?]+)`)

	// YouTubeOEmbedBase 是 YouTube oEmbed 地址。
	YouTubeOEmbedBase = "https://www.youtube.com/oembed"
	// YouTubeTimedTextBase 是 YouTube 字幕列表和字幕地址。
	YouTubeTimedTextBase = "https://www.youtube.com/api/timedtext"
	// BilibiliViewBase 是 Bilibili 视频信息接口。
	BilibiliViewBase = "https://api.bilibili.com/x/web-interface/view"
	// V2EXAPIBase 是 V2EX API 根。
	V2EXAPIBase = "https://www.v2ex.com/api"
	// GitHubAPIBase 是 GitHub API 根。测试会临时改它。
	GitHubAPIBase = "https://api.github.com"
)

// SiteText 为已知公开站点返回结构化文本。主机是 youtu.be 或以 youtube.com 结尾、bilibili.com 或其子域、v2ex.com 或其子域、或正好是 github.com 时才尝试提取。
// 解析失败、主机为空、没有提取器、提取出错或文本空白时 ok 为 false 且 err 为 nil。
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

// youtubeText 用 oEmbed 取标题和频道，再附上字幕。没有 11 位 id 时错误文案是 youtube: no video id。拼出来仍空白时错误文案是 youtube: empty。
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

// youtubeID 返回正则抓到的 11 位 id，没有则返回空字符串。
func youtubeID(rawURL string) string {
	if m := reYouTubeID.FindStringSubmatch(rawURL); len(m) > 1 {
		return m[1]
	}
	return ""
}

// youtubeCaptions 先拉字幕列表。失败、状态码大于等于 300 或不含 lang_code 时改拉 en。
// 否则取第一个 lang_code，长度不超过 12。列表里有 zh 就用 zh，有 zh-hans 就用 zh-Hans。
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

// fetchTimedText 拉取指定语言的字幕，去掉标签并收空白。超过 8000 个 rune 时截断并加 …。失败或状态码大于等于 300 时返回空字符串。
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

// bilibiliText 用 BV 号请求查看接口。没有 BV 号时错误文案是 bilibili: no bvid。code 不是 0 或 JSON 无效时错误文案是 bilibili: bad json。
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

// v2exText 在地址含 /api/topics/hot.json，或去掉末尾斜杠后以 v2ex.com 结尾时，拉热门主题。否则按主题 id 拉 show.json。没有 id 时错误文案是 v2ex: no topic id。
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

// v2exJSON 把主题数组格式化成编号列表，最多 8 条。每条正文去掉标签后超过 400 个 rune 就截断并加 …。JSON 不是非空数组时错误文案是 v2ex: bad json。
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

// githubRepoText 只接受路径正好两段的仓库根。仓库名小写后是 issues、pulls、actions、projects、settings 时不当成仓库。
// 请求 GitHubAPIBase/repos/<owner>/<repo>，Accept 为 application/vnd.github+json。JSON 无效时错误文案是 github: bad json。
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
