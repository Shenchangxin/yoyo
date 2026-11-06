package connector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type tokenCall struct {
	Method string
	URL    string
	Body   string
	Auth   string
}

func tokenCalls() map[string]tokenCall {
	return map[string]tokenCall{
		"gitlab":          {Method: http.MethodGet, URL: "https://gitlab.com/api/v4/projects?membership=true&simple=true&per_page=15", Auth: "private"},
		"notion":          {Method: http.MethodPost, URL: "https://api.notion.com/v1/search", Body: `{"page_size":10}`, Auth: "bearer"},
		"dropbox":         {Method: http.MethodPost, URL: "https://api.dropboxapi.com/2/files/list_folder", Body: `{"path":""}`, Auth: "bearer"},
		"figma":           {Method: http.MethodGet, URL: "https://api.figma.com/v1/me", Auth: "figma"},
		"miro":            {Method: http.MethodGet, URL: "https://api.miro.com/v2/boards", Auth: "bearer"},
		"airtable":        {Method: http.MethodGet, URL: "https://api.airtable.com/v0/meta/bases", Auth: "bearer"},
		"supabase":        {Method: http.MethodGet, URL: "https://api.supabase.com/v1/projects", Auth: "bearer"},
		"zoom":            {Method: http.MethodGet, URL: "https://api.zoom.us/v2/users/me", Auth: "bearer"},
		"calendly":        {Method: http.MethodGet, URL: "https://api.calendly.com/users/me", Auth: "bearer"},
		"calcom":          {Method: http.MethodGet, URL: "https://api.cal.com/v2/me", Auth: "bearer"},
		"linear":          {Method: http.MethodPost, URL: "https://api.linear.app/graphql", Body: `{"query":"{ issues(first:10) { nodes { title url } } }"}`, Auth: "bearer"},
		"todoist":         {Method: http.MethodGet, URL: "https://api.todoist.com/rest/v2/tasks", Auth: "bearer"},
		"ticktick":        {Method: http.MethodGet, URL: "https://api.ticktick.com/open/v1/project", Auth: "bearer"},
		"trello":          {Method: http.MethodGet, URL: "https://api.trello.com/1/members/me/boards?fields=name,url,desc&key={key}&token={token}", Auth: "trello"},
		"asana":           {Method: http.MethodGet, URL: "https://app.asana.com/api/1.0/projects?limit=15", Auth: "bearer"},
		"clickup":         {Method: http.MethodGet, URL: "https://api.clickup.com/api/v2/team", Auth: "raw"},
		"sentry":          {Method: http.MethodGet, URL: "https://sentry.io/api/0/projects/", Auth: "bearer"},
		"hubspot":         {Method: http.MethodGet, URL: "https://api.hubapi.com/crm/v3/objects/contacts?limit=10", Auth: "bearer"},
		"attio":           {Method: http.MethodGet, URL: "https://api.attio.com/v2/self", Auth: "bearer"},
		"ynab":            {Method: http.MethodGet, URL: "https://api.ynab.com/v1/budgets", Auth: "bearer"},
		"github":          {Method: http.MethodGet, URL: "https://api.github.com/user/repos?per_page=15&sort=updated", Auth: "bearer"},
		"wps":             {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"tencent-docs":    {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"weread":          {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"tencent-meeting": {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"linkedin":        {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"quickbooks":      {Method: http.MethodGet, URL: "", Auth: "bearer"},
		"zoho-invoice":    {Method: http.MethodGet, URL: "", Auth: "bearer"},
	}
}

func callTokenAPI(client *http.Client, provider, token, endpoint string) ([]map[string]string, bool, error) {
	spec, ok := tokenCalls()[provider]
	if !ok {
		return nil, false, nil
	}
	url := spec.URL
	if strings.HasPrefix(endpoint, "https://") {
		url = endpoint
	}
	if url == "" {
		return nil, true, fmt.Errorf("connector: %s 需要填写它自己的 https 接口地址", provider)
	}
	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}
	body := spec.Body
	if provider == "notion" {
		body = `{"page_size":10}`
	}
	var reader io.Reader
	if body != "" && method != http.MethodGet {
		reader = strings.NewReader(body)
	}
	if spec.Auth == "trello" {
		key, tok, _ := strings.Cut(token, ":")
		url = strings.ReplaceAll(url, "{key}", key)
		url = strings.ReplaceAll(url, "{token}", tok)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, true, err
	}
	switch spec.Auth {
	case "private":
		req.Header.Set("PRIVATE-TOKEN", token)
	case "figma":
		req.Header.Set("X-Figma-Token", token)
	case "raw":
		req.Header.Set("Authorization", token)
	case "trello":
	default:
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if provider == "notion" {
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")
	}
	if provider == "github" {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, true, fmt.Errorf("connector: %s %s", provider, truncate(string(raw), 180))
	}
	items := extractItems(provider, raw)
	if len(items) == 0 {
		return nil, true, fmt.Errorf("connector: %s 没有返回可读的条目", provider)
	}
	return items, true, nil
}

func extractItems(provider string, raw []byte) []map[string]string {
	var v any
	if json.Unmarshal(bytes.TrimSpace(raw), &v) != nil {
		return nil
	}
	arr := findArray(v, 0)
	if len(arr) == 0 {
		if one := itemFrom(provider, v); one != nil {
			return []map[string]string{one}
		}
		return nil
	}
	var out []map[string]string
	for _, el := range arr {
		if one := itemFrom(provider, el); one != nil {
			out = append(out, one)
		}
		if len(out) == 15 {
			break
		}
	}
	return out
}

func findArray(v any, depth int) []any {
	if depth > 4 {
		return nil
	}
	switch t := v.(type) {
	case []any:
		for _, el := range t {
			if _, ok := el.(map[string]any); ok {
				return t
			}
		}
	case map[string]any:
		for _, key := range []string{"items", "results", "values", "entries", "channels", "nodes", "data", "projects", "boards", "issues", "repositories", "bases", "budgets", "teams", "contacts"} {
			if arr, ok := t[key].([]any); ok {
				return arr
			}
		}
		for _, child := range t {
			if arr := findArray(child, depth+1); len(arr) > 0 {
				return arr
			}
		}
	}
	return nil
}

func itemFrom(provider string, v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	title := firstString(m, "title", "name", "subject", "full_name", "summary", "handle", "email")
	if title == "" {
		title = firstString(m, "id")
	}
	if title == "" {
		return nil
	}
	return map[string]string{
		"provider": provider,
		"title":    title,
		"url":      firstString(m, "html_url", "url", "web_url", "permalink", "link"),
		"text":     firstString(m, "description", "desc", "snippet", "body", "content", "plain_text"),
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := asString(m[key]); s != "" {
			return s
		}
	}
	return ""
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		var b strings.Builder
		for _, one := range t {
			if s := asString(one); s != "" {
				b.WriteString(s)
			}
		}
		return strings.TrimSpace(b.String())
	case map[string]any:
		if s := asString(t["plain_text"]); s != "" {
			return s
		}
		return asString(t["name"])
	default:
		return ""
	}
}

func slackList(client *http.Client, token string) ([]map[string]string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://slack.com/api/conversations.list?limit=15&types=public_channel", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var body struct {
		OK       bool   `json:"ok"`
		Error    string `json:"error"`
		Channels []struct {
			Name    string `json:"name"`
			Purpose struct {
				Value string `json:"value"`
			} `json:"purpose"`
		} `json:"channels"`
	}
	if json.Unmarshal(raw, &body) != nil || !body.OK {
		msg := body.Error
		if msg == "" {
			msg = truncate(string(raw), 160)
		}
		return nil, fmt.Errorf("connector: slack %s", msg)
	}
	var items []map[string]string
	for _, ch := range body.Channels {
		if ch.Name == "" {
			continue
		}
		items = append(items, map[string]string{"provider": "slack", "title": "#" + ch.Name, "text": ch.Purpose.Value})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("connector: slack 没有可读的频道")
	}
	return items, nil
}

func slackSend(client *http.Client, token string, d Draft) error {
	raw, _ := json.Marshal(map[string]string{"channel": d.To, "text": d.Body})
	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &out)
	if !out.OK {
		if out.Error == "" {
			out.Error = truncate(string(body), 160)
		}
		return fmt.Errorf("connector: slack %s", out.Error)
	}
	return nil
}
