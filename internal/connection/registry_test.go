package connection

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

func TestRegistryJSONRoundTrip(t *testing.T) {
	v := vault.New(t.TempDir())
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
	v := vault.New(t.TempDir())
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
