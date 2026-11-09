// rpc_buddy_test.go 锁住 buddy.* 经 Dispatch 的行为：密钥只进 vault，configure、get、log.tail 和 config.get 的应答里都没有它的值；
// 改了设置会发 state.changed；像密钥的普通变量被拒。测试换成只写文件的 vault，不碰系统钥匙串；不调 buddy.list，免得在测试里检测本机。

package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/vault"
)

// TestRPCBuddyNeverReturnsSecrets 锁住 buddy.configure 存下的密钥不出现在任何应答里，只有 hasKey；值在 vault 的 buddy.<id>.<NAME>。
func TestRPCBuddyNeverReturnsSecrets(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.Vault = vault.NewFileOnly(t.TempDir())
	call := func(method, params string) RPCResponse {
		return Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: json.RawMessage(params)})
	}
	const secret = "sk-ant-test-secret-0123456789"
	before := a.StateEpoch()
	res := call("buddy.configure", `{"id":"claude","maxParallel":3,"secrets":{"ANTHROPIC_API_KEY":"`+secret+`"}}`)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if a.StateEpoch() == before {
		t.Fatal("buddy.configure did not publish state")
	}
	got := call("buddy.get", `{"id":"claude"}`)
	for _, r := range []RPCResponse{res, got, call("config.get", `{}`), call("buddy.log.tail", `{"id":"claude"}`)} {
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		raw, _ := json.Marshal(r)
		if strings.Contains(string(raw), secret) {
			t.Fatalf("secret in response %s", raw)
		}
	}
	v, ok := got.Result.(buddy.View)
	if !ok || v.Config.MaxParallel != 3 || len(v.Config.Secrets) == 0 || !v.Config.Secrets[0].HasKey {
		t.Fatalf("view %+v", got.Result)
	}
	if stored, _ := a.Vault.GetStored("buddy.claude.ANTHROPIC_API_KEY"); stored != secret {
		t.Fatal("secret not in the vault")
	}
	if a.Config.Buddies.Items["claude"].MaxParallel != 3 || len(a.Config.Buddies.Items["claude"].Secrets) != 1 {
		t.Fatalf("config %+v", a.Config.Buddies.Items["claude"])
	}
	if bad := call("buddy.configure", `{"id":"claude","env":{"MY_TOKEN":"x"}}`); bad.Error == nil {
		t.Fatal("secret-like env accepted")
	}
	if missing := call("buddy.get", `{"id":"nobody"}`); missing.Error == nil {
		t.Fatal("unknown buddy returned")
	}
}

// TestRPCBuddyMethodsAreMutations 锁住 buddy.list、buddy.get、buddy.log.tail 是读，其余 buddy.* 的重绘主题是 buddy。
func TestRPCBuddyMethodsAreMutations(t *testing.T) {
	for _, m := range []string{"buddy.list", "buddy.get", "buddy.log.tail"} {
		if _, ok := app.RPCTopic(m); ok {
			t.Fatalf("%s is a read", m)
		}
	}
	for _, m := range []string{"buddy.install", "buddy.configure", "buddy.prefs", "buddy.refresh", "buddy.probe"} {
		if topic, ok := app.RPCTopic(m); !ok || topic != "buddy" {
			t.Fatalf("%s => %s %v", m, topic, ok)
		}
	}
}
