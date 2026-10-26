// 锁住连接 JSON 只留钥匙名字，以及 default 里已保存的钥匙会抄到聊天连接，环境变量不会覆盖自定义钥匙。

package connection

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

// TestRegistryJSONRoundTrip 确认写入的钥匙能 Lease 回来，默认项指向该连接，删除后文档不再存在。
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

// TestSeedChatAndSearch 确认已保存的 default 钥匙被抄到聊天连接，SeedChat 不会再新建一条，搜索连接记下钥匙名字。
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

// isolateVaultEnv 把 YOYO_API_KEY、OPENAI_API_KEY、ANTHROPIC_API_KEY 和 DEEPSEEK_API_KEY 设成空串。
func isolateVaultEnv(t *testing.T) {
	t.Helper()
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
}

// TestSyncChatAdoptsDefaultKeyAfterSave 确认保存 default 之前连接没有钥匙，保存后再 SyncChat 会抄上并更新端点。
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

// TestSyncChatReplacesStaleConnectionKey 确认默认聊天连接上的旧钥匙会被已保存的 default 换掉。
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

// TestUpsertChatDoesNotClobberDefaultKey 确认非默认聊天连接的 Upsert 不会改掉名字 default 里的钥匙。
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

// TestAdoptDefaultKeyDoesNotCopyEnvOntoCustomDefault 确认环境变量里的钥匙不会写进自定义默认连接。
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

// TestAdoptDefaultKeyDoesNotClobberCustomWithEnv 确认已有自定义钥匙时，环境变量不会把它覆盖掉。
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
