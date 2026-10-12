package video

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseTrendJSONShapes(t *testing.T) {
	cases := []struct {
		name, raw, title, hot string
	}{
		{"array data", `{"data":[{"title":"春日穿搭","hot":"120万"}]}`, "春日穿搭", "120万"},
		{"nested data object", `{"data":{"list":[{"name":"职场沟通","hotval":88}]}}`, "职场沟通", "88"},
		{"items", `{"items":[{"word":"成分党","index":12}]}`, "成分党", "12"},
	}
	for _, tc := range cases {
		got := parseTrendJSON([]byte(tc.raw))
		if len(got) != 1 || got[0].Title != tc.title || got[0].Hot != tc.hot {
			t.Fatalf("%s: %+v", tc.name, got)
		}
	}
	if parseTrendJSON([]byte(`{"no":"list"}`)) != nil && len(parseTrendJSON([]byte(`{"no":"list"}`))) != 0 {
		t.Fatal("empty")
	}
}

type stubHTTP struct {
	n    int
	body string
}

func (s *stubHTTP) Do(req *http.Request) (*http.Response, error) {
	s.n++
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestFetchTrendsCaches(t *testing.T) {
	e := testEngine(t)
	stub := &stubHTTP{body: `{"data":[{"title":"热点A","hot":"1"}]}`}
	e.HTTP = stub
	first, err := e.FetchTrends([]string{"weibo"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	groups, _ := first["trends"].([]TrendGroup)
	if len(groups) != 1 || !groups[0].OK || len(groups[0].Items) != 1 {
		t.Fatalf("first %+v", first)
	}
	if _, err := e.FetchTrends([]string{"weibo"}, 5); err != nil {
		t.Fatal(err)
	}
	if stub.n != 1 {
		t.Fatalf("cache missed, calls=%d", stub.n)
	}
}
