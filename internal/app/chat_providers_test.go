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
	oa := runtime.PrimaryOpenAI(cl)
	if oa == nil {
		t.Fatalf("%T", cl)
	}
	if !strings.Contains(oa.BaseURL, "flowyaipc.com") {
		t.Fatalf("base %s", oa.BaseURL)
	}
	if oa.APIKey != "gw-token" {
		t.Fatalf("key %q", oa.APIKey)
	}
	fc, ok := cl.(*runtime.FailoverClient)
	if !ok {
		t.Fatalf("want failover across settings providers, got %T", cl)
	}
	if len(fc.Routes) < 2 {
		t.Fatalf("want fallback routes, got %+v", fc.Routes)
	}
	if fc.Routes[0].ID != second.ID || fc.Routes[0].Model != "openclaw/default" {
		t.Fatalf("primary route %+v", fc.Routes[0])
	}
	var sawDefault bool
	for _, r := range fc.Routes[1:] {
		if r.ID == first.ID {
			sawDefault = true
			break
		}
	}
	if !sawDefault {
		t.Fatalf("missing settings default in failover chain %+v", fc.Routes)
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

func TestClientForCustomProviderIgnoresOpenAIEnv(t *testing.T) {
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "env-openai-should-not-leak")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	custom, err := a.UpsertChatProvider(app.ChatProviderIn{
		Name: "FlowY", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Model: "AIPC-deepseek-v4.1-flash", MakeDefault: true,
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSessionOn(t.TempDir(), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ConnectionID != custom.ID {
		t.Fatalf("new session connection %q", sess.ConnectionID)
	}
	cl, err := a.ClientFor(sess)
	if err != nil {
		t.Fatal(err)
	}
	oa := runtime.PrimaryOpenAI(cl)
	if oa == nil {
		t.Fatalf("%T", cl)
	}
	if oa.APIKey != "gw-token" {
		t.Fatalf("new session leaked env key %q", oa.APIKey)
	}
	if !strings.Contains(oa.BaseURL, "flowyaipc.com") {
		t.Fatalf("base %s", oa.BaseURL)
	}
	legacy, err := a.ClientFor(app.SessionMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if got := runtime.PrimaryOpenAI(legacy); got == nil || got.APIKey != "gw-token" {
		t.Fatalf("empty session meta leaked %v", got)
	}
}

func TestNewSessionUsesSettingsDefaultNotLastUsed(t *testing.T) {
	t.Setenv("YOYO_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "env-openai-should-not-leak")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	first, err := a.UpsertChatProvider(app.ChatProviderIn{
		Name: "OpenAI", Vendor: "openai", Endpoint: "https://api.openai.com/v1",
		Model: "gpt-4.1-mini", MakeDefault: true,
	}, "openai-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.UpsertChatProvider(app.ChatProviderIn{
		Name: "FlowY", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Model: "openclaw/default",
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSessionOn(t.TempDir(), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ConnectionID != first.ID {
		t.Fatalf("fresh session should start on default %q, got %q", first.ID, sess.ConnectionID)
	}
	if _, err := a.SetSessionModel(sess.ID, second.ID+"::openclaw/default"); err != nil {
		t.Fatal(err)
	}
	picked, err := a.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	pickedClient, err := a.ClientFor(picked)
	if err != nil {
		t.Fatal(err)
	}
	if oa := runtime.PrimaryOpenAI(pickedClient); oa == nil || oa.APIKey != "gw-token" {
		t.Fatalf("picked session should use the custom key, got %+v", oa)
	}
	next, err := a.NewSessionOn(t.TempDir(), "agent")
	if err != nil {
		t.Fatal(err)
	}
	if next.ConnectionID != first.ID {
		t.Fatalf("new session should follow Settings default %q, got %q", first.ID, next.ConnectionID)
	}
	cl, err := a.ClientFor(next)
	if err != nil {
		t.Fatal(err)
	}
	oa := runtime.PrimaryOpenAI(cl)
	if oa == nil {
		t.Fatalf("%T", cl)
	}
	if oa.APIKey != "openai-key" {
		t.Fatalf("new session leaked the wrong key %q", oa.APIKey)
	}
	if !strings.Contains(oa.BaseURL, "api.openai.com") {
		t.Fatalf("base %s", oa.BaseURL)
	}
}
