// 基址去掉尾部空白。客户端不设置 Timeout。TLS 握手超时至少 30 秒。

package runtime

import (
	"net/http"
	"testing"
	"time"
)

// TestNormalizeBaseURL 确认基址去掉尾部空格后与断言相同。
func TestNormalizeBaseURL(t *testing.T) {
	got := NormalizeBaseURL("https://server.flowyaipc.com/claw/v1 ")
	if got != "https://server.flowyaipc.com/claw/v1" {
		t.Fatalf("got %q", got)
	}
	c := NewOpenAIClient("https://server.flowyaipc.com/claw/v1 ", " tok ")
	if c.BaseURL != "https://server.flowyaipc.com/claw/v1" || c.APIKey != "tok" {
		t.Fatalf("%+v", c)
	}
}

// TestOpenAIHTTPClientTLSHandshake 确认 NewOpenAIClient 去掉密钥空白，且 TLSHandshakeTimeout 至少 30 秒。
func TestOpenAIHTTPClientTLSHandshake(t *testing.T) {
	c := NewOpenAIClient("https://server.flowyaipc.com/claw/v1", "k")
	tr, ok := c.HTTPClient.Transport.(*http.Transport)
	if !ok || tr == nil {
		t.Fatal("expected http.Transport")
	}
	if tr.TLSHandshakeTimeout < 30*time.Second {
		t.Fatalf("TLSHandshakeTimeout=%v", tr.TLSHandshakeTimeout)
	}
}
