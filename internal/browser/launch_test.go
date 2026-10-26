// 本文件锁住有头窗口死掉后改回隔离 Chrome、启动参数里的调试端口，以及 DevToolsActivePort 的读取和超时。
// 测试里的 9333 只是端口文件的样例数字。由 go test 调用。

package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHeadedForAutomationDropsDeadTakeover 锁住活着的有头窗口保持有头，死掉的有头改回无头，本来的无头保持无头。
func TestHeadedForAutomationDropsDeadTakeover(t *testing.T) {
	if !headedForAutomation(true, true) {
		t.Fatal("live takeover window must stay headed")
	}
	if headedForAutomation(true, false) {
		t.Fatal("dead headed profile must fall back to isolated Chrome")
	}
	if headedForAutomation(false, false) {
		t.Fatal("isolated stays isolated")
	}
}

// TestChromeLaunchArgsIncludeDebugAllowlist 锁住无头参数含 --remote-allow-origins=*、调试端口和 --headless=new，有头参数不开 headless、带 --new-window，并且不设置 --remote-debugging-address。
func TestChromeLaunchArgsIncludeDebugAllowlist(t *testing.T) {
	args := chromeLaunchArgs(`C:\tmp\profile`, 0, false, "")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--remote-allow-origins=*") {
		t.Fatalf("missing remote-allow-origins: %s", joined)
	}
	if !strings.Contains(joined, "--remote-debugging-port=0") {
		t.Fatalf("expected ephemeral debug port: %s", joined)
	}
	if !strings.Contains(joined, "--headless=new") {
		t.Fatalf("headless missing: %s", joined)
	}
	if !strings.HasSuffix(joined, "about:blank") {
		t.Fatalf("expected about:blank start url: %s", joined)
	}
	headed := strings.Join(chromeLaunchArgs(`C:\tmp\profile`, 0, true, "http://127.0.0.1:9/game.html"), " ")
	if strings.Contains(headed, "--headless") {
		t.Fatalf("headed launch must not be headless: %s", headed)
	}
	if !strings.Contains(headed, "--new-window") {
		t.Fatalf("headed launch should open a window: %s", headed)
	}
	if !strings.Contains(headed, "http://127.0.0.1:9/game.html") {
		t.Fatalf("headed launch should open the served file: %s", headed)
	}
	if !strings.Contains(headed, "--hide-crash-restore-bubble") {
		t.Fatalf("headed launch should skip session restore chrome: %s", headed)
	}
	if strings.Contains(headed, "--remote-debugging-address=") {
		t.Fatalf("debugging-address can prevent DevToolsActivePort on Windows: %s", headed)
	}
}

// TestReadDevToolsActivePort 锁住端口文件第一行 9333 能读成整数端口。
func TestReadDevToolsActivePort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DevToolsActivePort")
	if err := os.WriteFile(path, []byte("9333\n/devtools/browser/abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	port, err := readDevToolsActivePort(path)
	if err != nil || port != 9333 {
		t.Fatalf("port=%d err=%v", port, err)
	}
}

// TestWaitDevToolsPortTimesOut 锁住没有端口文件时等待会超时，错误里含有 chrome debug port did not come up。
func TestWaitDevToolsPortTimesOut(t *testing.T) {
	dir := t.TempDir()
	_, err := waitDevToolsPort(dir, time.Now().Add(40*time.Millisecond))
	if err == nil || !strings.Contains(err.Error(), "chrome debug port did not come up") {
		t.Fatalf("%v", err)
	}
}

// TestIsCDPGone 锁住关闭的网络连接算断开，协议层的非法 URL 不算。
func TestIsCDPGone(t *testing.T) {
	if !isCDPGone(fmt.Errorf("failed to write JSON message: failed to marshal JSON: failed to write msg: failed to write frame: use of closed network connection")) {
		t.Fatal("expected gone")
	}
	if isCDPGone(fmt.Errorf("cdp Page.navigate: Cannot navigate to invalid URL")) {
		t.Fatal("protocol error is not a dead socket")
	}
}

// TestWaitChromeDebugUsesHintPortWithoutPortFile 锁住 hint 端口已经在听 /json/version 时，不必有 DevToolsActivePort 文件。
func TestWaitChromeDebugUsesHintPortWithoutPortFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://127.0.0.1/devtools"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	got, err := waitChromeDebug(t.TempDir(), port, time.Now().Add(time.Second))
	if err != nil || got != port {
		t.Fatalf("got=%d err=%v", got, err)
	}
}

// TestWaitChromeDebugIgnoresStalePortFile 锁住文件里的端口没在听时，等待仍会超时。
func TestWaitChromeDebugIgnoresStalePortFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DevToolsActivePort")
	if err := os.WriteFile(path, []byte("65534\n/devtools/browser/abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := waitChromeDebug(dir, 0, time.Now().Add(250*time.Millisecond))
	if err == nil || !strings.Contains(err.Error(), "chrome debug port did not come up") {
		t.Fatalf("%v", err)
	}
}
