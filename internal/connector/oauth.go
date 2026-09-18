package connector

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

type CatalogEntry struct {
	Provider string `json:"provider"`
	Kind     Kind   `json:"kind"`
	Label    string `json:"label"`
	AuthURL  string `json:"auth_url,omitempty"`
}

func Catalog() []CatalogEntry {
	return []CatalogEntry{
		{Provider: "gmail", Kind: KindMail, Label: "Gmail", AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
		{Provider: "outlook", Kind: KindMail, Label: "Outlook", AuthURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"},
		{Provider: "feishu", Kind: KindIM, Label: "Feishu / 飞书", AuthURL: "https://open.feishu.cn/open-apis/authen/v1/index"},
		{Provider: "dingtalk", Kind: KindIM, Label: "DingTalk / 钉钉", AuthURL: "https://login.dingtalk.com/oauth2/auth"},
		{Provider: "wecom", Kind: KindIM, Label: "WeCom / 企业微信"},
		{Provider: "slack", Kind: KindIM, Label: "Slack", AuthURL: "https://slack.com/oauth/v2/authorize"},
		{Provider: "notion", Kind: KindDrive, Label: "Notion", AuthURL: "https://api.notion.com/v1/oauth/authorize"},
		{Provider: "github", Kind: KindDrive, Label: "GitHub", AuthURL: "https://github.com/login/oauth/authorize"},
		{Provider: "local", Kind: KindMail, Label: "Local mailbox (IMAP/MCP)"},
	}
}

func (b *Broker) AuthURL(provider, clientID, redirect string) (string, string, error) {
	state := nonce()
	var raw string
	switch strings.ToLower(provider) {
	case "gmail":
		q := url.Values{
			"client_id":     {clientID},
			"redirect_uri":  {redirect},
			"response_type": {"code"},
			"scope":         {"https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose"},
			"state":         {state},
			"access_type":   {"offline"},
		}
		raw = "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
	case "outlook":
		q := url.Values{
			"client_id":     {clientID},
			"redirect_uri":  {redirect},
			"response_type": {"code"},
			"scope":         {"offline_access Mail.Read Mail.ReadWrite Calendars.Read"},
			"state":         {state},
		}
		raw = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?" + q.Encode()
	case "slack":
		q := url.Values{"client_id": {clientID}, "state": {state}, "scope": {"channels:read,chat:write"}}
		raw = "https://slack.com/oauth/v2/authorize?" + q.Encode()
	default:
		return "", "", fmt.Errorf("connector: oauth for %s is MCP-pack only", provider)
	}
	return raw, state, nil
}

func nonce() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func LooksExfilDraft(to, body string) bool {
	blob := strings.ToLower(to + " " + body)
	if strings.Contains(blob, "webhook.site") || strings.Contains(blob, "requestbin") {
		return true
	}
	if strings.Count(blob, "@") >= 8 && (strings.Contains(blob, "http://") || strings.Contains(blob, "https://")) {
		return true
	}
	return false
}
