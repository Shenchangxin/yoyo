package webcontent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2EXNodeUsesTheAINode(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.RequestURI()
		_, _ = w.Write([]byte(`[{"title":"智能体","url":"https://www.v2ex.com/t/2","content":"Agent"}]`))
	}))
	t.Cleanup(srv.Close)
	prev := V2EXAPIBase
	V2EXAPIBase = srv.URL + "/api"
	t.Cleanup(func() { V2EXAPIBase = prev })
	hits, err := V2EXNode(context.Background(), srv.Client(), "ai", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "node_name=ai") || len(hits) != 1 || hits[0].URL != "https://www.v2ex.com/t/2" {
		t.Fatalf("path %s hits %+v", path, hits)
	}
	again, err := V2EXNode(context.Background(), srv.Client(), "../hot", 5)
	if err != nil || !strings.Contains(path, "node_name=ai") {
		t.Fatalf("node %s %v", path, err)
	}
	if again[0].Title != "智能体" {
		t.Fatalf("%+v", again)
	}
}
