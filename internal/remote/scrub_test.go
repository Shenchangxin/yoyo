package remote

import "testing"

func TestScrubDropsSecretsKeepsPresence(t *testing.T) {
	in := map[string]any{
		"model":      "gpt",
		"api_key":    "sk-live",
		"search_key": "s",
		"has_key":    true,
		"nested":     map[string]any{"token": "t", "name": "openai"},
		"items":      []any{map[string]any{"password": "p", "id": "1"}},
	}
	out := Scrub(in).(map[string]any)
	if _, ok := out["api_key"]; ok {
		t.Fatal("api_key left the machine")
	}
	if _, ok := out["search_key"]; ok {
		t.Fatal("search_key left the machine")
	}
	if out["has_key"] != true {
		t.Fatal("has_key is not a secret")
	}
	nested := out["nested"].(map[string]any)
	if _, ok := nested["token"]; ok || nested["name"] != "openai" {
		t.Fatalf("nested %+v", nested)
	}
}
