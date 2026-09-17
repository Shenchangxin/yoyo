package mcp

import (
	"bufio"
	"fmt"
	"io"
	"testing"
)

func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion != "2025-11-25" {
		t.Fatalf("%s", ProtocolVersion)
	}
}

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
