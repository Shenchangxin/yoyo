package app_test

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func TestChatProvidersParallelAndClientFor(t *testing.T) {
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	first, err := a.UpsertChatProvider(app.ChatProviderIn{
		Name: "OpenAI", Vendor: "openai", Endpoint: "https://api.openai.com/v1",
		Model: "gpt-4.1-mini", Models: []string{"gpt-4.1-mini", "gpt-4.1"}, MakeDefault: true,
	}, "openai-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.UpsertChatProvider(app.ChatProviderIn{
		Name: "FlowY", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Model: "openclaw/default", Models: []string{"openclaw/default"},
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	list := a.ListChatProviders()
	if len(list) < 2 {
		t.Fatalf("want 2+ providers, got %+v", list)
	}
	var sawFirst, sawSecond bool
	for _, p := range list {
		if p.ID == first.ID {
			sawFirst = true
			if !p.IsDefault {
				t.Fatalf("first should stay default %+v", p)
			}
		}
		if p.ID == second.ID {
			sawSecond = true
			if p.IsDefault {
				t.Fatalf("second should not steal default %+v", p)
			}
		}
	}
	if !sawFirst || !sawSecond {
		t.Fatalf("%+v", list)
	}
	if a.Config.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("default base %s", a.Config.BaseURL)
	}

	cl, err := a.ClientFor(app.SessionMeta{ConnectionID: second.ID, Model: "openclaw/default"})
	if err != nil {
		t.Fatal(err)
	}
	oa, ok := cl.(*runtime.OpenAIClient)
	if !ok {
		t.Fatalf("%T", cl)
	}
	if !strings.Contains(oa.BaseURL, "flowyaipc.com") {
		t.Fatalf("base %s", oa.BaseURL)
	}
	if oa.APIKey != "gw-token" {
		t.Fatalf("key %q", oa.APIKey)
	}

	if err := a.SetDefaultChatProvider(second.ID); err != nil {
		t.Fatal(err)
	}
	if a.Config.BaseURL != "https://server.flowyaipc.com/claw/v1" {
		t.Fatalf("promoted base %s", a.Config.BaseURL)
	}
	if a.Config.Model != "openclaw/default" {
		t.Fatalf("promoted model %s", a.Config.Model)
	}

	sess, err := a.NewSessionOn(t.TempDir(), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ConnectionID != second.ID {
		t.Fatalf("new session connection %q", sess.ConnectionID)
	}
	encoded := second.ID + "::openclaw/default"
	updated, err := a.SetSessionModel(sess.ID, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Model != "openclaw/default" || updated.ConnectionID != second.ID {
		t.Fatalf("%+v", updated)
	}
}
