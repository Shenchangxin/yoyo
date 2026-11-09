package mcpserve

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestBridgeHelperProcess 不是真测试：YOYO_MCP_BRIDGE_HELPER=1 时把本进程当成桥，像 main 在 YOYO_MCP_BRIDGE=1 下那样运行，供其他测试启动。
func TestBridgeHelperProcess(t *testing.T) {
	if os.Getenv("YOYO_MCP_BRIDGE_HELPER") != "1" {
		return
	}
	if err := RunBridge(os.Stdin, os.Stdout, os.Getenv("YOYO_MCP_URL"), os.Getenv("YOYO_MCP_TOKEN")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

// helperArgs 是启动桥进程的参数：只跑 TestBridgeHelperProcess。
var helperArgs = []string{"-test.run=^TestBridgeHelperProcess$"}

// lines 把桥写出的每一行解成一条响应。
func lines(t *testing.T, out string) []rpcOut {
	t.Helper()
	var got []rpcOut
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			got = append(got, parse(t, []byte(line)))
		}
	}
	return got
}

// TestBridgeRoundTripsOverStdio 锁住桥在独立进程里把按行的请求转给控制面：握手拿到会话号、通知不回、
// 工具清单和调用结果按行回来；stdin 关掉后进程正常退出，并结束自己的会话。
func TestBridgeRoundTripsOverStdio(t *testing.T) {
	s, token := serveEcho(t)
	cmd := exec.Command(os.Args[0], helperArgs...)
	cmd.Env = append(os.Environ(), "YOYO_MCP_BRIDGE_HELPER=1", "YOYO_MCP_URL="+s.URL(), "YOYO_MCP_TOKEN="+token)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	watchdog := time.AfterFunc(30*time.Second, func() { _ = cmd.Process.Kill() })
	defer watchdog.Stop()
	out := bufio.NewReader(stdout)
	exchange := func(line string) rpcOut {
		t.Helper()
		if _, err := io.WriteString(stdin, line+"\n"); err != nil {
			t.Fatal(err)
		}
		reply, err := out.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read after %s: %v (stderr %s)", line, err, stderr.String())
		}
		return parse(t, bytes.TrimSpace(reply))
	}

	init := exchange(rpcBody(1, "initialize", map[string]any{"protocolVersion": "2025-03-26"}))
	if string(init.ID) != "1" || !strings.Contains(string(init.Result), `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("initialize = %+v", init)
	}
	s.mu.Lock()
	sessions := len(s.sessions)
	s.mu.Unlock()
	if sessions != 1 {
		t.Fatalf("sessions after initialize = %d", sessions)
	}
	if _, err := io.WriteString(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n\n"); err != nil {
		t.Fatal(err)
	}
	list := exchange(rpcBody(2, "tools/list", nil))
	if string(list.ID) != "2" || !strings.Contains(string(list.Result), "orchestrator_follow") {
		t.Fatalf("tools/list = %+v", list)
	}
	call := exchange(rpcBody(3, "tools/call", map[string]any{"name": "orchestrator_report", "arguments": map[string]any{"state": "ready"}}))
	if text, isErr := callText(t, call.Result); text != `orchestrator_report {"state":"ready"}` || isErr {
		t.Fatalf("tools/call = %q %v", text, isErr)
	}

	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if rest, _ := io.ReadAll(out); len(bytes.TrimSpace(rest)) != 0 {
		t.Fatalf("unexpected output after the last reply: %s", rest)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("bridge exit: %v (stderr %s)", err, stderr.String())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sessions) != 0 {
		t.Fatalf("session left open after stdin closed: %d", len(s.sessions))
	}
}

// TestBridgeAnswersFailedRequests 锁住控制面拒绝时，桥给请求回一条带同一 id 的 JSON-RPC 错误，给通知什么都不回，然后接着读下一条。
func TestBridgeAnswersFailedRequests(t *testing.T) {
	s, _ := serveEcho(t)
	in := strings.Join([]string{
		rpcBody(7, "tools/list", nil),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"x8","method":"ping"}`,
	}, "\n")
	var out bytes.Buffer
	if err := RunBridge(strings.NewReader(in), &out, s.URL(), "not-a-registered-token"); err != nil {
		t.Fatal(err)
	}
	got := lines(t, out.String())
	if len(got) != 2 || string(got[0].ID) != "7" || string(got[1].ID) != `"x8"` {
		t.Fatalf("replies = %s", out.String())
	}
	for _, r := range got {
		if r.Error == nil || r.Error.Code != codeInternal || !strings.Contains(r.Error.Message, "401") {
			t.Fatalf("reply = %s", out.String())
		}
	}
}

// TestBridgeReadsContentLengthFrames 锁住桥也收 Content-Length 分帧的消息，回复仍是一行一条，并且和按行的消息可以混着来。
func TestBridgeReadsContentLengthFrames(t *testing.T) {
	s, token := serveEcho(t)
	ping, list := rpcBody(1, "ping", nil), rpcBody(2, "tools/list", nil)
	in := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(ping), ping) +
		fmt.Sprintf("Content-Length: %d\r\nContent-Type: application/json\r\n\r\n%s", len(list), list) +
		"\n" + rpcBody(3, "ping", nil)
	var out bytes.Buffer
	if err := RunBridge(strings.NewReader(in), &out, s.URL(), token); err != nil {
		t.Fatal(err)
	}
	got := lines(t, out.String())
	if len(got) != 3 || string(got[0].Result) != "{}" || !strings.Contains(string(got[1].Result), "orchestrator_report") || string(got[2].ID) != "3" {
		t.Fatalf("replies = %s", out.String())
	}
	if strings.Count(out.String(), "\n") != 3 {
		t.Fatalf("not one line per reply: %q", out.String())
	}
}

// TestBridgeSurvivesAnUnreachableServer 锁住控制面不在时，桥给请求回错误而不是卡住或退出。
func TestBridgeSurvivesAnUnreachableServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + ln.Addr().String() + "/mcp"
	_ = ln.Close()
	var out bytes.Buffer
	if err := RunBridge(strings.NewReader(rpcBody(1, "ping", nil)+"\n"), &out, endpoint, NewToken()); err != nil {
		t.Fatal(err)
	}
	got := lines(t, out.String())
	if len(got) != 1 || got[0].Error == nil || !strings.Contains(got[0].Error.Message, "unreachable") {
		t.Fatalf("replies = %s", out.String())
	}
}

// TestBridgeOnlyTalksToLoopback 锁住桥拒绝不是本机回环地址上的 http 地址，也拒绝空 token，这时什么都不发也不写。
func TestBridgeOnlyTalksToLoopback(t *testing.T) {
	for _, tc := range []struct{ endpoint, token string }{
		{"http://example.com/mcp", "t"},
		{"https://127.0.0.1:1/mcp", "t"},
		{"http://127.0.0.1.example.com:1/mcp", "t"},
		{"127.0.0.1:1/mcp", "t"},
		{"", "t"},
		{"http://127.0.0.1:1/mcp", " "},
	} {
		var out bytes.Buffer
		if err := RunBridge(strings.NewReader(rpcBody(1, "ping", nil)+"\n"), &out, tc.endpoint, tc.token); err == nil || out.Len() != 0 {
			t.Fatalf("%+v: err %v, wrote %q", tc, err, out.String())
		}
	}
	var out bytes.Buffer
	if err := RunBridge(strings.NewReader(""), &out, "http://localhost:1/mcp", "t"); err != nil || out.Len() != 0 {
		t.Fatalf("empty stdin: %v %q", err, out.String())
	}
}

// TestReadFrameRejectsOversizedFrames 锁住 Content-Length 超过上限或不是数字时，readFrame 报错而不是去读那么多字节。
func TestReadFrameRejectsOversizedFrames(t *testing.T) {
	for _, head := range []string{fmt.Sprintf("Content-Length: %d", MaxBody+1), "Content-Length: lots"} {
		r := bufio.NewReader(strings.NewReader(head + "\r\n\r\n{}"))
		if _, err := readFrame(r); err == nil {
			t.Fatalf("%s accepted", head)
		}
	}
	var msg map[string]any
	r := bufio.NewReader(strings.NewReader("\n\n" + `{"a":1}`))
	frame, err := readFrame(r)
	if err != nil || json.Unmarshal(frame, &msg) != nil || msg["a"] != float64(1) {
		t.Fatalf("frame %q err %v", frame, err)
	}
	if _, err := readFrame(r); err != io.EOF {
		t.Fatalf("after the last frame: %v", err)
	}
}
