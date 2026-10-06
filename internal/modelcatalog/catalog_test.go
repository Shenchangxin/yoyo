package modelcatalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const fixture = `{
  "openai": {
    "id": "openai",
    "name": "OpenAI",
    "models": {
      "gpt-4o": {
        "id": "gpt-4o",
        "name": "GPT-4o",
        "family": "gpt",
        "reasoning": false,
        "release_date": "2024-05-13",
        "modalities": {"input": ["text", "image", "pdf"], "output": ["text"]},
        "limit": {"context": 128000, "output": 16384},
        "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}
      },
      "whisper-1": {
        "id": "whisper-1",
        "name": "Whisper",
        "modalities": {"input": ["audio"], "output": ["audio"]},
        "limit": {"context": 0, "output": 0}
      }
    }
  },
  "anthropic": {
    "id": "anthropic",
    "models": {
      "claude-sonnet-4-6": {
        "id": "claude-sonnet-4-6",
        "name": "Claude Sonnet 4.6",
        "family": "claude-sonnet",
        "reasoning": true,
        "release_date": "2026-02-17",
        "modalities": {"input": ["text", "image"], "output": ["text"]},
        "limit": {"context": 1000000, "output": 128000}
      }
    }
  },
  "moonshotai-cn": {
    "id": "moonshotai-cn",
    "models": {
      "kimi-k2.6": {
        "id": "kimi-k2.6",
        "name": "Kimi K2.6",
        "family": "kimi-k2",
        "release_date": "2026-01-01",
        "modalities": {"input": ["text"], "output": ["text"]},
        "limit": {"context": 262144, "output": 8192}
      }
    }
  }
}`

func TestFamilyOf(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4-6":      "claude-sonnet",
		"gpt-5.2-pro":            "gpt-pro",
		"gpt-4o":                 "gpt",
		"o3-mini":                "o-mini",
		"deepseek-v4-flash":      "deepseek-flash",
		"kimi-k2-0905":           "kimi-k2",
		"text-embedding-3-small": "text-embedding",
		"openai/gpt-4.1-mini":    "gpt-mini",
	}
	for id, want := range cases {
		if got := FamilyOf(id); got != want {
			t.Errorf("%s: got %q want %q", id, got, want)
		}
	}
}

func TestParseMapsProvidersAndDropsAudio(t *testing.T) {
	c, err := Parse([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Providers["claude"]; !ok {
		t.Fatalf("missing claude: %+v", keys(c.Providers))
	}
	if _, ok := c.Providers["openai"]["gpt-4o"]; !ok {
		t.Fatal("missing gpt-4o")
	}
	if _, ok := c.Providers["openai"]["whisper-1"]; ok {
		t.Fatal("whisper should be dropped")
	}
	if _, ok := c.Providers["kimi"]["kimi-k2.6"]; !ok {
		t.Fatal("moonshotai-cn should map to kimi")
	}
	gpt := c.Providers["openai"]["gpt-4o"]
	if gpt.Family != "gpt" || gpt.Model.ContextWindow != 128000 {
		t.Fatalf("%+v", gpt)
	}
	if len(gpt.Model.Input) != 2 || gpt.Model.Input[0] != "text" {
		t.Fatalf("input %v", gpt.Model.Input)
	}
	if gpt.Model.Cost == nil || gpt.Model.Cost.CacheRead != 1.25 {
		t.Fatalf("cost %+v", gpt.Model.Cost)
	}
	if got := c.WindowFor("anthropic/claude-sonnet-4-6"); got != 1_000_000 {
		t.Fatalf("window %d", got)
	}
}

func TestLoadCacheTTLAndStale(t *testing.T) {
	t.Setenv("YOYO_MODELS_DEV_URL", "")
	t.Cleanup(func() { live.Store(nil); Get = defaultGet })
	hits := 0
	Get = func(string) ([]byte, error) {
		hits++
		return []byte(fixture), nil
	}
	dir := t.TempDir()
	c, err := Load(dir, false)
	if err != nil || hits != 1 {
		t.Fatalf("first load: hits=%d err=%v", hits, err)
	}
	if c.Providers["openai"]["gpt-4o"].Model.ID != "gpt-4o" {
		t.Fatalf("%+v", c)
	}
	c2, err := Load(dir, false)
	if err != nil || hits != 1 {
		t.Fatalf("cached: hits=%d err=%v", hits, err)
	}
	if c2.FetchedAt != c.FetchedAt {
		t.Fatal("expected disk cache")
	}
	c3, err := Load(dir, true)
	if err != nil || hits != 2 {
		t.Fatalf("refresh: hits=%d err=%v", hits, err)
	}
	if c3.Providers["openai"] == nil {
		t.Fatal("refresh dropped providers")
	}
	Get = func(string) ([]byte, error) {
		hits++
		return nil, os.ErrDeadlineExceeded
	}
	stale, err := Load(dir, true)
	if err != nil || !stale.Stale || stale.Providers["claude"] == nil {
		t.Fatalf("stale fallback: %+v %v", stale, err)
	}

	// expire cache by rewriting fetchedAt
	p := filepath.Join(dir, cacheName)
	var onDisk Catalog
	b, _ := os.ReadFile(p)
	_ = json.Unmarshal(b, &onDisk)
	onDisk.FetchedAt = time.Now().UTC().Add(-CacheTTL - time.Hour).Format(time.RFC3339Nano)
	b, _ = json.Marshal(onDisk)
	_ = os.WriteFile(p, b, 0o644)
	Get = func(string) ([]byte, error) {
		hits++
		return []byte(fixture), nil
	}
	if _, err := Load(dir, false); err != nil {
		t.Fatal(err)
	}
}

func TestLoadHTTP(t *testing.T) {
	t.Setenv("YOYO_MODELS_DEV_URL", "")
	t.Cleanup(func() { live.Store(nil); Get = defaultGet; URL = DefaultURL })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing UA")
		}
		_, _ = w.Write([]byte(fixture))
	}))
	t.Cleanup(srv.Close)
	URL = srv.URL
	Get = defaultGet
	c, err := Load(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if c.WindowFor("gpt-4o") != 128000 {
		t.Fatalf("%+v", c)
	}
}

func keys(m map[string]map[string]Entry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
