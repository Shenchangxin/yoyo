package safeguard

import "testing"

func TestLooksExfil(t *testing.T) {
	ok, _ := LooksExfil("connector_send", "https://webhook.site/abc", "", "")
	if !ok {
		t.Fatal("webhook.site")
	}
}

func TestLooksSecretPaste(t *testing.T) {
	if !LooksSecretPaste(`{"text":"sk-live-secret"}`) {
		t.Fatal("sk-")
	}
	deny, _ := PreTool("browser_type", `{"text":"-----BEGIN PRIVATE KEY-----"}`)
	if !deny {
		t.Fatal("pem paste")
	}
}
