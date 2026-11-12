// route_model_test.go 锁住主会话用哪个模型：它跟着应用的默认走，和它开出的工人是同一个；在设置里换默认，下一轮就换；
// 在主会话上换模型，换的就是应用的默认，钥匙不动。其余会话自己选的模型照旧只管自己，不改默认。
// 连接表换成只读测试 vault 的一份，开发机上的钥匙串拿不到，也不会被写。

package app

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/vault"
)

// modelSpy 记下每次模型调用用的模型名，回一句固定的话。
type modelSpy struct {
	mu     sync.Mutex
	models []string
}

// Chat 记录 req.Model 并回一句好的。
func (s *modelSpy) Chat(_ context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	s.mu.Lock()
	s.models = append(s.models, req.Model)
	s.mu.Unlock()
	return runtime.Message{Role: runtime.RoleAssistant, Content: "好的。"}, nil
}

// last 是最近一次调用用的模型名。
func (s *modelSpy) last() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.models) == 0 {
		return ""
	}
	return s.models[len(s.models)-1]
}

// openModelApp 开一个应用，连接表和钥匙都落在测试目录里，并存好两条聊天连接，第一条是默认。
func openModelApp(t *testing.T) (a *App, first, second ChatProvider) {
	t.Helper()
	for _, name := range []string{vault.EnvAPIKey, "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY"} {
		t.Setenv(name, "")
	}
	a = openBuddyApp(t, t.TempDir(), &fakeAgents{})
	t.Cleanup(func() { _ = a.Close() })
	reg, err := connection.Open(t.TempDir(), a.Vault)
	if err != nil {
		t.Fatal(err)
	}
	a.Video.Conn = reg
	first, err = a.UpsertChatProvider(ChatProviderIn{
		Name: "OpenAI", Vendor: "openai", Endpoint: "https://api.openai.com/v1",
		Model: "gpt-4.1-mini", Models: []string{"gpt-4.1-mini", "gpt-4.1"}, MakeDefault: true,
	}, "openai-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err = a.UpsertChatProvider(ChatProviderIn{
		Name: "FlowY", Vendor: "custom", Endpoint: "https://server.flowyaipc.com/claw/v1",
		Model: "openclaw/default",
	}, "gw-token")
	if err != nil {
		t.Fatal(err)
	}
	return a, first, second
}

// defaultProvider 是设置里标成默认的那条聊天连接。
func defaultProvider(t *testing.T, a *App) ChatProvider {
	t.Helper()
	for _, p := range a.ListChatProviders() {
		if p.IsDefault {
			return p
		}
	}
	t.Fatal("no default chat provider")
	return ChatProvider{}
}

// TestMainSessionFollowsTheAppDefaultModel 锁住主会话的模型就是应用的默认。
func TestMainSessionFollowsTheAppDefaultModel(t *testing.T) {
	a, first, second := openModelApp(t)
	main, err := a.MainSession()
	if err != nil {
		t.Fatal(err)
	}
	spy := &modelSpy{}
	send := func(id, text string) string {
		t.Helper()
		if _, err := a.SendOpts(context.Background(), id, text, spy, nil, false); err != nil {
			t.Fatal(err)
		}
		return spy.last()
	}
	if got := send(main.ID, "你好"); got != "gpt-4.1-mini" {
		t.Fatalf("first turn used %q", got)
	}

	// 在设置里换了默认：主会话建的时候钉下的模型不再拦着，下一轮就用新的。
	if err := a.SetDefaultChatProvider(second.ID); err != nil {
		t.Fatal(err)
	}
	if got := send(main.ID, "换了吗"); got != "openclaw/default" {
		t.Fatalf("after Settings changed the default the router still used %q", got)
	}
	if cl, err := a.ClientFor(a.chatSlotFor(main, main.ID)); err != nil {
		t.Fatal(err)
	} else if oa := runtime.PrimaryOpenAI(cl); oa == nil || oa.APIKey != "gw-token" || !strings.Contains(oa.BaseURL, "flowyaipc.com") {
		t.Fatalf("the router's client is not the default connection: %+v", oa)
	}

	// 在主会话上换模型：换的是应用的默认。连接、模型、钥匙都跟着，工人用的空会话也是它。
	if _, err := a.SetSessionModel(main.ID, first.ID+"::gpt-4.1"); err != nil {
		t.Fatal(err)
	}
	def := defaultProvider(t, a)
	if def.ID != first.ID || def.Model != "gpt-4.1" || !def.HasKey {
		t.Fatalf("default provider after picking in the main session: %+v", def)
	}
	if a.Config.Model != "gpt-4.1" || !strings.Contains(a.Config.BaseURL, "api.openai.com") {
		t.Fatalf("config after picking: model %q base %q", a.Config.Model, a.Config.BaseURL)
	}
	worker, err := a.ClientFor(SessionMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if oa := runtime.PrimaryOpenAI(worker); oa == nil || oa.APIKey != "openai-key" || !strings.Contains(oa.BaseURL, "api.openai.com") {
		t.Fatalf("workers are not on the picked connection: %+v", oa)
	}
	if got := send(main.ID, "现在呢"); got != "gpt-4.1" {
		t.Fatalf("after picking in the main session the router used %q", got)
	}
}

// TestOtherSessionsKeepTheirOwnModel 锁住主会话之外的会话：自己选的模型只管自己，不改应用的默认，这一轮也确实用它。
func TestOtherSessionsKeepTheirOwnModel(t *testing.T) {
	a, first, second := openModelApp(t)
	if _, err := a.MainSession(); err != nil {
		t.Fatal(err)
	}
	other, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetSessionModel(other.ID, second.ID+"::openclaw/default"); err != nil {
		t.Fatal(err)
	}
	if def := defaultProvider(t, a); def.ID != first.ID || def.Model != "gpt-4.1-mini" {
		t.Fatalf("a pick in another session changed the default: %+v", def)
	}
	if a.Config.Model != "gpt-4.1-mini" {
		t.Fatalf("config model %q", a.Config.Model)
	}
	spy := &modelSpy{}
	if _, err := a.SendOpts(context.Background(), other.ID, "你好", spy, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := spy.last(); got != "openclaw/default" {
		t.Fatalf("the session's own pick was ignored: %q", got)
	}
}
