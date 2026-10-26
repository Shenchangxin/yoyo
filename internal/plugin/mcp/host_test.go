// 检查协议版本，以及一行 JSON 和 Content-Length 两种读帧。

package mcp

import (
	"bufio"
	"fmt"
	"io"
	"testing"
)

// TestProtocolVersion 确认 ProtocolVersion 等于 2025-11-25。
func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion != "2025-11-25" {
		t.Fatalf("%s", ProtocolVersion)
	}
}

// TestReadJSONLine 确认以 '{' 开头的一行 JSON 能读出 id 1。
func TestReadJSONLine(t *testing.T) {
	r, w := io.Pipe()
	s := &Server{stdout: bufio.NewReader(r)}
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"ok\":true}}\n")
	}()
	res, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != 1 {
		t.Fatalf("%+v", res)
	}
}

// TestReadContentLength 确认 Content-Length 帧能读出 id 2。
func TestReadContentLength(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":2,"result":{}}`
	r, w := io.Pipe()
	s := &Server{stdout: bufio.NewReader(r)}
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
	}()
	res, err := s.read()
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != 2 {
		t.Fatalf("%+v", res)
	}
}
