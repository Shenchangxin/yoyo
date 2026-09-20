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

func TestPreToolDoesNotScanWriteBodiesForFormat(t *testing.T) {
	args := `{"path":"api/export.go","content":"// The export format is JSON for clients."}`
	if deny, reason := PreTool("write_file", args); deny {
		t.Fatalf("write_file: %s", reason)
	}
	if LooksDelete("write_file", args, "") {
		t.Fatal("LooksDelete matched a Go comment")
	}
	if !LooksDelete("shell", "format c:", "") {
		t.Fatal("format c: must match")
	}
}
