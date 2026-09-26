package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connector"
)

func TestLadderBlocksBrowserWhenMailConnected(t *testing.T) {
	b, err := connector.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	acct := b.Connect(connector.Account{Provider: "gmail", Kind: connector.KindMail, Label: "mail"})
	tools := &WorkspaceTools{Connectors: b, AllowedConnectors: []string{acct.ID}}
	err = tools.ladderBlock("browser_open", `{"url":"https://mail.google.com/mail"}`)
	if err == nil || !strings.Contains(err.Error(), "connector") {
		t.Fatalf("want ladder block, got %v", err)
	}
	if err := tools.ladderBlock("browser_open", `{"url":"https://example.com"}`); err != nil {
		t.Fatal(err)
	}
}
