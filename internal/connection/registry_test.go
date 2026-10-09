package connection

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

func TestRegistryJSONRoundTrip(t *testing.T) {
	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.Upsert(Connection{
		Name: "Seedream", Vendor: "volcengine", Endpoint: "https://ark.example",
		Capabilities: []string{CapImage}, Models: []string{"doubao"},
		DefaultModel: map[string]string{CapImage: "doubao"}, Active: true,
	}, "secret-key")
	if err != nil {
		t.Fatal(err)
	}
	if !c.HasKey || c.ID == "" || c.VaultKey == "" {
		t.Fatalf("%+v", c)
	}
	got, err := r.Get(c.ID)
	if err != nil || got.Name != "Seedream" || !got.HasKey {
		t.Fatalf("get %+v %v", got, err)
	}
	act, err := r.Active(CapImage, "")
	if err != nil || act.ID != c.ID {
		t.Fatalf("active %+v %v", act, err)
	}
	if key, err := r.Lease(act); err != nil || key != "secret-key" {
		t.Fatalf("lease %q %v", key, err)
	}
	defs := r.Defaults()
	if defs[CapImage] != c.ID {
		t.Fatalf("defaults %+v", defs)
	}
	list, err := r.List(CapImage)
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
	if err := r.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(c.ID); err == nil {
		t.Fatal("deleted still present")
	}
}

func TestSeedChatAndSearch(t *testing.T) {
	v := vault.NewFileOnly(t.TempDir())
	v.Set("default", "chat-key")
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.SeedChat("openai", "https://api.openai.com/v1", "gpt-4.1-mini")
	if err != nil {
		t.Fatal(err)
	}
	if !c.HasKey {
		t.Fatal("expected default vault key copied onto chat connection")
	}
	again, err := r.SeedChat("claude", "https://example", "claude")
	if err != nil || again.ID != c.ID {
		t.Fatalf("seed is one-shot %+v %v", again, err)
	}
	synced, err := r.SyncChat("openai", "https://api.openai.com/v1", "gpt-4.1", "", "o4-mini")
	if err != nil {
		t.Fatal(err)
	}
	if ModelFor(synced, CapChat) != "gpt-4.1" {
		t.Fatalf("model %q", ModelFor(synced, CapChat))
	}
	foundExtra := false
	for _, m := range synced.Models {
		if m == "o4-mini" {
			foundExtra = true
		}
	}
	if !foundExtra {
		t.Fatalf("models %v", synced.Models)
	}
	r.SeedSearch("https://search.example/{q}", "search")
	s, err := r.Active(CapSearch, "")
	if err != nil || s.Endpoint != "https://search.example/{q}" {
		t.Fatalf("search %+v %v", s, err)
	}
	cas, err := r.Active(CapStorage, "")
	if err != nil || cas.Protocol != "cas" {
		t.Fatalf("cas %+v %v", cas, err)
	}
}

func isolateVaultEnv(t *testing.T) {
	t.Helper()
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
}

func TestSyncChatAdoptsDefaultKeyAfterSave(t *testing.T) {
	isolateVaultEnv(t)
	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.SeedChat("custom", "https://server.flowyaipc.com/claw/v1", "openclaw/default")
	if err != nil {
		t.Fatal(err)
	}
	if c.HasKey {
		t.Fatal("expected empty key before vault save")
	}
	v.Set("default", "gw-token")
	synced, err := r.SyncChat("custom", "https://server.flowyaipc.com/claw/v1 ", "openclaw/default", "")
	if err != nil {
		t.Fatal(err)
	}
	if !synced.HasKey {
		t.Fatal("SyncChat should copy default vault key onto the chat connection")
	}
	if synced.Endpoint != "https://server.flowyaipc.com/claw/v1" {
		t.Fatalf("endpoint %q", synced.Endpoint)
	}
	key, err := r.Lease(synced)
	if err != nil || key != "gw-token" {
		t.Fatalf("lease %q %v", key, err)
	}
}

func TestSyncChatReplacesStaleConnectionKey(t *testing.T) {
	isolateVaultEnv(t)
	v := vault.NewFileOnly(t.TempDir())
	v.Set("default", "old-openai")
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.SeedChat("openai", "https://api.openai.com/v1", "gpt-4.1-mini")
	if err != nil {
		t.Fatal(err)
	}
	if key, err := r.Lease(c); err != nil || key != "old-openai" {
		t.Fatalf("seed lease %v %v", key, err)
	}
	v.Set("default", "gw-token")
	synced, err := r.SyncChat("custom", "https://server.flowyaipc.com/claw/v1", "openclaw/default", "")
	if err != nil {
		t.Fatal(err)
	}
	key, err := r.Lease(synced)
	if err != nil || key != "gw-token" {
		t.Fatalf("stale connection key kept %q %v", key, err)
	}
}

func TestUpsertChatDoesNotClobberDefaultKey(t *testing.T) {
	isolateVaultEnv(t)
	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.Upsert(Connection{
		Name: "OpenAI", Vendor: "openai", Endpoint: "https://api.openai.com/v1",
		Capabilities: []string{CapChat}, Models: []string{"gpt-4.1-mini"},
		DefaultModel: map[string]string{CapChat: "gpt-4.1-mini"}, Active: true,
	}, "openai-key")
	if err != nil {
		t.Fatal(err)
	}
	_ = r.SetDefault(CapChat, first.ID)
	second, err := r.Upsert(Connection{
		Name: "Custom", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Capabilities: []string{CapChat}, Models: []string{"openclaw/default"},
		DefaultModel: map[string]string{CapChat: "openclaw/default"}, Active: true,
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	def, err := v.Get("default")
	if err != nil || def != "openai-key" {
		t.Fatalf("default vault %q %v", def, err)
	}
	k1, _ := r.Lease(first)
	k2, _ := r.Lease(second)
	if k1 != "openai-key" || k2 != "gw-token" {
		t.Fatalf("keys %q %q", k1, k2)
	}
}

func TestAdoptDefaultKeyDoesNotCopyEnvOntoCustomDefault(t *testing.T) {
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "env-openai")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.Upsert(Connection{
		Name: "Custom", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Capabilities: []string{CapChat}, Models: []string{"openclaw/default"},
		DefaultModel: map[string]string{CapChat: "openclaw/default"}, Active: true,
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	_ = r.SetDefault(CapChat, c.ID)
	v.Set("default", "")
	got := r.adoptDefaultKey(c)
	key, err := r.Lease(got)
	if err != nil || key != "gw-token" {
		t.Fatalf("custom default clobbered with env %q %v", key, err)
	}
}

func TestAdoptDefaultKeyDoesNotClobberCustomWithEnv(t *testing.T) {
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "env-openai")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.Upsert(Connection{
		Name: "OpenAI", Vendor: "openai", Endpoint: "https://api.openai.com/v1",
		Capabilities: []string{CapChat}, Models: []string{"gpt-4.1-mini"},
		DefaultModel: map[string]string{CapChat: "gpt-4.1-mini"}, Active: true,
	}, "openai-key")
	if err != nil {
		t.Fatal(err)
	}
	_ = r.SetDefault(CapChat, first.ID)
	second, err := r.Upsert(Connection{
		Name: "Custom", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Capabilities: []string{CapChat}, Models: []string{"openclaw/default"},
		DefaultModel: map[string]string{CapChat: "openclaw/default"}, Active: true,
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	got := r.adoptDefaultKey(second)
	key, err := r.Lease(got)
	if err != nil || key != "gw-token" {
		t.Fatalf("custom key clobbered %q %v", key, err)
	}
}
