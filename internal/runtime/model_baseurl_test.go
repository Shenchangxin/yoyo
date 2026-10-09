package runtime

import (
	"net/http"
	"testing"
	"time"
)

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
