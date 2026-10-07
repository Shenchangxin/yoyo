package runtime

import "testing"

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
