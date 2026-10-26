// 供应商目录和 OAuth 授权地址。客户端密钥只出现在调用参数里，不写入 connectors.json。

package connector

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// CatalogEntry 是目录里的一个供应商，含种类、显示名和可选的授权地址。
type CatalogEntry struct {
	Provider string `json:"provider"`
	Kind     Kind   `json:"kind"`
	Label    string `json:"label"`
	AuthURL  string `json:"auth_url,omitempty"`
	MCPPack  bool   `json:"mcp_pack,omitempty"`
}

// Catalog 返回内置供应商列表。wecom 和 wechat 没有 AuthURL，MCPPack 为真。
func Catalog() []CatalogEntry {
	return []CatalogEntry{
		{Provider: "gmail", Kind: KindMail, Label: "Gmail", AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
		{Provider: "outlook", Kind: KindMail, Label: "Outlook", AuthURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"},
		{Provider: "gcal", Kind: KindCalendar, Label: "Google Calendar", AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
		{Provider: "feishu", Kind: KindIM, Label: "Feishu / 飞书", AuthURL: "https://open.feishu.cn/open-apis/authen/v1/index", MCPPack: true},
		{Provider: "dingtalk", Kind: KindIM, Label: "DingTalk / 钉钉", AuthURL: "https://login.dingtalk.com/oauth2/auth", MCPPack: true},
		{Provider: "wecom", Kind: KindIM, Label: "WeCom / 企业微信", MCPPack: true},
		{Provider: "wechat", Kind: KindIM, Label: "WeChat assistant (MCP-pack)", MCPPack: true},
		{Provider: "slack", Kind: KindIM, Label: "Slack", AuthURL: "https://slack.com/oauth/v2/authorize"},
		{Provider: "notion", Kind: KindDrive, Label: "Notion", AuthURL: "https://api.notion.com/v1/oauth/authorize", MCPPack: true},
		{Provider: "github", Kind: KindDrive, Label: "GitHub", AuthURL: "https://github.com/login/oauth/authorize", MCPPack: true},
		{Provider: "local", Kind: KindMail, Label: "Local mailbox (IMAP/MCP)"},
	}
}

// AuthURL 为 gmail、gcal、outlook、slack 拼出授权地址，并返回随机 state。其他供应商返回错误，说明只走 MCP 包。
// provider 转成小写。gmail 或 gcal 拼 Google 的授权地址，outlook 拼微软的，slack 拼 Slack 的。其他值返回 MCP-pack only，错误文案带上 provider。
// clientID 放进查询参数 client_id。slack 只放 client_id、state 和 scope。
// redirect 在 gmail、gcal 和 outlook 时放进 redirect_uri。slack 不使用它。
func (b *Broker) AuthURL(provider, clientID, redirect string) (string, string, error) {
	state := nonce()
	var raw string
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		q := url.Values{
			"client_id":     {clientID},
			"redirect_uri":  {redirect},
			"response_type": {"code"},
			"scope":         {"https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.send https://www.googleapis.com/auth/gmail.compose https://www.googleapis.com/auth/calendar.readonly https://www.googleapis.com/auth/calendar.events"},
			"state":         {state},
			"access_type":   {"offline"},
		}
		raw = "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
	case "outlook":
		q := url.Values{
			"client_id":     {clientID},
			"redirect_uri":  {redirect},
			"response_type": {"code"},
			"scope":         {"offline_access Mail.Read Mail.ReadWrite Mail.Send Calendars.Read Calendars.ReadWrite"},
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

// nonce 返回 8 字节随机数的十六进制。
func nonce() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// LooksExfilDraft 在正文或收件人含 webhook.site 或 requestbin 时返回真。@ 至少 8 个且含 http:// 或 https:// 时也返回真。
// to 和 body 用空格拼起来再转小写。含 webhook.site 或 requestbin 时返回真。@ 至少 8 个且含 http:// 或 https:// 时也返回真。
// body 和 to 用空格拼起来再转小写，按同一套条件判断。
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
