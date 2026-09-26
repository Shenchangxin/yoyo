package connector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type TokenStore interface {
	Get(name string) (string, error)
	Set(name, value string)
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

func (b *Broker) SetTokens(store TokenStore) {
	b.mu.Lock()
	b.tokens = store
	b.mu.Unlock()
}

func (b *Broker) SetHTTP(client *http.Client) {
	b.mu.Lock()
	b.http = client
	b.mu.Unlock()
}

func (b *Broker) client() *http.Client {
	if b.http != nil {
		return b.http
	}
	return http.DefaultClient
}

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

func (b *Broker) SaveToken(vaultKey string, tok Token) error {
	if b.tokens == nil || vaultKey == "" {
		return fmt.Errorf("connector: no token store")
	}
	raw, _ := json.Marshal(tok)
	b.tokens.Set(vaultKey, string(raw))
	return nil
}

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

func (b *Broker) liveSend(a Account, d Draft) error {
	tok, err := b.loadToken(a.VaultKey)
	if err != nil {
		return err
	}
	provider := strings.ToLower(a.Provider)
	if a.Kind == KindCalendar || provider == "gcal" {
		return liveCreateEvent(b.client(), provider, tok.AccessToken, d)
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

func gmailList(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=8"
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
		if i >= 5 {
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

func gmailSend(client *http.Client, token string, d Draft) error {
	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s", d.To, d.Subject, d.Body)
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
		return fmt.Errorf("gmail send: %s", truncate(string(body), 300))
	}
	return nil
}

func outlookList(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://graph.microsoft.com/v1.0/me/messages?$top=8&$select=subject,from,bodyPreview"
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
		return fmt.Errorf("outlook send: %s", truncate(string(body), 300))
	}
	return nil
}

func liveCreateEvent(client *http.Client, provider, token string, d Draft) error {
	start, end := eventWindow(d.To)
	switch strings.ToLower(provider) {
	case "gmail", "gcal":
		return gcalInsert(client, token, d, start, end)
	case "outlook":
		return outlookCreateEvent(client, token, d, start, end)
	default:
		return fmt.Errorf("connector: calendar write for %s is MCP-pack only", provider)
	}
}

func eventWindow(to string) (time.Time, time.Time) {
	start := time.Now().UTC().Add(time.Hour).Truncate(time.Minute)
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(to)); err == nil {
		start = t
	}
	return start, start.Add(time.Hour)
}

func gcalInsert(client *http.Client, token string, d Draft, start, end time.Time) error {
	payload := map[string]any{
		"summary":     d.Subject,
		"description": d.Body,
		"start":       map[string]string{"dateTime": start.Format(time.RFC3339), "timeZone": "UTC"},
		"end":         map[string]string{"dateTime": end.Format(time.RFC3339), "timeZone": "UTC"},
	}
	if strings.Contains(d.To, "@") && !strings.Contains(d.To, "T") {
		payload["attendees"] = []map[string]string{{"email": strings.TrimSpace(d.To)}}
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "https://www.googleapis.com/calendar/v3/calendars/primary/events", strings.NewReader(string(raw)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gcal insert: %s", truncate(string(body), 300))
	}
	return nil
}

func outlookCreateEvent(client *http.Client, token string, d Draft, start, end time.Time) error {
	payload := map[string]any{
		"subject": d.Subject,
		"body":    map[string]string{"contentType": "Text", "content": d.Body},
		"start":   map[string]string{"dateTime": start.UTC().Format("2006-01-02T15:04:05"), "timeZone": "UTC"},
		"end":     map[string]string{"dateTime": end.UTC().Format("2006-01-02T15:04:05"), "timeZone": "UTC"},
	}
	if strings.Contains(d.To, "@") && !strings.Contains(d.To, "T") {
		payload["attendees"] = []map[string]any{
			{"emailAddress": map[string]string{"address": strings.TrimSpace(d.To)}, "type": "required"},
		}
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
		return fmt.Errorf("outlook event: %s", truncate(string(body), 300))
	}
	return nil
}

func looksCalendar(query string) bool {
	q := strings.ToLower(query)
	for _, n := range []string{"calendar", "event", "日程", "会议", "meetup", "agenda"} {
		if strings.Contains(q, n) {
			return true
		}
	}
	return false
}

func gcalList(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://www.googleapis.com/calendar/v3/calendars/primary/events?maxResults=8&singleEvents=true&orderBy=startTime&timeMin=" + url.QueryEscape(time.Now().UTC().Format(time.RFC3339))
	if query != "" && !looksCalendar(query) {
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

func outlookEvents(client *http.Client, token, query string) ([]map[string]string, error) {
	u := "https://graph.microsoft.com/v1.0/me/events?$top=8&$select=subject,start,end,location"
	if query != "" && !looksCalendar(query) {
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
		return nil, fmt.Errorf("outlook events: %s", truncate(string(body), 300))
	}
	var doc struct {
		Value []struct {
			Subject  string `json:"subject"`
			Start    struct{ DateTime string `json:"dateTime"` } `json:"start"`
			Location struct{ DisplayName string `json:"displayName"` } `json:"location"`
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

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
