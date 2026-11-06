package connector

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExtractNotionAndGitHubShapes(t *testing.T) {
	notion := []byte(`{"results":[{"title":[{"plain_text":"周报"}],"url":"https://www.notion.so/abc"}]}`)
	items := extractItems("notion", notion)
	if len(items) != 1 || items[0]["title"] != "周报" || items[0]["url"] != "https://www.notion.so/abc" {
		t.Fatalf("%#v", items)
	}
	repos := []byte(`[{"full_name":"yo/yoyo","html_url":"https://github.com/yo/yoyo","description":"desk"}]`)
	items = extractItems("github", repos)
	if len(items) != 1 || items[0]["title"] != "yo/yoyo" {
		t.Fatalf("%#v", items)
	}
}

func TestParseMailHeaders(t *testing.T) {
	raw := "Subject: 账单\r\nFrom: a@b.com\r\na3 OK Fetch completed"
	items := parseMailHeaders("qq-mail", raw)
	if len(items) != 1 || items[0]["title"] != "账单" || items[0]["text"] != "a@b.com" {
		t.Fatalf("%#v", items)
	}
}

func TestUseNotionReadsTheTokenAPI(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	toks := memTokens{}
	b.SetTokens(toks)
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.notion.com" {
			t.Fatalf("host %s", r.URL.Host)
		}
		if r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("Notion-Version") == "" {
			t.Fatalf("headers %#v", r.Header)
		}
		body := `{"results":[{"title":[{"plain_text":"笔记"}],"url":"https://www.notion.so/n"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	got, err := b.Use("notion", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Account.Provider != "notion" || len(got.Items) != 1 || got.Items[0]["title"] != "笔记" {
		t.Fatalf("%#v", got)
	}
}

func TestUsePhoneNeedsASync(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Use("ios-health", "", ""); err == nil || !strings.Contains(err.Error(), "同步") {
		t.Fatal(err)
	}
	b.PutPhone("ios-health", []map[string]string{{"title": "睡眠 7 小时"}})
	got, err := b.Use("ios-health", "", "")
	if err != nil || len(got.Items) != 1 || got.Items[0]["title"] != "睡眠 7 小时" {
		t.Fatalf("%v %#v", err, got)
	}
}

func TestCatalogStillListsTheDirectory(t *testing.T) {
	if len(Catalog()) < 40 {
		t.Fatalf("catalog %d", len(Catalog()))
	}
}
