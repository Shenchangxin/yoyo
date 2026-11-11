package buddy

import (
	"slices"
	"testing"
)

// TestAdoptYoyoChatFillsEmptyAgent 锁住刚装好、还没配过的 agent 用上 yoyo 的模型和钥匙；
// 自建端点会把地址记进环境变量。已经写过的模型和钥匙不会被盖掉。
func TestAdoptYoyoChatFillsEmptyAgent(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.YoyoChat = func() (string, string, string, string) {
			return "gpt-4.1", "openai", "sk-yoyo-openai", "https://api.openai.com/v1"
		}
	})
	f.m.adoptYoyoChat("codex")
	item := f.settings.get().Item("codex")
	if item.Model != "gpt-4.1" || item.Env != nil {
		t.Fatalf("codex %+v", item)
	}
	if got, _ := f.vault.GetStored("buddy.codex.OPENAI_API_KEY"); got != "sk-yoyo-openai" || !slices.Contains(item.Secrets, "OPENAI_API_KEY") {
		t.Fatalf("key %q secrets %v", got, item.Secrets)
	}

	f.m.opts.YoyoChat = func() (string, string, string, string) {
		return "other-model", "openai", "sk-other", "https://proxy.example/v1"
	}
	f.m.adoptYoyoChat("codex")
	item = f.settings.get().Item("codex")
	if item.Model != "gpt-4.1" || item.Env != nil {
		t.Fatalf("overwrote %+v", item)
	}
	if got, _ := f.vault.GetStored("buddy.codex.OPENAI_API_KEY"); got != "sk-yoyo-openai" {
		t.Fatalf("key overwritten: %s", got)
	}
}

// TestAdoptYoyoChatKeepsOwnModelAndWritesCustomEndpoint 锁住人已经选了模型时只补钥匙；
// 端点不是官方地址时写下 OPENAI_BASE_URL，已有的地址不改。
func TestAdoptYoyoChatKeepsOwnModelAndWritesCustomEndpoint(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.YoyoChat = func() (string, string, string, string) {
			return "deepseek-chat", "deepseek", "sk-yoyo-deepseek", "https://api.deepseek.com/v1"
		}
	})
	_ = f.settings.save(func(s *Settings) {
		s.Items = map[string]ItemConfig{"codex": {Model: "own-model"}}
	})
	f.m.adoptYoyoChat("codex")
	item := f.settings.get().Item("codex")
	if item.Model != "own-model" || item.Env["OPENAI_BASE_URL"] != "https://api.deepseek.com/v1" {
		t.Fatalf("%+v", item)
	}
	if got, _ := f.vault.GetStored("buddy.codex.OPENAI_API_KEY"); got != "sk-yoyo-deepseek" {
		t.Fatalf("key %q", got)
	}

	_ = f.settings.save(func(s *Settings) {
		cur := s.Items["codex"]
		cur.Env["OPENAI_BASE_URL"] = "https://kept.example/v1"
		s.Items["codex"] = cur
	})
	f.vault.Set("buddy.codex.OPENAI_API_KEY", "")
	f.m.keyMu.Lock()
	delete(f.m.keyCache, "buddy.codex.OPENAI_API_KEY")
	f.m.keyMu.Unlock()
	f.m.adoptYoyoChat("codex")
	item = f.settings.get().Item("codex")
	if item.Env["OPENAI_BASE_URL"] != "https://kept.example/v1" {
		t.Fatalf("base url overwritten: %+v", item.Env)
	}
}

// TestAdoptYoyoChatUsesMatchingSecret 锁住提供商对得上时写入这项声明的变量，不写到别的 agent 上。
func TestAdoptYoyoChatUsesMatchingSecret(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.YoyoChat = func() (string, string, string, string) {
			return "claude-sonnet", "anthropic", "sk-ant-yoyo", ""
		}
	})
	f.m.adoptYoyoChat("claude")
	if got, _ := f.vault.GetStored("buddy.claude.ANTHROPIC_API_KEY"); got != "sk-ant-yoyo" {
		t.Fatalf("claude key %q", got)
	}
	if got, _ := f.vault.GetStored("buddy.codex.OPENAI_API_KEY"); got != "" {
		t.Fatalf("codex got %q", got)
	}
	env, values := f.m.secretEnv(Entry{ID: "claude", Secrets: []SecretSpec{{Name: "ANTHROPIC_API_KEY", Provider: "anthropic"}}}, f.settings.get().Item("claude"))
	if env["ANTHROPIC_API_KEY"] != "sk-ant-yoyo" || len(values) != 1 {
		t.Fatalf("env %+v values %v", env, values)
	}
}

// TestAdoptYoyoChatSkipsConfiguredKey 锁住已经有钥匙时不再写入，空着的模型仍会补上。
func TestAdoptYoyoChatSkipsConfiguredKey(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.YoyoChat = func() (string, string, string, string) {
			return "gpt-4.1", "openai", "sk-yoyo", "https://api.openai.com/v1"
		}
	})
	f.vault.Set("buddy.codex.OPENAI_API_KEY", "sk-own")
	f.m.adoptYoyoChat("codex")
	item := f.settings.get().Item("codex")
	if item.Model != "gpt-4.1" {
		t.Fatalf("model %q", item.Model)
	}
	if got, _ := f.vault.GetStored("buddy.codex.OPENAI_API_KEY"); got != "sk-own" {
		t.Fatalf("key %q", got)
	}
}
