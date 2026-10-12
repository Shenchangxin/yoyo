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

type TrendItem struct {
	Title string `json:"title"`
	Hot   string `json:"hot,omitempty"`
	URL   string `json:"url,omitempty"`
}

type TrendGroup struct {
	Platform string      `json:"platform"`
	Label    string      `json:"label"`
	Items    []TrendItem `json:"items"`
	OK       bool        `json:"ok"`
}

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

type trendCacheEntry struct {
	at    time.Time
	group TrendGroup
}

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

func (e *Engine) storeTrend(key string, g TrendGroup) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.trendCache == nil {
		e.trendCache = map[string]trendCacheEntry{}
	}
	e.trendCache[key] = trendCacheEntry{at: time.Now(), group: g}
}

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
