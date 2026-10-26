// 用 TokenStore 里的访问令牌读写信件和日历。令牌不进 connectors.json。
// HTTP 429 和 5xx 的错误文本带 outcome_unknown。

package connector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TokenStore 按名字读写令牌字符串。名字是账号上的 VaultKey。
type TokenStore interface {
	Get(name string) (string, error)
	Set(name, value string)
}

// Token 是访问令牌、刷新令牌和过期时间。SaveToken 把它编成 JSON 交给 TokenStore。
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

// SetTokens 换上令牌仓库。
func (b *Broker) SetTokens(store TokenStore) {
	b.mu.Lock()
	b.tokens = store
	b.mu.Unlock()
}

// SetHTTP 换上 HTTP 客户端。测试用它替换传输。
func (b *Broker) SetHTTP(client *http.Client) {
	b.mu.Lock()
	b.http = client
	b.mu.Unlock()
}

// client 返回已设置的 HTTP 客户端，否则用 http.DefaultClient。
func (b *Broker) client() *http.Client {
	if b.http != nil {
		return b.http
	}
	return http.DefaultClient
}

// Exchange 用授权码向 Google 或微软换令牌。client secret 只放进这次表单，函数不把它写入 JSON。gmail、gcal、outlook 以外返回错误。
func (b *Broker) Exchange(provider, code, clientID, secret, redirect string) (Token, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {secret},
		"redirect_uri":  {redirect},
		"grant_type":    {"authorization_code"},
	}
	endpoint := ""
	switch provider {
	case "gmail":
		endpoint = "https://oauth2.googleapis.com/token"
	case "gcal":
		endpoint = "https://oauth2.googleapis.com/token"
	case "outlook":
		endpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	default:
		return Token{}, fmt.Errorf("connector: token exchange for %s is MCP-pack only", provider)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := b.client().Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return Token{}, fmt.Errorf("connector: token exchange %s: %s", resp.Status, truncate(string(body), 300))
	}
	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return Token{}, err
	}
	if tok.AccessToken == "" {
		return Token{}, fmt.Errorf("connector: empty access token")
	}
	if tok.Expiry.IsZero() {
		var extra struct {
			ExpiresIn int `json:"expires_in"`
		}
		_ = json.Unmarshal(body, &extra)
		if extra.ExpiresIn > 0 {
			tok.Expiry = time.Now().UTC().Add(time.Duration(extra.ExpiresIn) * time.Second)
		}
	}
	return tok, nil
}

// SaveToken 把令牌 JSON 存到 TokenStore 的 vaultKey 名下。仓库或名字为空时返回错误。
func (b *Broker) SaveToken(vaultKey string, tok Token) error {
	if b.tokens == nil || vaultKey == "" {
		return fmt.Errorf("connector: no token store")
	}
	raw, _ := json.Marshal(tok)
	b.tokens.Set(vaultKey, string(raw))
	return nil
}

// loadToken 从 TokenStore 读取。正文不是 { 开头时当成裸访问令牌。JSON 解析失败时也退回裸令牌。
func (b *Broker) loadToken(vaultKey string) (Token, error) {
	if b.tokens == nil || vaultKey == "" {
		return Token{}, fmt.Errorf("connector: no token")
	}
	raw, err := b.tokens.Get(vaultKey)
	if err != nil {
		return Token{}, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Token{}, fmt.Errorf("connector: empty token")
	}
	if !strings.HasPrefix(raw, "{") {
		return Token{AccessToken: raw}, nil
	}
	var tok Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return Token{AccessToken: raw}, nil
	}
	return tok, nil
}

// liveRead 用账号的 VaultKey 取令牌。日历种类或查询像日历时走日历接口，否则走邮件。其他供应商返回 MCP-pack only。
func (b *Broker) liveRead(a Account, query string) ([]map[string]string, error) {
	tok, err := b.loadToken(a.VaultKey)
	if err != nil {
		return nil, err
	}
	provider := strings.ToLower(a.Provider)
	if a.Kind == KindCalendar || looksCalendar(query) {
		switch provider {
		case "gmail", "gcal":
			return gcalList(b.client(), tok.AccessToken, query)
		case "outlook":
			return outlookEvents(b.client(), tok.AccessToken, query)
		}
	}
	switch provider {
	case "gmail", "gcal":
		return gmailList(b.client(), tok.AccessToken, query)
	case "outlook":
		return outlookList(b.client(), tok.AccessToken, query)
	default:
		return nil, fmt.Errorf("connector: live read for %s is MCP-pack only", a.Provider)
	}
}

// liveSend 对日历种类或 gcal 写日程，gmail 和 outlook 发信。其他供应商返回错误。
func (b *Broker) liveSend(a Account, d Draft) error {
	tok, err := b.loadToken(a.VaultKey)
	if err != nil {
		return err
	}
	provider := strings.ToLower(a.Provider)
	if a.Kind == KindCalendar || provider == "gcal" {
		return liveWriteEvent(b.client(), provider, tok.AccessToken, d)
	}
	switch provider {
	case "gmail":
		return gmailSend(b.client(), tok.AccessToken, d)
	case "outlook":
		return outlookSend(b.client(), tok.AccessToken, d)
	default:
		return fmt.Errorf("connector: live send for %s is MCP-pack only", a.Provider)
	}
}

// gmailList 列出最多 20 封 Gmail，并尽量补上主题和发件人。空列表返回一条 note。
func gmailList(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=20"
	if query != "" {
		u += "&q=" + url.QueryEscape(query)
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gmail list: %s", truncate(string(body), 300))
	}
	var doc struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(body, &doc)
	out := make([]map[string]string, 0, len(doc.Messages))
	for i, m := range doc.Messages {
		if i >= 20 {
			break
		}
		item := map[string]string{"id": m.ID, "provider": "gmail"}
		meta, err := gmailMeta(client, token, m.ID)
		if err == nil {
			for k, v := range meta {
				item[k] = v
			}
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "gmail", "note": "no messages", "query": query})
	}
	return out, nil
}

// gmailMeta 读取一封信的 Subject、From 和 snippet。
func gmailMeta(client *http.Client, token, id string) (map[string]string, error) {
	u := "https://gmail.googleapis.com/gmail/v1/users/me/messages/" + url.PathEscape(id) + "?format=metadata&metadataHeaders=Subject&metadataHeaders=From"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gmail meta")
	}
	var doc struct {
		Snippet string `json:"snippet"`
		Payload struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
	}
	_ = json.Unmarshal(body, &doc)
	out := map[string]string{"body": doc.Snippet}
	for _, h := range doc.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			out["subject"] = h.Value
		case "from":
			out["from"] = h.Value
		}
	}
	return out, nil
}

// gmailSend 把草稿编成 MIME，再以 base64 raw 调用 Gmail send。
func gmailSend(client *http.Client, token string, d Draft) error {
	raw := mimeMail(d)
	enc := base64.RawURLEncoding.EncodeToString([]byte(raw))
	payload, _ := json.Marshal(map[string]string{"raw": enc})
	req, _ := http.NewRequest(http.MethodPost, "https://gmail.googleapis.com/gmail/v1/users/me/messages/send", strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("gmail send", resp.StatusCode, string(body))
	}
	return nil
}

// outlookList 列出最多 20 封 Outlook 邮件的主题、发件人和预览。
func outlookList(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://graph.microsoft.com/v1.0/me/messages?$top=20&$select=subject,from,bodyPreview"
	if query != "" {
		u += "&$search=" + url.QueryEscape(query)
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("outlook list: %s", truncate(string(body), 300))
	}
	var doc struct {
		Value []struct {
			Subject     string `json:"subject"`
			BodyPreview string `json:"bodyPreview"`
			From        struct {
				EmailAddress struct {
					Address string `json:"address"`
				} `json:"emailAddress"`
			} `json:"from"`
		} `json:"value"`
	}
	_ = json.Unmarshal(body, &doc)
	out := make([]map[string]string, 0, len(doc.Value))
	for _, m := range doc.Value {
		out = append(out, map[string]string{
			"provider": "outlook",
			"subject":  m.Subject,
			"from":     m.From.EmailAddress.Address,
			"body":     m.BodyPreview,
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "outlook", "note": "no messages", "query": query})
	}
	return out, nil
}

// outlookSend 调用 Graph 的 sendMail。状态码大于等于 300 时返回错误。
func outlookSend(client *http.Client, token string, d Draft) error {
	payload, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"subject": d.Subject,
			"body":    map[string]string{"contentType": "Text", "content": d.Body},
			"toRecipients": []map[string]any{
				{"emailAddress": map[string]string{"address": d.To}},
			},
		},
	})
	req, _ := http.NewRequest(http.MethodPost, "https://graph.microsoft.com/v1.0/me/sendMail", strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("outlook send", resp.StatusCode, string(body))
	}
	return nil
}

// mimeMail 拼出文本邮件。有附件时用 multipart，单个附件最多读 8 MiB。
func mimeMail(d Draft) string {
	var hdr strings.Builder
	fmt.Fprintf(&hdr, "To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n", d.To, d.Subject)
	if d.Cc != "" {
		fmt.Fprintf(&hdr, "Cc: %s\r\n", d.Cc)
	}
	if d.Bcc != "" {
		fmt.Fprintf(&hdr, "Bcc: %s\r\n", d.Bcc)
	}
	if d.ReplyTo != "" {
		fmt.Fprintf(&hdr, "In-Reply-To: %s\r\nReferences: %s\r\n", d.ReplyTo, d.ReplyTo)
	}
	if len(d.Attachments) == 0 {
		hdr.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		hdr.WriteString(d.Body)
		return hdr.String()
	}
	const bound = "yoyo-mix"
	fmt.Fprintf(&hdr, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n--%s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n", bound, bound, d.Body)
	for _, p := range d.Attachments {
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			continue
		}
		if len(b) > 8<<20 {
			b = b[:8<<20]
		}
		fmt.Fprintf(&hdr, "--%s\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=%q\r\nContent-Transfer-Encoding: base64\r\n\r\n%s\r\n",
			bound, filepath.Base(p), wrap64(base64.StdEncoding.EncodeToString(b)))
	}
	fmt.Fprintf(&hdr, "--%s--\r\n", bound)
	return hdr.String()
}

// wrap64 把 base64 按 76 个字符折行。
func wrap64(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 76 {
		end := i + 76
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}

// liveWriteEvent 拒绝 recurrence。操作名含 delete 则删除，含 update 或已有 EventID 则修改，否则创建。
func liveWriteEvent(client *http.Client, provider, token string, d Draft) error {
	if strings.TrimSpace(d.Recurrence) != "" {
		return fmt.Errorf("connector: recurring series must be edited in the calendar app (422)")
	}
	op := strings.ToLower(strings.TrimSpace(d.Op))
	if strings.Contains(op, "delete") || op == "calendar.delete" {
		return liveDeleteEvent(client, provider, token, d)
	}
	if strings.Contains(op, "update") || op == "calendar.update" || d.EventID != "" {
		return livePatchEvent(client, provider, token, d)
	}
	return liveCreateEvent(client, provider, token, d)
}

// liveCreateEvent 按供应商调用 Google 或 Outlook 创建日程。
func liveCreateEvent(client *http.Client, provider, token string, d Draft) error {
	start, end := eventWindow(d)
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		return gcalInsert(client, token, d, start, end)
	case "outlook":
		return outlookCreateEvent(client, token, d, start, end)
	default:
		return fmt.Errorf("connector: calendar write for %s is MCP-pack only", provider)
	}
}

// livePatchEvent 要求 EventID 和 TargetVersion，再按供应商修改。
func livePatchEvent(client *http.Client, provider, token string, d Draft) error {
	if strings.TrimSpace(d.EventID) == "" || strings.TrimSpace(d.TargetVersion) == "" {
		return fmt.Errorf("connector: calendar update needs event_id and target_version")
	}
	start, end := eventWindow(d)
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		return gcalMutate(client, token, d, start, end, false)
	case "outlook":
		return outlookMutate(client, token, d, start, end, false)
	default:
		return fmt.Errorf("connector: calendar write for %s is MCP-pack only", provider)
	}
}

// liveDeleteEvent 要求 EventID 和 TargetVersion，再按供应商删除。
func liveDeleteEvent(client *http.Client, provider, token string, d Draft) error {
	if strings.TrimSpace(d.EventID) == "" || strings.TrimSpace(d.TargetVersion) == "" {
		return fmt.Errorf("connector: calendar delete needs event_id and target_version")
	}
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		return gcalMutate(client, token, d, time.Time{}, time.Time{}, true)
	case "outlook":
		return outlookMutate(client, token, d, time.Time{}, time.Time{}, true)
	default:
		return fmt.Errorf("connector: calendar write for %s is MCP-pack only", provider)
	}
}

// sendStatusError 对 412 说明事件已变。429 和 5xx 的错误文本含 outcome_unknown。
func sendStatusError(op string, code int, body string) error {
	if code == 412 {
		return fmt.Errorf("%s: HTTP %d the event changed; prepare a new action: %s", op, code, truncate(body, 300))
	}
	if code >= 500 || code == 429 {
		return fmt.Errorf("%s: HTTP %d outcome_unknown: %s", op, code, truncate(body, 300))
	}
	return fmt.Errorf("%s: HTTP %d: %s", op, code, truncate(body, 300))
}

// eventWindow 从 Start 或 To 解析开始时间，失败则用一小时后。结束时间不晚于开始时，全天加 24 小时，否则加 1 小时。
func eventWindow(d Draft) (time.Time, time.Time) {
	start := time.Now().UTC().Add(time.Hour).Truncate(time.Minute)
	for _, raw := range []string{d.Start, d.To} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			start = t
			break
		}
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			start = t
			break
		}
	}
	end := start.Add(time.Hour)
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(d.End)); err == nil {
		end = t
	} else if t, err := time.Parse("2006-01-02", strings.TrimSpace(d.End)); err == nil {
		end = t
	}
	if !end.After(start) {
		if d.AllDay {
			end = start.Add(24 * time.Hour)
		} else {
			end = start.Add(time.Hour)
		}
	}
	return start, end
}

// gcalInsert 向 Google Calendar 插入事件。日历 ID 为空时用 primary。
func gcalInsert(client *http.Client, token string, d Draft, start, end time.Time) error {
	tz := d.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	cal := d.CalendarID
	if cal == "" {
		cal = "primary"
	}
	payload := map[string]any{
		"summary":     d.Subject,
		"description": d.Body,
		"location":    d.Location,
	}
	if d.AllDay {
		payload["start"] = map[string]string{"date": start.Format("2006-01-02")}
		payload["end"] = map[string]string{"date": end.Format("2006-01-02")}
	} else {
		payload["start"] = map[string]string{"dateTime": start.Format(time.RFC3339), "timeZone": tz}
		payload["end"] = map[string]string{"dateTime": end.Format(time.RFC3339), "timeZone": tz}
	}
	var attendees []map[string]string
	if strings.Contains(d.To, "@") && !strings.Contains(d.To, "T") {
		attendees = append(attendees, map[string]string{"email": strings.TrimSpace(d.To)})
	}
	for _, a := range d.Attendees {
		a = strings.TrimSpace(a)
		if a != "" {
			attendees = append(attendees, map[string]string{"email": a})
		}
	}
	if len(attendees) > 0 {
		payload["attendees"] = attendees
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "https://www.googleapis.com/calendar/v3/calendars/"+url.PathEscape(cal)+"/events", strings.NewReader(string(raw)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("gcal insert", resp.StatusCode, string(body))
	}
	return nil
}

// gcalMutate 用 If-Match 带着 TargetVersion 修改或删除 Google 事件。
func gcalMutate(client *http.Client, token string, d Draft, start, end time.Time, del bool) error {
	cal := d.CalendarID
	if cal == "" {
		cal = "primary"
	}
	u := "https://www.googleapis.com/calendar/v3/calendars/" + url.PathEscape(cal) + "/events/" + url.PathEscape(d.EventID)
	method := http.MethodPatch
	var raw []byte
	if del {
		method = http.MethodDelete
	} else {
		payload := map[string]any{
			"summary":     d.Subject,
			"description": d.Body,
			"location":    d.Location,
		}
		tz := d.TimeZone
		if tz == "" {
			tz = "UTC"
		}
		if d.AllDay {
			payload["start"] = map[string]string{"date": start.Format("2006-01-02")}
			payload["end"] = map[string]string{"date": end.Format("2006-01-02")}
		} else if !start.IsZero() {
			payload["start"] = map[string]string{"dateTime": start.Format(time.RFC3339), "timeZone": tz}
			payload["end"] = map[string]string{"dateTime": end.Format(time.RFC3339), "timeZone": tz}
		}
		raw, _ = json.Marshal(payload)
	}
	req, _ := http.NewRequest(method, u, strings.NewReader(string(raw)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("If-Match", d.TargetVersion)
	if !del {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("gcal mutate", resp.StatusCode, string(body))
	}
	return nil
}

// outlookCreateEvent 调用 Graph 创建事件。
func outlookCreateEvent(client *http.Client, token string, d Draft, start, end time.Time) error {
	tz := d.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	payload := map[string]any{
		"subject":  d.Subject,
		"body":     map[string]string{"contentType": "Text", "content": d.Body},
		"location": map[string]string{"displayName": d.Location},
		"isAllDay": d.AllDay,
		"start":    map[string]string{"dateTime": start.UTC().Format("2006-01-02T15:04:05"), "timeZone": tz},
		"end":      map[string]string{"dateTime": end.UTC().Format("2006-01-02T15:04:05"), "timeZone": tz},
	}
	var attendees []map[string]any
	if strings.Contains(d.To, "@") && !strings.Contains(d.To, "T") {
		attendees = append(attendees, map[string]any{"emailAddress": map[string]string{"address": strings.TrimSpace(d.To)}, "type": "required"})
	}
	for _, a := range d.Attendees {
		a = strings.TrimSpace(a)
		if a != "" {
			attendees = append(attendees, map[string]any{"emailAddress": map[string]string{"address": a}, "type": "required"})
		}
	}
	if len(attendees) > 0 {
		payload["attendees"] = attendees
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "https://graph.microsoft.com/v1.0/me/events", strings.NewReader(string(raw)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("outlook event", resp.StatusCode, string(body))
	}
	return nil
}

// outlookMutate 用 If-Match 带着 TargetVersion 修改或删除 Outlook 事件。
func outlookMutate(client *http.Client, token string, d Draft, start, end time.Time, del bool) error {
	u := "https://graph.microsoft.com/v1.0/me/events/" + url.PathEscape(d.EventID)
	method := http.MethodPatch
	var raw []byte
	if del {
		method = http.MethodDelete
	} else {
		tz := d.TimeZone
		if tz == "" {
			tz = "UTC"
		}
		payload := map[string]any{
			"subject":  d.Subject,
			"body":     map[string]string{"contentType": "Text", "content": d.Body},
			"location": map[string]string{"displayName": d.Location},
			"isAllDay": d.AllDay,
		}
		if !start.IsZero() {
			payload["start"] = map[string]string{"dateTime": start.UTC().Format("2006-01-02T15:04:05"), "timeZone": tz}
			payload["end"] = map[string]string{"dateTime": end.UTC().Format("2006-01-02T15:04:05"), "timeZone": tz}
		}
		raw, _ = json.Marshal(payload)
	}
	req, _ := http.NewRequest(method, u, strings.NewReader(string(raw)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("If-Match", d.TargetVersion)
	if !del {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return sendStatusError("outlook mutate", resp.StatusCode, string(body))
	}
	return nil
}

// looksCalendar 在查询里出现 calendar、event、日程、会议、meetup 或 agenda 时返回真。
func looksCalendar(query string) bool {
	q := strings.ToLower(query)
	for _, n := range []string{"calendar", "event", "日程", "会议", "meetup", "agenda"} {
		if strings.Contains(q, n) {
			return true
		}
	}
	return false
}

// gcalList 列出主日历最多 20 个事件。查询里有两个 RFC3339 时间时用作 timeMin 和 timeMax。
func gcalList(client *http.Client, token, query string) ([]map[string]string, error) {
	timeMin := time.Now().UTC().Format(time.RFC3339)
	timeMax := ""
	if min, max, ok := ParseTimeRange(query); ok {
		timeMin = min.Format(time.RFC3339)
		timeMax = max.Format(time.RFC3339)
	}
	u := "https://www.googleapis.com/calendar/v3/calendars/primary/events?maxResults=20&singleEvents=true&orderBy=startTime&timeMin=" + url.QueryEscape(timeMin)
	if timeMax != "" {
		u += "&timeMax=" + url.QueryEscape(timeMax)
	}
	if query != "" && !looksCalendar(query) {
		if _, _, ok := ParseTimeRange(query); !ok {
			u += "&q=" + url.QueryEscape(query)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gcal list: %s", truncate(string(body), 300))
	}
	var doc struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			Location string `json:"location"`
		} `json:"items"`
	}
	_ = json.Unmarshal(body, &doc)
	out := make([]map[string]string, 0, len(doc.Items))
	for _, it := range doc.Items {
		start := it.Start.DateTime
		if start == "" {
			start = it.Start.Date
		}
		out = append(out, map[string]string{
			"provider": "gcal",
			"kind":     "calendar",
			"id":       it.ID,
			"subject":  it.Summary,
			"start":    start,
			"location": it.Location,
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "gcal", "kind": "calendar", "note": "no events", "query": query})
	}
	return out, nil
}

// outlookEvents 列出最多 20 个 Outlook 事件。查询不像日历且不是时间范围时加上 $search。
func outlookEvents(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://graph.microsoft.com/v1.0/me/events?$top=20&$select=subject,start,end,location"
	if query != "" && !looksCalendar(query) {
		if _, _, ok := ParseTimeRange(query); !ok {
			u += "&$search=" + url.QueryEscape(query)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("outlook events: %s", truncate(string(body), 300))
	}
	var doc struct {
		Value []struct {
			Subject string `json:"subject"`
			Start   struct {
				DateTime string `json:"dateTime"`
			} `json:"start"`
			Location struct {
				DisplayName string `json:"displayName"`
			} `json:"location"`
		} `json:"value"`
	}
	_ = json.Unmarshal(body, &doc)
	out := make([]map[string]string, 0, len(doc.Value))
	for _, it := range doc.Value {
		out = append(out, map[string]string{
			"provider": "outlook",
			"kind":     "calendar",
			"subject":  it.Subject,
			"start":    it.Start.DateTime,
			"location": it.Location.DisplayName,
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "outlook", "kind": "calendar", "note": "no events", "query": query})
	}
	return out, nil
}

// truncate 超过 n 个字节时截断并加上省略号。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// envSecret 读取 YOYO_GMAIL_CLIENT_SECRET 或 YOYO_OUTLOOK_CLIENT_SECRET。本文件没有调用它。
func envSecret(provider string) string {
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		return strings.TrimSpace(os.Getenv("YOYO_GMAIL_CLIENT_SECRET"))
	case "outlook":
		return strings.TrimSpace(os.Getenv("YOYO_OUTLOOK_CLIENT_SECRET"))
	default:
		return ""
	}
}
