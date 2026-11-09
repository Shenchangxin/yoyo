// env_test.go 锁住环境清洗、密钥只注入给自己那一项、配置和视图从不带出密钥值、日志打码，以及配置和偏好的取值校验。

package buddy

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildEnvWhitelist 锁住只留白名单变量：YOYO_*、像密钥的变量和不认识的变量都丢掉，并补上 TERM=dumb、NO_COLOR=1。
func TestBuildEnvWhitelist(t *testing.T) {
	env := BuildEnv([]string{
		"PATH=/usr/bin", "HOME=/home/me", "LC_ALL=C", "XDG_CONFIG_HOME=/home/me/.config", "https_proxy=http://proxy:8080",
		"YOYO_DEBUG=1", "OPENAI_API_KEY=sk-1", "GITHUB_TOKEN=ghp", "AWS_SECRET_ACCESS_KEY=a", "DB_PASSWORD=p", "RANDOM_VAR=r", "TERM=xterm-256color",
	})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, keep := range []string{"PATH=/usr/bin", "HOME=/home/me", "LC_ALL=C", "XDG_CONFIG_HOME=", "https_proxy=", "TERM=dumb", "NO_COLOR=1"} {
		if !strings.Contains(joined, "\n"+keep) {
			t.Errorf("missing %s in %v", keep, env)
		}
	}
	for _, drop := range []string{"YOYO_DEBUG", "OPENAI_API_KEY", "GITHUB_TOKEN", "AWS_SECRET", "DB_PASSWORD", "RANDOM_VAR", "xterm"} {
		if strings.Contains(joined, drop) {
			t.Errorf("leaked %s in %v", drop, env)
		}
	}
}

// TestSecretsOnlyForTheirBuddy 锁住 vault 里的密钥只注入给自己那一项；复用 yoyo 的钥匙要人打开才生效。
func TestSecretsOnlyForTheirBuddy(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.YoyoKey = func(provider string) (string, bool) { return "sk-yoyo-openai-key", provider == "openai" }
	})
	a := f.withCustom(t, "alpha")
	b := f.withCustom(t, "beta")
	if _, err := f.m.Configure(a, ConfigPatch{Secrets: map[string]string{"ALPHA_API_KEY": "sk-alpha-123456"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.m.Configure(b, ConfigPatch{Secrets: map[string]string{"BETA_TOKEN": "beta-secret-999"}}); err != nil {
		t.Fatal(err)
	}
	la, err := f.m.Launch(a)
	if err != nil {
		t.Fatal(err)
	}
	if getEnv(la.Env, "ALPHA_API_KEY") != "sk-alpha-123456" || getEnv(la.Env, "BETA_TOKEN") != "" {
		t.Fatalf("alpha env %v", la.Env)
	}
	for _, arg := range append([]string{la.Command}, la.Args...) {
		if strings.Contains(arg, "sk-alpha") {
			t.Fatal("secret on the command line")
		}
	}
	lb, _ := f.m.Launch(b)
	if getEnv(lb.Env, "BETA_TOKEN") != "beta-secret-999" || getEnv(lb.Env, "ALPHA_API_KEY") != "" {
		t.Fatalf("beta env %v", lb.Env)
	}

	codex := Entry{ID: "codex", Secrets: []SecretSpec{{Name: "OPENAI_API_KEY", Provider: "openai"}}}
	if env, _ := f.m.secretEnv(codex, ItemConfig{}); env["OPENAI_API_KEY"] != "" {
		t.Fatal("reused yoyo key without opting in")
	}
	if env, values := f.m.secretEnv(codex, ItemConfig{Reuse: []string{"OPENAI_API_KEY"}}); env["OPENAI_API_KEY"] != "sk-yoyo-openai-key" || len(values) != 1 {
		t.Fatalf("reuse %v", env)
	}
}

// TestConfigureNeverEchoesSecrets 锁住保存密钥后视图、列表和 JSON 里只有 hasKey；空值删除；像密钥的普通变量被拒。
func TestConfigureNeverEchoesSecrets(t *testing.T) {
	f := newFixture(t, nil)
	id := f.withCustom(t, "alpha")
	v, err := f.m.Configure(id, ConfigPatch{Secrets: map[string]string{"ALPHA_API_KEY": "sk-very-secret-value"}})
	if err != nil {
		t.Fatal(err)
	}
	views, _ := f.m.List()
	for _, out := range []any{v, views, f.m.Settings()} {
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), "sk-very-secret-value") {
			t.Fatalf("secret leaked: %s", raw)
		}
	}
	if len(v.Config.Secrets) != 1 || !v.Config.Secrets[0].HasKey || v.Config.Secrets[0].Name != "ALPHA_API_KEY" {
		t.Fatalf("secret view %+v", v.Config.Secrets)
	}
	if got, _ := f.vault.GetStored("buddy.alpha.ALPHA_API_KEY"); got != "sk-very-secret-value" {
		t.Fatal("secret not in the vault")
	}
	v, _ = f.m.Configure(id, ConfigPatch{Secrets: map[string]string{"ALPHA_API_KEY": ""}})
	if got, _ := f.vault.GetStored("buddy.alpha.ALPHA_API_KEY"); len(v.Config.Secrets) != 0 || got != "" {
		t.Fatalf("delete kept %+v", v.Config.Secrets)
	}
	env := map[string]string{"MY_TOKEN": "x"}
	if _, err := f.m.Configure(id, ConfigPatch{Env: &env}); err == nil {
		t.Fatal("secret-like env accepted")
	}
	bad := map[string]string{"1BAD": "x"}
	if _, err := f.m.Configure(id, ConfigPatch{Env: &bad}); err == nil {
		t.Fatal("invalid env name accepted")
	}
	for _, p := range []ConfigPatch{{Mode: ptr("yolo")}, {Approval: ptr("auto")}, {MaxParallel: ptr(9)}} {
		if _, err := f.m.Configure(id, p); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	if _, err := f.m.Configure("nope", ConfigPatch{}); err != ErrNotFound {
		t.Fatalf("unknown id: %v", err)
	}
}

// TestPrefsValidation 锁住偏好的取值范围：npm 源、Python 源、默认 agent 和单轮时长。
func TestPrefsValidation(t *testing.T) {
	f := newFixture(t, nil)
	for _, p := range []PrefsPatch{{NPMRegistry: ptr("http://mirror")}, {PyIndex: ptr("ftp://x")}, {DefaultAgent: ptr("nobody")}, {TurnMinutes: ptr(1)}} {
		if _, err := f.m.SetPrefs(p); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	got, err := f.m.SetPrefs(PrefsPatch{NPMRegistry: ptr("npmmirror"), DefaultAgent: ptr("codex"), TurnMinutes: ptr(90), AutoPick: ptr(true)})
	if err != nil || got.NPMRegistry != "npmmirror" || got.DefaultAgent != "codex" || got.TurnMinutes != 90 || !got.AutoPick {
		t.Fatalf("prefs %+v %v", got, err)
	}
}

// TestRedactText 锁住已知密钥值、Bearer、key=value 形式和 *_KEY= 赋值都会被打码。
func TestRedactText(t *testing.T) {
	in := "boom sk-alpha-123456 Authorization: Bearer abc.def.ghi token=t0p api_key=zzz OPENAI_API_KEY=sk-proj-xyz done"
	out := redactText(in, "sk-alpha-123456")
	for _, leak := range []string{"sk-alpha-123456", "abc.def.ghi", "t0p", "zzz", "sk-proj-xyz"} {
		if strings.Contains(out, leak) {
			t.Errorf("leaked %s in %s", leak, out)
		}
	}
	if !strings.Contains(out, "boom") || !strings.Contains(out, "done") {
		t.Fatalf("over-redacted %s", out)
	}
}

// TestScriptEnvDropsSecrets 锁住写进终端脚本的变量只有 PATH 和加进来的非密钥变量，密钥、TERM 和系统变量都不写。
func TestScriptEnvDropsSecrets(t *testing.T) {
	lines := scriptEnv([]string{"PATH=/a b:/c", "HOME=/home/me", "OPENAI_API_KEY=sk-1", "TERM=dumb", "NO_COLOR=1", "GEMINI_CLI_IDE=0"})
	joined := strings.Join(lines, "\n")
	if joined != "export PATH='/a b:/c'\nexport GEMINI_CLI_IDE=0" {
		t.Fatalf("script env %q", joined)
	}
	if shQuote("it's") != `'it'\''s'` || shQuote("plain-1.2") != "plain-1.2" {
		t.Fatal("shell quoting")
	}
}

// ptr 返回值的指针。
func ptr[T any](v T) *T { return &v }
