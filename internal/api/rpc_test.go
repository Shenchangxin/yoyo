package api

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRPCHealthAndUnknown(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ok := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "health"})
	if ok.Error != nil {
		t.Fatal(ok.Error)
	}
	m, _ := ok.Result.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", ok.Result)
	}
	bad := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "nope"})
	if bad.Error == nil {
		t.Fatal("expected method error")
	}
}

func TestLineClientHealth(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	pr, pw := io.Pipe()
	qr, qw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ServeRPC(ctx, a, pr, qw) }()
	cli := NewLineClient(qr, pw)
	v, err := cli.Call("health", nil)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := v.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", v)
	}
}
