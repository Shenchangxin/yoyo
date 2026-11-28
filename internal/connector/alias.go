// alias.go 是代理人的邮件别名：工单代人发出去的信，主题带上 [yoyo:工单号]，回信地址是人自己邮箱的加号子地址
// （you+yoyo-工单号@域名，Gmail 和 Microsoft 365 都认）。对方回信时，读信那边按标记把回信交回原来的工单。
// 信仍然从人自己的账号发出，发之前照样要人批准；别名只决定回信回到哪里。

package connector

import (
	"encoding/json"
	"io"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
)

var (
	aliasSubject = regexp.MustCompile(`\[yoyo:([A-Za-z0-9_.-]{1,40})\]`)
	aliasAddress = regexp.MustCompile(`(?i)\+yoyo-([a-z0-9_.-]{1,40})@`)
	ticketChars  = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,40}$`)
)

// selfAddress 记着从供应商那里问到的账号地址，按账号编号。只在内存里。
var selfAddress sync.Map

// AliasTicket 从主题或收件地址里认出工单号。认不出时为空。
func AliasTicket(text string) string {
	if m := aliasSubject.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	if m := aliasAddress.FindStringSubmatch(text); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// AccountAddress 是账号自己的邮箱地址：先看连接时填的地址，再看名字，最后看从供应商问到的。都没有时为空。小写。
func AccountAddress(a Account) string {
	for _, v := range []string{a.Endpoint, a.Label} {
		if addr, err := mail.ParseAddress(strings.TrimSpace(v)); err == nil && strings.Contains(addr.Address, "@") {
			return strings.ToLower(addr.Address)
		}
	}
	if v, ok := selfAddress.Load(a.ID); ok {
		return v.(string)
	}
	return ""
}

// SubAddress 把 ticket 拼成 addr 的加号子地址。addr 自己已经带加号时换掉原来的。
func SubAddress(addr, ticket string) string {
	at := strings.LastIndex(addr, "@")
	if at <= 0 || !ticketChars.MatchString(ticket) {
		return ""
	}
	local := addr[:at]
	if i := strings.Index(local, "+"); i >= 0 {
		local = local[:i]
	}
	return local + "+yoyo-" + strings.ToLower(ticket) + addr[at:]
}

// withAlias 给工单发出去的信打上标记。日历、本地和没有工单的草稿不动。self 是账号地址，空时只改主题。
func withAlias(d Draft, a Account, self string) Draft {
	ticket := strings.TrimSpace(d.Ticket)
	provider := strings.ToLower(a.Provider)
	if !ticketChars.MatchString(ticket) || a.Kind == KindCalendar || provider == "gcal" || provider == "slack" {
		return d
	}
	tag := "[yoyo:" + ticket + "]"
	if !strings.Contains(d.Subject, tag) {
		d.Subject = strings.TrimSpace(d.Subject + " " + tag)
	}
	if d.Alias == "" && self != "" {
		d.Alias = SubAddress(self, ticket)
	}
	return d
}

// aliasSelf 是发信账号的地址。不知道时去供应商问一次（Gmail 的 profile、Graph 的 /me），问到了记在内存里。问不到时为空，只打主题标记。
func (b *Broker) aliasSelf(a Account) string {
	if addr := AccountAddress(a); addr != "" {
		return addr
	}
	provider := strings.ToLower(a.Provider)
	if provider != "gmail" && provider != "outlook" {
		return ""
	}
	tok, err := b.freshToken(a, false)
	if err != nil || tok.AccessToken == "" {
		return ""
	}
	u := "https://gmail.googleapis.com/gmail/v1/users/me/profile"
	if provider == "outlook" {
		u = "https://graph.microsoft.com/v1.0/me?$select=mail,userPrincipalName"
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	resp, err := b.client().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var doc struct {
		Email string `json:"emailAddress"`
		Mail  string `json:"mail"`
		UPN   string `json:"userPrincipalName"`
	}
	_ = json.Unmarshal(body, &doc)
	for _, v := range []string{doc.Email, doc.Mail, doc.UPN} {
		if addr, err := mail.ParseAddress(strings.TrimSpace(v)); err == nil {
			out := strings.ToLower(addr.Address)
			selfAddress.Store(a.ID, out)
			return out
		}
	}
	return ""
}

// outlookReplyTo 是 Graph 发信时的 replyTo。没有别名时是空列表。
func outlookReplyTo(alias string) []map[string]any {
	if strings.TrimSpace(alias) == "" {
		return []map[string]any{}
	}
	return []map[string]any{{"emailAddress": map[string]string{"address": alias}}}
}
