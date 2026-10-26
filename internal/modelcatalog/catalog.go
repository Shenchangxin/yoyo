// Package modelcatalog 拉取 models.dev 的模型目录，收成 Yoyo 预设用的子集，并缓存在磁盘上。
// 默认地址是 https://models.dev/api.json。缓存文件名是 catalog-v1.json，新鲜期限是 12 小时。HTTP 正文最多读 32 << 20 字节。
// 拉取失败且磁盘上已有目录时，返回那份目录并把 Stale 设为真，错误改为 nil。
package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultURL 是目录的默认地址 https://models.dev/api.json。
	DefaultURL = "https://models.dev/api.json"
	// CacheTTL 是磁盘缓存的新鲜期限，值为 12 * time.Hour。cacheFresh 用 FetchedAt 与它比较。
	CacheTTL = 12 * time.Hour
	// MaxBody 是 HTTP 正文读取上限，值为 32 << 20。defaultGet 用 LimitReader 截到这个长度。
	MaxBody = 32 << 20
	// cacheName 是磁盘缓存文件名 catalog-v1.json。
	cacheName = "catalog-v1.json"
	// CatalogVer 是写入缓存的版本号，值为 3。Load 和 Parse 都把它写进 Version。
	CatalogVer = 3
)

var (
	// URL 是目录来源。测试可以改它。YOYO_MODELS_DEV_URL 非空时 catalogURL 用环境变量，不用 URL。
	URL = DefaultURL
	// Get 拉取目录正文。默认是 defaultGet。测试可以换成桩。
	Get = defaultGet

	// mu 串行化 Load，避免并发写同一份缓存。
	mu sync.Mutex
	// live 保存最近一次 Install 的目录。WindowFor 和 OutputFor 读它。
	live atomic.Pointer[Catalog]
)

// Catalog 是给 Yoyo 预设用的 models.dev 子集，形状给前端读。
type Catalog struct {
	Version   int                         `json:"version"`
	FetchedAt string                      `json:"fetchedAt"`
	Source    string                      `json:"source"`
	Stale     bool                        `json:"stale,omitempty"`
	Providers map[string]map[string]Entry `json:"providers"`
}

// Entry 是一条模型，带族名和发布日期。
type Entry struct {
	Model       Model  `json:"model"`
	Family      string `json:"family,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
}

// Model 是前端用的模型字段。ContextWindow 和 MaxTokens 来自来源里的 limit。
type Model struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	Reasoning       bool     `json:"reasoning,omitempty"`
	ReasoningLevels []string `json:"reasoningLevels,omitempty"`
	Input           []string `json:"input,omitempty"`
	ContextWindow   int      `json:"contextWindow,omitempty"`
	MaxTokens       int      `json:"maxTokens,omitempty"`
	Cost            *Cost    `json:"cost,omitempty"`
}

// Cost 是输入、输出、缓存读和缓存写的价格。四个数都是 0 时 convertModel 不填 Cost。
type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// originMap 把 models.dev 的来源 id 映到 Yoyo 预设 id。
// 同一个模型 id 先写入的留下，后写入的同 id 被跳过。moonshotai 排在 moonshotai-cn 前面，alibaba 排在 alibaba-cn 前面。
var originMap = []struct{ src, dst string }{
	{"openai", "openai"},
	{"anthropic", "claude"},
	{"google", "gemini"},
	{"deepseek", "deepseek"},
	{"moonshotai", "kimi"},
	{"moonshotai-cn", "kimi"},
	{"zai", "zai"},
	{"zhipuai", "zai"},
	{"xai", "grok"},
	{"alibaba", "qwen"},
	{"alibaba-cn", "qwen"},
	{"groq", "groq"},
}

// catalogURL 返回要拉取的地址。YOYO_MODELS_DEV_URL 去掉空白后非空就用它，否则用 URL。
func catalogURL() string {
	if v := strings.TrimSpace(os.Getenv("YOYO_MODELS_DEV_URL")); v != "" {
		return v
	}
	return URL
}

// Install 存一份目录的副本，供 WindowFor 和 OutputFor 读取。
// c 先复制，再把副本的指针交给 live.Store。
func Install(c Catalog) {
	cp := c
	live.Store(&cp)
}

// WindowFor 从已安装的目录返回模型的上下文长度。还没有目录时返回 0。
// model 在已安装目录非 nil 时传给目录的 WindowFor。目录为 nil 时返回 0。
func WindowFor(model string) int {
	c := live.Load()
	if c == nil {
		return 0
	}
	return c.WindowFor(model)
}

// WindowFor 返回该模型的 ContextWindow。没有条目或长度不是正数时返回 0。
// model 传给 lookup。没有条目或 ContextWindow 小于等于 0 时返回 0。
func (c Catalog) WindowFor(model string) int {
	e := c.lookup(model)
	if e == nil || e.Model.ContextWindow <= 0 {
		return 0
	}
	return e.Model.ContextWindow
}

// OutputFor 从已安装的目录返回该模型写明的输出 token 上限，未知时返回 0。
// 调用方对 0 保留供应商默认值，所以未知模型不会被发一个可能被拒绝的上限。
// model 在已安装目录非 nil 时传给目录的 OutputFor。目录为 nil 时返回 0。
func OutputFor(model string) int {
	c := live.Load()
	if c == nil {
		return 0
	}
	return c.OutputFor(model)
}

// OutputFor 返回该模型的 MaxTokens。没有条目或上限不是正数时返回 0。
// model 传给 lookup。没有条目或 MaxTokens 小于等于 0 时返回 0。
func (c Catalog) OutputFor(model string) int {
	e := c.lookup(model)
	if e == nil || e.Model.MaxTokens <= 0 {
		return 0
	}
	return e.Model.MaxTokens
}

// lookup 按模型 id 找条目。候选依次是原 id、最后一个斜杠后面的一段，以及各自去掉末尾 -\d{8} 的形式。
// 每个候选先按映射键精确匹配，再按大小写不敏感匹配。都没有时返回 nil。
// model 去掉空白。为空或 Providers 为 nil 时返回 nil。否则作为候选，并可能加上最后一个斜杠后的一段，以及去掉末尾日期后缀的形式。
func (c Catalog) lookup(model string) *Entry {
	id := strings.TrimSpace(model)
	if id == "" || c.Providers == nil {
		return nil
	}
	cands := []string{id}
	if i := strings.LastIndex(id, "/"); i >= 0 && i+1 < len(id) {
		cands = append(cands, id[i+1:])
	}
	n := len(cands)
	for i := 0; i < n; i++ {
		u := dateSuffix.ReplaceAllString(cands[i], "")
		if u != cands[i] {
			cands = append(cands, u)
		}
	}
	for _, cand := range cands {
		for _, models := range c.Providers {
			if e, ok := models[cand]; ok {
				return &e
			}
			for k, e := range models {
				if strings.EqualFold(k, cand) {
					ee := e
					return &ee
				}
			}
		}
	}
	return nil
}

// Load 读取或刷新 cacheDir 里的 catalog-v1.json。refresh 为假且缓存仍在 CacheTTL 内时直接用磁盘上的那份。
// 否则调用 Get。网络或解析失败时，若已有上一份就把它标成 Stale 并返回 nil 错误。
// 成功时 Source 写成 models.dev，FetchedAt 是当前 UTC 的 RFC3339Nano，Version 写成 CatalogVer。
// 缓存目录权限 0o755，文件权限 0o644。建目录或编码失败时仍返回这份新目录，只是不一定落盘。
func Load(cacheDir string, refresh bool) (Catalog, error) {
	mu.Lock()
	defer mu.Unlock()
	cachePath := filepath.Join(cacheDir, cacheName)
	previous, hasPrev := readCache(cachePath)
	if hasPrev && !refresh && cacheFresh(previous) {
		Install(previous)
		return previous, nil
	}
	raw, err := Get(catalogURL())
	if err != nil {
		return staleOrErr(previous, hasPrev, err)
	}
	c, err := Parse(raw)
	if err != nil {
		return staleOrErr(previous, hasPrev, err)
	}
	c.Source = "models.dev"
	c.FetchedAt = time.Now().UTC().Format(time.RFC3339Nano)
	c.Version = CatalogVer
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		if b, err := json.Marshal(c); err == nil {
			_ = os.WriteFile(cachePath, b, 0o644)
		}
	}
	Install(c)
	return c, nil
}

// staleOrErr 在没有上一份目录时返回空目录和原错误。有上一份时把 Stale 设为真，Install 之后返回它和 nil。
// previous 在 hasPrev 为真时把 Stale 设为 true，传给 Install，并作为返回值。
// hasPrev 为假时返回空目录和 err。为真时不返回 err。
// err 在 hasPrev 为假时原样返回。
func staleOrErr(previous Catalog, hasPrev bool, err error) (Catalog, error) {
	if !hasPrev {
		return Catalog{}, err
	}
	previous.Stale = true
	Install(previous)
	return previous, nil
}

// cacheFresh 解析 FetchedAt，先试 RFC3339Nano，再试 RFC3339。解析失败返回假。距离现在不超过 CacheTTL 时返回真。
func cacheFresh(c Catalog) bool {
	t, err := time.Parse(time.RFC3339Nano, c.FetchedAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, c.FetchedAt)
	}
	if err != nil {
		return false
	}
	return time.Since(t) <= CacheTTL
}

// readCache 读取缓存文件。读失败、JSON 失败或 Providers 为空时返回假。成功时把 Stale 设为假。
// path 交给 os.ReadFile。读失败、JSON 失败或 Providers 为空时返回 false。
func readCache(path string) (Catalog, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, false
	}
	var c Catalog
	if json.Unmarshal(b, &c) != nil || len(c.Providers) == 0 {
		return Catalog{}, false
	}
	c.Stale = false
	return c, true
}

// apiProvider 是 models.dev 里的一个来源。
type apiProvider struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Models map[string]apiModel `json:"models"`
}

// apiModel 是来源里的一条模型原文。
type apiModel struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Family      string        `json:"family"`
	Reasoning   bool          `json:"reasoning"`
	ReleaseDate string        `json:"release_date"`
	Modalities  apiModalities `json:"modalities"`
	Limit       apiLimit      `json:"limit"`
	Cost        *apiCost      `json:"cost"`
}

// apiModalities 是来源声明的输入和输出模态。
type apiModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// apiLimit 是来源声明的上下文和输出上限。
type apiLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// apiCost 是来源里的价格，字段名用下划线。
type apiCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// Parse 把 models.dev 的 JSON 收成 Catalog。Version 是 CatalogVer，Source 是 models.dev。
// 没有任何映到的来源时返回 models.dev: no mapped providers。
// raw 传给 decodeOrigins。
func Parse(raw []byte) (Catalog, error) {
	origins, err := decodeOrigins(raw)
	if err != nil {
		return Catalog{}, err
	}
	out := Catalog{
		Version:   CatalogVer,
		Source:    "models.dev",
		Providers: map[string]map[string]Entry{},
	}
	for _, m := range originMap {
		src, ok := origins[m.src]
		if !ok || len(src.Models) == 0 {
			continue
		}
		dst := out.Providers[m.dst]
		if dst == nil {
			dst = map[string]Entry{}
			out.Providers[m.dst] = dst
		}
		for key, am := range src.Models {
			e, keep := convertModel(key, am)
			if !keep {
				continue
			}
			if _, exists := dst[e.Model.ID]; exists {
				continue
			}
			dst[e.Model.ID] = e
		}
	}
	if len(out.Providers) == 0 {
		return Catalog{}, fmt.Errorf("models.dev: no mapped providers")
	}
	return out, nil
}

// decodeOrigins 读取来源表。顶层对象能解出并且不是只含一个 providers 键时，就把它当来源表。
// 否则再按 {"providers": ...} 解。包一层的结果为空时返回 models.dev: empty catalog。
// raw 交给 json.Unmarshal。先按顶层来源表解，不行再按带 providers 的包装解。
func decodeOrigins(raw []byte) (map[string]apiProvider, error) {
	var origins map[string]apiProvider
	if err := json.Unmarshal(raw, &origins); err == nil && len(origins) > 0 {
		if _, looksWrapped := origins["providers"]; !looksWrapped || len(origins) > 1 {
			return origins, nil
		}
	}
	var wrap struct {
		Providers map[string]apiProvider `json:"providers"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("models.dev: %w", err)
	}
	if len(wrap.Providers) == 0 {
		return nil, fmt.Errorf("models.dev: empty catalog")
	}
	return wrap.Providers, nil
}

// convertModel 把一条来源模型收成 Entry。id 为空时用映射键。keepModel 拒绝时返回假。
// 名字为空时用 id。family 为空或与 id 只是大小写不同时改用 FamilyOf。输入模态经 filterInput。
// 价格四个字段全是 0 或 Cost 为 nil 时不填 Cost。
// key 在来源模型的 ID 去掉空白后为空时，去掉空白后当作 id。
func convertModel(key string, am apiModel) (Entry, bool) {
	id := strings.TrimSpace(am.ID)
	if id == "" {
		id = strings.TrimSpace(key)
	}
	if id == "" {
		return Entry{}, false
	}
	if !keepModel(am) {
		return Entry{}, false
	}
	name := strings.TrimSpace(am.Name)
	if name == "" {
		name = id
	}
	fam := strings.TrimSpace(am.Family)
	if fam == "" || strings.EqualFold(fam, id) {
		fam = FamilyOf(id)
	}
	in := filterInput(am.Modalities.Input)
	e := Entry{
		Model: Model{
			ID:            id,
			Name:          name,
			Reasoning:     am.Reasoning,
			Input:         in,
			ContextWindow: am.Limit.Context,
			MaxTokens:     am.Limit.Output,
		},
		Family:      fam,
		ReleaseDate: strings.TrimSpace(am.ReleaseDate),
	}
	if am.Cost != nil && (am.Cost.Input != 0 || am.Cost.Output != 0 || am.Cost.CacheRead != 0 || am.Cost.CacheWrite != 0) {
		e.Model.Cost = &Cost{
			Input:      am.Cost.Input,
			Output:     am.Cost.Output,
			CacheRead:  am.Cost.CacheRead,
			CacheWrite: am.Cost.CacheWrite,
		}
	}
	return e, true
}

// keepModel 在上下文长度大于 0 时保留。否则输出模态里要有 text 或 image 才保留。
// am 的 Limit.Context 大于 0 时返回 true。否则看 Modalities.Output 去掉空白并转成小写后是否为 text 或 image。
func keepModel(am apiModel) bool {
	if am.Limit.Context > 0 {
		return true
	}
	if len(am.Modalities.Output) == 0 {
		return false
	}
	for _, o := range am.Modalities.Output {
		switch strings.ToLower(strings.TrimSpace(o)) {
		case "text", "image":
			return true
		}
	}
	return false
}

// filterInput 只留下 text 和 image，转成小写并去重，顺序与第一次出现相同。
// in 逐项去掉空白并转成小写，只留下 text 和 image，并按第一次出现去重。
func filterInput(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range in {
		s := strings.ToLower(strings.TrimSpace(v))
		if s != "text" && s != "image" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// defaultGet 用 GET 拉取 rawURL。超时 25 秒。Accept 是 application/json，User-Agent 是 yoyo-modelcatalog。
// 状态码大于等于 300 时返回错误。正文用 LimitReader 截到 MaxBody。
func defaultGet(rawURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "yoyo-modelcatalog")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models.dev: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxBody))
}
