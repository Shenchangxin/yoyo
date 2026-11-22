// 本文件拥有公开检索用的 JSON 和订阅解析：Hacker News、arXiv、GitHub、DEV、Lobsters、RSS 和 Atom。
// 不拥有来源轮换和开口额度。调用方是 app 的 nativeProbe。登录、浏览器和外部命令不在这里。

package webcontent

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// HNSearchBase 是 Hacker News 按时间检索的地址。测试会临时改它。
	HNSearchBase = "https://hn.algolia.com/api/v1/search_by_date"
	// ArxivQueryBase 是 arXiv API 的地址。测试会临时改它。
	ArxivQueryBase = "https://export.arxiv.org/api/query"
	// DevToArticlesBase 是 DEV 文章列表的地址。测试会临时改它。
	DevToArticlesBase = "https://dev.to/api/articles"
	// LobstersAIBase 是 Lobsters ai 标签的 JSON。测试会临时改它。
	LobstersAIBase = "https://lobste.rs/t/ai.json"

	xmlnsAttr = regexp.MustCompile(`\sxmlns(:[A-Za-z0-9]+)?="[^"]*"`)
)

// HackerNews 按 query 取最近的故事。query 空白时用 AI agent。limit 小于 1 时按 5，大于 20 时按 20。
// ctx 传给 get。client 传给 get。没有标题或没有公开链接时跳过。故事没有外链时用 Hacker News 的讨论页。
func HackerNews(ctx context.Context, client *http.Client, query string, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	query = strings.TrimSpace(query)
	if query == "" {
		query = "AI agent"
	}
	api := strings.TrimRight(HNSearchBase, "/") + "?tags=story&hitsPerPage=" + strconv.Itoa(limit) + "&query=" + url.QueryEscape(query)
	body, err := getOK(ctx, client, api, "application/json")
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Hits []struct {
			Title    string `json:"title"`
			URL      string `json:"url"`
			Story    string `json:"story_text"`
			ObjectID string `json:"objectID"`
		} `json:"hits"`
	}
	if json.Unmarshal([]byte(body), &wrap) != nil {
		return nil, fmt.Errorf("hn: bad json")
	}
	var hits []Hit
	for _, item := range wrap.Hits {
		title := clipText(collapseSpace(stripTags(item.Title)), 180)
		link := hnURL(item.URL, item.ObjectID)
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(stripTags(item.Story)), 400)})
		if len(hits) >= limit {
			break
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("hn: empty")
	}
	return hits, nil
}

// ArxivRecent 在 cs.AI、cs.CL、cs.LG 里按 query 取最新论文。query 空白时用 AI agent。
// limit 小于 1 时按 5，大于 20 时按 20。ctx 传给 get。client 传给 get。
// query 里的加号、括号和引号会换成空格，避免打坏检索式。http 的 arxiv.org 链接改成 https。
func ArxivRecent(ctx context.Context, client *http.Client, query string, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	term := arxivTerm(query)
	search := "all:" + term + " AND (cat:cs.AI OR cat:cs.CL OR cat:cs.LG)"
	api := strings.TrimRight(ArxivQueryBase, "?") + "?search_query=" + url.QueryEscape(search) + "&sortBy=submittedDate&sortOrder=descending&max_results=" + strconv.Itoa(limit)
	body, err := getOK(ctx, client, api, "application/atom+xml, application/xml, text/xml")
	if err != nil {
		return nil, err
	}
	return ParseFeed(body, limit)
}

// GitHubUpdated 检索近 14 天有推送、名字或简介里带 query 的仓库。query 空白时用 AI agent。
// limit 小于 1 时按 5，大于 20 时按 20。ctx 传给 get。client 传给 get。只留下 https 的 html_url。
func GitHubUpdated(ctx context.Context, client *http.Client, query string, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	query = strings.TrimSpace(query)
	if query == "" {
		query = "AI agent"
	}
	since := time.Now().UTC().AddDate(0, 0, -14).Format("2006-01-02")
	q := query + " in:name,description pushed:>" + since
	api := strings.TrimRight(GitHubAPIBase, "/") + "/search/repositories?sort=updated&order=desc&per_page=" + strconv.Itoa(limit) + "&q=" + url.QueryEscape(q)
	body, err := getOK(ctx, client, api, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Items []struct {
			FullName    string `json:"full_name"`
			HTMLURL     string `json:"html_url"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(body), &wrap) != nil {
		return nil, fmt.Errorf("github search: bad json")
	}
	var hits []Hit
	for _, item := range wrap.Items {
		title := clipText(strings.TrimSpace(item.FullName), 180)
		link := strings.TrimSpace(item.HTMLURL)
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(item.Description), 400)})
		if len(hits) >= limit {
			break
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("github search: empty")
	}
	return hits, nil
}

// DevToHot 取 DEV 近一周某个标签下的热文。topic 决定标签：大模型用 llm，人工智能用 machinelearning，其余用 ai。
// limit 小于 1 时按 5，大于 20 时按 20。ctx 传给 get。client 传给 get。
func DevToHot(ctx context.Context, client *http.Client, topic string, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	api := strings.TrimRight(DevToArticlesBase, "/") + "?top=7&per_page=" + strconv.Itoa(limit) + "&tag=" + url.QueryEscape(devTag(topic))
	body, err := getOK(ctx, client, api, "application/json")
	if err != nil {
		return nil, err
	}
	var arr []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if json.Unmarshal([]byte(body), &arr) != nil {
		return nil, fmt.Errorf("devto: bad json")
	}
	var hits []Hit
	for _, item := range arr {
		title := clipText(collapseSpace(stripTags(item.Title)), 180)
		link := strings.TrimSpace(item.URL)
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(stripTags(item.Description)), 400)})
		if len(hits) >= limit {
			break
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("devto: empty")
	}
	return hits, nil
}

// LobstersAI 读取 Lobsters 的 ai 标签。limit 小于 1 时按 5，大于 20 时按 20。
// ctx 传给 get。client 传给 get。外链不是 https 时改用站内短链。
func LobstersAI(ctx context.Context, client *http.Client, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	body, err := getOK(ctx, client, LobstersAIBase, "application/json")
	if err != nil {
		return nil, err
	}
	var arr []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Short       string `json:"short_id_url"`
		Description string `json:"description"`
	}
	if json.Unmarshal([]byte(body), &arr) != nil {
		return nil, fmt.Errorf("lobsters: bad json")
	}
	var hits []Hit
	for _, item := range arr {
		title := clipText(collapseSpace(stripTags(item.Title)), 180)
		link := strings.TrimSpace(item.URL)
		if !strings.HasPrefix(strings.ToLower(link), "https://") {
			link = strings.TrimSpace(item.Short)
		}
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(stripTags(item.Description)), 400)})
		if len(hits) >= limit {
			break
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("lobsters: empty")
	}
	return hits, nil
}

// ReadFeed 拉取一条 RSS 或 Atom。rawURL 空白时返回 feed: empty url。limit 交给 ParseFeed。
// ctx 传给 get。client 传给 get。状态码大于等于 300 时返回错误。
func ReadFeed(ctx context.Context, client *http.Client, rawURL string, limit int) ([]Hit, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("feed: empty url")
	}
	body, err := getOK(ctx, client, rawURL, "application/rss+xml, application/atom+xml, application/xml, text/xml")
	if err != nil {
		return nil, err
	}
	return ParseFeed(body, limit)
}

// ParseFeed 从 RSS 或 Atom 正文里取出标题、https 链接和摘要。limit 小于 1 时按 5，大于 20 时按 20。
// 默认命名空间会先去掉，这样 Atom 的 title 和 link 才能对上。正文被截断时，尽量留下已经完整的 item。
// http 的 arxiv.org 链接改成 https。没有可用条目时返回 feed: empty。
func ParseFeed(body string, limit int) ([]Hit, error) {
	limit = clampLimit(limit)
	body = strings.TrimSpace(strings.TrimPrefix(body, "\uFEFF"))
	if i := strings.Index(body, "<"); i > 0 {
		body = body[i:]
	}
	body = xmlnsAttr.ReplaceAllString(body, "")
	if hits := parseRSS(body, limit); len(hits) > 0 {
		return hits, nil
	}
	if hits := parseAtom(body, limit); len(hits) > 0 {
		return hits, nil
	}
	if i := strings.LastIndex(body, "</item>"); i > 0 {
		repaired := body[:i+len("</item>")] + "</channel></rss>"
		if hits := parseRSS(repaired, limit); len(hits) > 0 {
			return hits, nil
		}
	}
	return nil, fmt.Errorf("feed: empty")
}

func parseRSS(body string, limit int) []Hit {
	var doc struct {
		Channel struct {
			Items []struct {
				Title string `xml:"title"`
				Link  struct {
					Href string `xml:"href,attr"`
					Body string `xml:",chardata"`
				} `xml:"link"`
				Description struct {
					Raw string `xml:",innerxml"`
				} `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if xml.Unmarshal([]byte(body), &doc) != nil {
		return nil
	}
	var hits []Hit
	for _, item := range doc.Channel.Items {
		title := clipText(collapseSpace(stripTags(item.Title)), 180)
		link := cleanURL(item.Link.Href)
		if !strings.HasPrefix(strings.ToLower(link), "https://") {
			link = cleanURL(item.Link.Body)
		}
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(stripTags(item.Description.Raw)), 400)})
		if len(hits) >= limit {
			break
		}
	}
	return hits
}

func parseAtom(body string, limit int) []Hit {
	var doc struct {
		Entries []struct {
			Title   string `xml:"title"`
			ID      string `xml:"id"`
			Summary struct {
				Raw string `xml:",innerxml"`
			} `xml:"summary"`
			Content struct {
				Raw string `xml:",innerxml"`
			} `xml:"content"`
			Links []struct {
				Href string `xml:"href,attr"`
				Rel  string `xml:"rel,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if xml.Unmarshal([]byte(body), &doc) != nil || len(doc.Entries) == 0 {
		return nil
	}
	var hits []Hit
	for _, item := range doc.Entries {
		title := clipText(collapseSpace(stripTags(item.Title)), 180)
		link := atomLink(item.Links, item.ID)
		if title == "" || !strings.HasPrefix(strings.ToLower(link), "https://") {
			continue
		}
		snip := item.Summary.Raw
		if strings.TrimSpace(snip) == "" {
			snip = item.Content.Raw
		}
		hits = append(hits, Hit{Title: title, URL: link, Snippet: clipText(collapseSpace(stripTags(snip)), 400)})
		if len(hits) >= limit {
			break
		}
	}
	return hits
}

func atomLink(links []struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}, id string) string {
	alt := ""
	for _, link := range links {
		href := cleanURL(link.Href)
		if !strings.HasPrefix(strings.ToLower(href), "https://") && !strings.HasPrefix(strings.ToLower(href), "http://") {
			continue
		}
		rel := strings.ToLower(strings.TrimSpace(link.Rel))
		if rel == "" || rel == "alternate" {
			if strings.HasPrefix(strings.ToLower(href), "https://") {
				return href
			}
			if alt == "" {
				alt = href
			}
		}
	}
	if id = cleanURL(id); strings.HasPrefix(strings.ToLower(id), "https://") {
		return id
	}
	if strings.HasPrefix(strings.ToLower(alt), "https://") {
		return alt
	}
	return ""
}

func getOK(ctx context.Context, client *http.Client, rawURL, accept string) (string, error) {
	status, _, _, body, err := get(ctx, client, rawURL, accept)
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("http %d", status)
	}
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("empty body")
	}
	return body, nil
}

func hnURL(raw, objectID string) string {
	raw = cleanURL(raw)
	if strings.HasPrefix(strings.ToLower(raw), "https://") {
		return raw
	}
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return ""
	}
	for _, r := range objectID {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return "https://news.ycombinator.com/item?id=" + objectID
}

func arxivTerm(query string) string {
	query = strings.Map(func(r rune) rune {
		switch r {
		case ':', '+', '(', ')', '"', '\'':
			return ' '
		default:
			return r
		}
	}, query)
	query = strings.Join(strings.Fields(query), " ")
	if query == "" {
		return "AI agent"
	}
	return query
}

func devTag(topic string) string {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "大模型", "llm":
		return "llm"
	case "人工智能", "机器学习", "machine learning":
		return "machinelearning"
	default:
		return "ai"
	}
}

func cleanURL(raw string) string {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	raw = strings.Trim(raw, "<>\"' \t\r\n")
	lower := strings.ToLower(raw)
	for _, prefix := range []string{"http://arxiv.org/", "http://www.arxiv.org/", "http://export.arxiv.org/"} {
		if strings.HasPrefix(lower, prefix) {
			return "https://" + raw[len("http://"):]
		}
	}
	return raw
}

func clipText(text string, n int) string {
	text = strings.TrimSpace(text)
	if n < 1 || utf8.RuneCountInString(text) <= n {
		return text
	}
	return string([]rune(text)[:n])
}
