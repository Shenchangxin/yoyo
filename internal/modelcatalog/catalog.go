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
	DefaultURL = "https://models.dev/api.json"
	CacheTTL   = 12 * time.Hour
	MaxBody    = 32 << 20
	cacheName  = "catalog-v1.json"
	CatalogVer = 3
)

var (
	// URL is the catalog origin. Tests may override.
	URL = DefaultURL
	Get = defaultGet

	mu   sync.Mutex
	live atomic.Pointer[Catalog]
)

// Catalog is the frontend-shaped models.dev subset for yoyo presets.
type Catalog struct {
	Version   int                         `json:"version"`
	FetchedAt string                      `json:"fetchedAt"`
	Source    string                      `json:"source"`
	Stale     bool                        `json:"stale,omitempty"`
	Providers map[string]map[string]Entry `json:"providers"`
}

type Entry struct {
	Model       Model  `json:"model"`
	Family      string `json:"family,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
}

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

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// origin → yoyo preset id. First match wins for a given model id, so
// international hosts (moonshotai, alibaba) beat their -cn twins.
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

func catalogURL() string {
	if v := strings.TrimSpace(os.Getenv("YOYO_MODELS_DEV_URL")); v != "" {
		return v
	}
	return URL
}

func Install(c Catalog) {
	cp := c
	live.Store(&cp)
}

func WindowFor(model string) int {
	c := live.Load()
	if c == nil {
		return 0
	}
	return c.WindowFor(model)
}

func (c Catalog) WindowFor(model string) int {
	e := c.lookup(model)
	if e == nil || e.Model.ContextWindow <= 0 {
		return 0
	}
	return e.Model.ContextWindow
}

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

func staleOrErr(previous Catalog, hasPrev bool, err error) (Catalog, error) {
	if !hasPrev {
		return Catalog{}, err
	}
	previous.Stale = true
	Install(previous)
	return previous, nil
}

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

type apiProvider struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Models map[string]apiModel `json:"models"`
}

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

type apiModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type apiLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type apiCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

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
