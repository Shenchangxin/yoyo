// probe_test.go 锁住探测和健康检查：全探成功记下模型和能力、要求登录记成 needs_auth、缺必填密钥不启动进程、
// 轻探失败进入「刚断开」并在 5 分钟后落成离线、正在服务工单的视为在线，以及图标清洗。

package buddy

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
)

// TestFullProbeOnline 锁住全探成功后在线，模型、能力和登录方式都进视图，探测在 buddies/.probe/<id> 里进行。
func TestFullProbeOnline(t *testing.T) {
	var dir atomic.Value
	f := newFixture(t, func(o *Options) {
		dial := pipeDial(acptest.Script{
			Caps:        map[string]any{"loadSession": true, "sessionCapabilities": map[string]any{"resume": map[string]any{}}},
			AuthMethods: []acp.AuthMethod{{ID: "oauth", Name: "Sign in"}},
			Models:      []acp.ModelInfo{{ModelID: "m-large", Name: "Large"}, {ModelID: "m-small"}},
		})
		o.Dial = func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
			dir.Store(opts.Dir)
			return dial(ctx, opts)
		}
	})
	id := f.withCustom(t, "fake")
	v, err := f.m.Probe(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if v.State != StateOnline || len(v.Config.Models) != 2 || v.Config.Models[1].Name != "m-small" || !v.Caps.Resume || !v.Caps.Load {
		t.Fatalf("view %+v caps %+v", v.Config.Models, v.Caps)
	}
	if v.Auth == nil || len(v.Auth.Methods) != 1 || v.Auth.Methods[0].Type != "agent" {
		t.Fatalf("auth %+v", v.Auth)
	}
	if got, _ := dir.Load().(string); !strings.HasSuffix(strings.ReplaceAll(got, `\`, "/"), ".probe/fake") {
		t.Fatalf("probe dir %q", got)
	}
	if !f.m.Ready(id) || !f.m.Ready(YoyoID) {
		t.Fatal("online buddy not ready")
	}
	roster := f.m.Roster()
	if len(roster) != 2 || roster[0].ID != YoyoID || roster[1].ID != id || !roster[1].Ready {
		t.Fatalf("roster %+v", roster)
	}
}

// TestProbeNeedsAuth 锁住 session/new 要求登录时记成 needs_auth，并保留握手里的登录方式。
func TestProbeNeedsAuth(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.Dial = pipeDial(acptest.Script{NeedAuth: true, AuthMethods: []acp.AuthMethod{{ID: "login", Name: "Log in", Type: "terminal"}}})
	})
	id := f.withCustom(t, "fake")
	v, _ := f.m.Probe(context.Background(), id)
	if v.State != StateNeedsAuth || v.Auth == nil || v.Auth.Methods[0].ID != "login" || v.Auth.Methods[0].Type != "terminal" {
		t.Fatalf("view %s %+v", v.State, v.Auth)
	}
	if f.m.Ready(id) {
		t.Fatal("needs_auth buddy is ready")
	}
}

// TestProbeFailureIsRedacted 锁住全探失败记成出错，详情和探测日志里的密钥值被打码。
func TestProbeFailureIsRedacted(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.Dial = func(context.Context, acp.Options) (*acp.Client, error) {
			return nil, errors.New("agent exited: 1: invalid key sk-alpha-123456")
		}
	})
	id := f.withCustom(t, "fake")
	_, _ = f.m.Configure(id, ConfigPatch{Secrets: map[string]string{"FAKE_API_KEY": "sk-alpha-123456"}})
	v, _ := f.m.Probe(context.Background(), id)
	if v.State != StateError || strings.Contains(v.Detail, "sk-alpha-123456") || !strings.Contains(v.Detail, "invalid key") {
		t.Fatalf("view %s %q", v.State, v.Detail)
	}
	logs, _ := f.m.LogTail(id, 50)
	if !strings.Contains(logs, "invalid key") || strings.Contains(logs, "sk-alpha-123456") {
		t.Fatalf("log %q", logs)
	}
}

// TestNeedsConfigSkipsLaunch 锁住缺必填密钥时直接记成 needs_config，不启动进程；填上后才去探测。
func TestNeedsConfigSkipsLaunch(t *testing.T) {
	var dials atomic.Int32
	f := newFixture(t, func(o *Options) {
		dial := pipeDial(acptest.Script{})
		o.Dial = func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
			dials.Add(1)
			return dial(ctx, opts)
		}
	})
	id := f.withCustom(t, "fake")
	f.m.mu.Lock()
	for i := range f.m.entries {
		if f.m.entries[i].ID == id {
			f.m.entries[i].Secrets = []SecretSpec{{Name: "FAKE_API_KEY", Required: true}}
		}
	}
	f.m.mu.Unlock()
	v, _ := f.m.Probe(context.Background(), id)
	if v.State != StateNeedsConfig || v.Detail != "FAKE_API_KEY" || dials.Load() != 0 {
		t.Fatalf("view %s %q dials %d", v.State, v.Detail, dials.Load())
	}
	f.vault.Set(secretKey(id, "FAKE_API_KEY"), "sk-filled-in-now")
	if v, _ := f.m.Probe(context.Background(), id); v.State != StateOnline || dials.Load() != 1 {
		t.Fatalf("after filling: %s dials %d", v.State, dials.Load())
	}
}

// TestRecentlyLostTiming 锁住在线的项轻探失败后进入「刚断开」，5 分钟内不落成离线，满 5 分钟后落成离线；
// 正在服务工单的项不被轻探，并且视为在线。
func TestRecentlyLostTiming(t *testing.T) {
	var broken atomic.Bool
	working := map[string]int{}
	f := newFixture(t, func(o *Options) {
		dial := pipeDial(acptest.Script{})
		o.Dial = func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
			if broken.Load() {
				return nil, errors.New("connection refused")
			}
			return dial(ctx, opts)
		}
		o.Working = func() map[string]int { return working }
	})
	id := f.withCustom(t, "fake")
	if v, _ := f.m.Probe(context.Background(), id); v.State != StateOnline {
		t.Fatalf("start %s", v.State)
	}
	broken.Store(true)
	f.m.CheckHealth(context.Background())
	if v, _ := f.m.Get(id); v.State != StateRecentlyLost || v.Detail != "connection refused" {
		t.Fatalf("after loss %s %q", v.State, v.Detail)
	}
	f.clock.add(4 * time.Minute)
	f.m.Settle()
	if v, _ := f.m.Get(id); v.State != StateRecentlyLost {
		t.Fatalf("settled early: %s", v.State)
	}
	f.clock.add(time.Minute)
	f.m.Settle()
	if v, _ := f.m.Get(id); v.State != StateOffline {
		t.Fatalf("not offline: %s", v.State)
	}
	working[id] = 1
	if v, _ := f.m.Get(id); v.State != StateOnline || v.Working != 1 {
		t.Fatalf("working buddy %s", v.State)
	}
	broken.Store(false)
	f.m.CheckHealth(context.Background())
	working[id] = 0
	if v, _ := f.m.Get(id); v.State != StateOffline {
		t.Fatalf("health check touched a working buddy: %s", v.State)
	}
	f.m.CheckHealth(context.Background())
	if v, _ := f.m.Get(id); v.State != StateOnline {
		t.Fatalf("light probe did not recover via full probe: %s", v.State)
	}
}

// TestNoteRun 锁住工人跑完一轮成功就算在线，要求登录时记成 needs_auth。
func TestNoteRun(t *testing.T) {
	f := newFixture(t, nil)
	id := f.withCustom(t, "fake")
	f.m.NoteRun(id, nil)
	if v, _ := f.m.Get(id); v.State != StateOnline {
		t.Fatalf("after run %s", v.State)
	}
	f.m.NoteAuthFailure(id, "token expired")
	if v, _ := f.m.Get(id); v.State != StateNeedsAuth || v.Detail != "token expired" {
		t.Fatalf("after auth failure %s %q", v.State, v.Detail)
	}
}

// TestModelsFromConfigOptions 锁住 session/new 没给 models 时，从 category 为 model 的设置里取模型，分组的取值也能摊平。
func TestModelsFromConfigOptions(t *testing.T) {
	sess := acp.NewSessionResponse{SessionID: "s", ConfigOptions: []acp.SessionConfigOption{
		{ID: "effort", Category: "thought_level", Options: []byte(`[{"value":"high","name":"High"}]`)},
		{ID: "model", Category: "model", CurrentValue: "b", Options: []byte(`[{"group":"fast","name":"Fast","options":[{"value":"a","name":"A"}]},{"group":"smart","name":"Smart","options":[{"value":"b","name":"B"}]}]`)},
	}}
	info := infoFrom(acp.InitializeResponse{}, &sess, true, time.Now())
	if len(info.Models) != 2 || info.Models[0].ModelID != "a" || info.Models[1].Name != "B" || info.CurrentModel != "b" || len(info.Options) != 2 {
		t.Fatalf("info %+v", info)
	}
	flat := acp.SessionConfigOption{Options: []byte(`[{"value":"x","name":"X"}]`)}
	if v := flat.Values(); len(v) != 1 || v[0].Value != "x" {
		t.Fatalf("flat %+v", v)
	}
}

// TestSanitizeSVG 锁住图标清洗：脚本、外部引用、事件属性和 style 被去掉，非 svg 根元素和过大的图标被拒。
func TestSanitizeSVG(t *testing.T) {
	raw := `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 16 16" onload="alert(1)" style="x">` +
		`<script>alert(1)</script><foreignObject><div>hi</div></foreignObject>` +
		`<path d="M0 0h16v16H0z" fill="currentColor" onclick="x()"/><use xlink:href="#a"/><rect width="4" height="4" fill="url(https://evil/x)"/></svg>`
	out, err := sanitizeSVG([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"script", "alert", "foreignObject", "onload", "onclick", "style", "use", "xlink", "evil", "url("} {
		if strings.Contains(s, bad) {
			t.Errorf("kept %s: %s", bad, s)
		}
	}
	if !strings.Contains(s, `d="M0 0h16v16H0z"`) || !strings.Contains(s, `viewBox="0 0 16 16"`) || !strings.HasPrefix(s, `<svg xmlns="http://www.w3.org/2000/svg"`) {
		t.Fatalf("lost geometry: %s", s)
	}
	if _, err := sanitizeSVG([]byte(`<html><svg/></html>`)); err == nil {
		t.Fatal("non-svg root accepted")
	}
	big := `<svg>` + strings.Repeat(`<path d="M0 0h16v16H0z"/>`, 400) + `</svg>`
	if _, err := sanitizeSVG([]byte(big)); err == nil {
		t.Fatal("oversized icon accepted")
	}
}
