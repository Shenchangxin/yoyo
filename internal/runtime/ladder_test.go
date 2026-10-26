// 已连接的邮件账户挡住对应站点的 browser_open。其它主机放行。

package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connector"
)

// TestLadderBlocksBrowserWhenMailConnected 确认已连接且允许的 gmail 账户下，browser_open mail.google.com 被拒绝，example.com 放行。
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
