// cursor_key_test.go 锁住 Cursor 的 API key 预检：被拒绝或没问成时不注入，探测记成需要登录并写明原因；通过时钥匙仍在。

package buddy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
)

// plantCursor 放一个假的托管安装，让探测走得到启动。
func plantCursor(t *testing.T, f *fixture) {
	t.Helper()
	dir := t.TempDir()
	cmd := filepath.Join(dir, "cursor-agent")
	if err := os.WriteFile(cmd, []byte("echo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.m.writeInstall("cursor", &InstallRecord{Kind: "binary", Version: "1", Command: cmd, Args: []string{"acp"}, Dir: dir}); err != nil {
		t.Fatal(err)
	}
	f.m.mu.Lock()
	e, ok := f.m.entryLocked("cursor")
	f.m.mu.Unlock()
	if !ok {
		t.Fatal("cursor missing from catalog")
	}
	f.m.detectOne(context.Background(), e)
}

// TestCursorDropsRejectedAPIKey 锁住兑换被拒绝的钥匙不会进进程，探测变成需要登录，并说明钥匙没被接受。
func TestCursorDropsRejectedAPIKey(t *testing.T) {
	const key = "sk-not-a-cursor-key"
	f := newFixture(t, func(o *Options) {
		o.CursorKeyCheck = func(context.Context, string) (bool, error) { return false, nil }
		o.Dial = func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
			if getEnv(opts.Env, cursorAPIKey) != "" {
				t.Errorf("rejected key was injected")
				return nil, context.DeadlineExceeded
			}
			return pipeDial(acptest.Script{NeedAuth: true, AuthMethods: []acp.AuthMethod{{ID: "cursor_login", Name: "Cursor Login"}}})(ctx, opts)
		}
	})
	f.vault.Set("buddy.cursor.CURSOR_API_KEY", key)
	plantCursor(t, f)
	v, err := f.m.Probe(context.Background(), "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if v.State != StateNeedsAuth || !strings.Contains(v.Detail, "API key") {
		t.Fatalf("state %s detail %q", v.State, v.Detail)
	}
}

// TestCursorDropsUnreachableAPIKey 锁住登录服务没问成时同样不注入，界面上说的是没问成，不是超时。
func TestCursorDropsUnreachableAPIKey(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.CursorKeyCheck = func(context.Context, string) (bool, error) { return false, errors.New("offline") }
		o.Dial = pipeDial(acptest.Script{NeedAuth: true})
	})
	f.vault.Set("buddy.cursor.CURSOR_API_KEY", "sk-offline")
	plantCursor(t, f)
	v, err := f.m.Probe(context.Background(), "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if v.State != StateNeedsAuth || !strings.Contains(v.Detail, "登录服务") || getEnv(mustLaunch(t, f).Env, cursorAPIKey) != "" {
		t.Fatalf("state %s detail %q", v.State, v.Detail)
	}
}

// TestCursorKeepsAcceptedAPIKey 锁住兑换通过的钥匙仍然注入，探测可以在线。
func TestCursorKeepsAcceptedAPIKey(t *testing.T) {
	const key = "crsr-accepted"
	f := newFixture(t, func(o *Options) {
		o.CursorKeyCheck = func(context.Context, string) (bool, error) { return true, nil }
		o.Dial = func(ctx context.Context, opts acp.Options) (*acp.Client, error) {
			if getEnv(opts.Env, cursorAPIKey) != key {
				t.Errorf("accepted key missing, env has %q", getEnv(opts.Env, cursorAPIKey))
			}
			return pipeDial(acptest.Script{Models: []acp.ModelInfo{{ModelID: "cursor"}}})(ctx, opts)
		}
	})
	f.vault.Set("buddy.cursor.CURSOR_API_KEY", key)
	plantCursor(t, f)
	v, err := f.m.Probe(context.Background(), "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if v.State != StateOnline || v.Detail != "" {
		t.Fatalf("state %s detail %q", v.State, v.Detail)
	}
}

// TestExchangeCursorKey 锁住兑换接口的三种回答：2xx 通过，401 拒绝，其它状态算没问成。
func TestExchangeCursorKey(t *testing.T) {
	var status int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Method != http.MethodPost {
			t.Errorf("request %s missing auth %v", r.Method, r.Header.Get("Authorization") == "")
		}
		w.WriteHeader(status)
	}))
	defer srv.Close()
	status = http.StatusOK
	ok, err := exchangeCursorKey(context.Background(), srv.Client(), srv.URL, "secret-key")
	if !ok || err != nil {
		t.Fatalf("accept ok %v err %v", ok, err)
	}
	status = http.StatusUnauthorized
	ok, err = exchangeCursorKey(context.Background(), srv.Client(), srv.URL, "secret-key")
	if ok || err != nil {
		t.Fatalf("reject ok %v err %v", ok, err)
	}
	status = http.StatusBadGateway
	ok, err = exchangeCursorKey(context.Background(), srv.Client(), srv.URL, "secret-key")
	if ok || err == nil {
		t.Fatalf("gateway ok %v err %v", ok, err)
	}
}

// mustLaunch 启动 cursor，失败即停。
func mustLaunch(t *testing.T, f *fixture) acp.Launch {
	t.Helper()
	l, err := f.m.Launch("cursor")
	if err != nil {
		t.Fatal(err)
	}
	return l
}
