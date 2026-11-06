package webcontent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBilibiliPopular(t *testing.T) {
	var referer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer = r.Header.Get("Referer")
		if !strings.Contains(r.URL.RawQuery, "ps=2") {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[{"title":"热门","bvid":"BV1test00001","desc":"说明"},{"title":"","bvid":"BV1skip"}]}}`))
	}))
	t.Cleanup(srv.Close)
	prev := BilibiliPopularBase
	BilibiliPopularBase = srv.URL
	t.Cleanup(func() { BilibiliPopularBase = prev })
	hits, err := BilibiliPopular(context.Background(), srv.Client(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if referer != "https://www.bilibili.com/" || len(hits) != 1 || hits[0].URL != "https://www.bilibili.com/video/BV1test00001" {
		t.Fatalf("referer %s hits %+v", referer, hits)
	}
}

func TestV2EXHot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/topics/hot.json") {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"title":"今日","url":"https://www.v2ex.com/t/1","content":"<p>你好</p>"},{"title":"没链接"}]`))
	}))
	t.Cleanup(srv.Close)
	prev := V2EXAPIBase
	V2EXAPIBase = srv.URL + "/api"
	t.Cleanup(func() { V2EXAPIBase = prev })
	hits, err := V2EXHot(context.Background(), srv.Client(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Title != "今日" || hits[0].Snippet != "你好" {
		t.Fatalf("%+v", hits)
	}
}
