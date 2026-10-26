// trends.go 拉取微博、抖音、知乎、B站、头条和百度的热榜，并缓存五分钟。

package video

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TrendItem 是一条热榜的标题、热度和链接。
type TrendItem struct {
	Title string `json:"title"`
	Hot   string `json:"hot,omitempty"`
	URL   string `json:"url,omitempty"`
}

// TrendGroup 是一个平台的热榜。OK 表示这次拿到了至少一条。
type TrendGroup struct {
	Platform string      `json:"platform"`
	Label    string      `json:"label"`
	Items    []TrendItem `json:"items"`
	OK       bool        `json:"ok"`
}

// trendSources 列出热榜地址。主地址在 60s.viki.moe，部分平台另有 v2.xxapi.cn 备用。知乎和头条没有备用地址。
var trendSources = []struct {
	key, label, primary, fallback string
}{
	{"weibo", "微博", "https://60s.viki.moe/v2/weibo", "https://v2.xxapi.cn/api/weibohot"},
	{"douyin", "抖音", "https://60s.viki.moe/v2/douyin", "https://v2.xxapi.cn/api/douyinhot"},
	{"zhihu", "知乎", "https://60s.viki.moe/v2/zhihu", ""},
	{"bilibili", "B站", "https://60s.viki.moe/v2/bilibili", "https://v2.xxapi.cn/api/bilibilihot"},
	{"toutiao", "头条", "https://60s.viki.moe/v2/toutiao", ""},
	{"baidu", "百度", "https://60s.viki.moe/v2/baidu/hot", "https://v2.xxapi.cn/api/baiduhot"},
}

// trendCacheEntry 记下一次热榜和写入时间。
type trendCacheEntry struct {
	at    time.Time
	group TrendGroup
}

// FetchTrends 按平台拉热榜。limit 小于等于 0 时用 15，大于 30 时用 30。平台为空时拉微博、抖音和知乎。Engine.HTTP 为空时用 12 秒超时的客户端。五分钟内的缓存直接复用。返回 trends 和 updated 的 Unix 秒。
func (e *Engine) FetchTrends(platforms []string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 15
	}
	if limit > 30 {
		limit = 30
	}
	want := map[string]bool{}
	for _, p := range platforms {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			want[p] = true
		}
	}
	if len(want) == 0 {
		want["weibo"] = true
		want["douyin"] = true
		want["zhihu"] = true
	}
	httpDo := e.HTTP
	if httpDo == nil {
		httpDo = &http.Client{Timeout: 12 * time.Second}
	}
	groups := make([]TrendGroup, 0, len(want))
	for _, src := range trendSources {
		if !want[src.key] {
			continue
		}
		g := e.cachedTrend(src.key)
		if g.Platform == "" {
			g = fetchTrendGroup(httpDo, src.key, src.label, src.primary, src.fallback)
			e.storeTrend(src.key, g)
		}
		if len(g.Items) > limit {
			g.Items = g.Items[:limit]
		}
		groups = append(groups, g)
	}
	return map[string]any{"trends": groups, "updated": time.Now().Unix()}, nil
}

// cachedTrend 返回未超过五分钟的缓存。没有缓存时返回零值。
func (e *Engine) cachedTrend(key string) TrendGroup {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.trendCache == nil {
		return TrendGroup{}
	}
	ent, ok := e.trendCache[key]
	if !ok || time.Since(ent.at) > 5*time.Minute {
		return TrendGroup{}
	}
	return ent.group
}

// storeTrend 把这一组热榜放进内存缓存。
func (e *Engine) storeTrend(key string, g TrendGroup) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.trendCache == nil {
		e.trendCache = map[string]trendCacheEntry{}
	}
	e.trendCache[key] = trendCacheEntry{at: time.Now(), group: g}
}

// fetchTrendGroup 先请求主地址。结果为空且备用地址非空时再请求备用。拿到条目时 OK 为真。
func fetchTrendGroup(httpDo HTTPDoer, key, label, primary, fallback string) TrendGroup {
	g := TrendGroup{Platform: key, Label: label, Items: []TrendItem{}}
	items := fetchTrendURL(httpDo, primary)
	if len(items) == 0 && fallback != "" {
		items = fetchTrendURL(httpDo, fallback)
	}
	g.Items = items
	g.OK = len(items) > 0
	return g
}

// fetchTrendURL 用 GET 拉取 JSON，超时 10 秒，Accept 为 application/json，User-Agent 为 yoyo-studio/0.4。状态码大于等于 300、客户端为空或正文读失败时返回 nil。正文最多 1 MiB。
func fetchTrendURL(httpDo HTTPDoer, rawURL string) []TrendItem {
	if httpDo == nil || rawURL == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "yoyo-studio/0.4")
	res, err := httpDo.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil
	}
	return parseTrendJSON(body)
}

// parseTrendJSON 从条目里取标题、热度和链接。标题依次看 title、name、word、query。热度看 hot、hotval、heat、index、num。链接看 url、link、href。没有标题的条目丢掉。
func parseTrendJSON(raw []byte) []TrendItem {
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil
	}
	arr := trendArray(top)
	out := make([]TrendItem, 0, len(arr))
	for _, v := range arr {
		m, _ := v.(map[string]any)
		if m == nil {
			continue
		}
		title := trendStr(m, "title", "name", "word", "query")
		if title == "" {
			continue
		}
		out = append(out, TrendItem{
			Title: title,
			Hot:   trendStr(m, "hot", "hotval", "heat", "index", "num"),
			URL:   trendStr(m, "url", "link", "href"),
		})
	}
	return out
}

// trendArray 从 data 数组、data 里嵌套的 data、以及 data 对象或顶层的 list、items、result、hot 里取出条目数组。
func trendArray(top map[string]any) []any {
	if arr := asSlice(top["data"]); len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			if nested := asSlice(m["data"]); len(nested) > 0 {
				return nested
			}
		}
		return arr
	}
	if m, ok := top["data"].(map[string]any); ok {
		for _, k := range []string{"data", "list", "items", "result", "hot"} {
			if arr := asSlice(m[k]); len(arr) > 0 {
				return arr
			}
		}
	}
	for _, k := range []string{"list", "items", "result"} {
		if arr := asSlice(top[k]); len(arr) > 0 {
			return arr
		}
	}
	return nil
}

// trendStr 返回第一个非空且不是 <nil> 的字段文本。
func trendStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
