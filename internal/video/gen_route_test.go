// gen_route_test.go 锁住生图按连接协议分发，以及 OpenAI 兼容地址不重复补版本段。

package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestImageRouteFollowsProtocol 确认 openai-images 协议不论厂商都走 OpenAI 兼容实现，即梦和混元走各自的实现，认不出的协议返回空串。
func TestImageRouteFollowsProtocol(t *testing.T) {
	cases := []struct {
		p    Provider
		want string
	}{
		{Provider{Provider: "aliyun", Protocol: "openai-images"}, "openai"},
		{Provider{Provider: "kolors", Protocol: "openai-images"}, "openai"},
		{Provider{Provider: "zhipu", Protocol: "openai-images"}, "openai"},
		{Provider{Provider: "gemini", Protocol: "google-gemini-image"}, "gemini"},
		{Provider{Provider: "volcengine", Protocol: "volcengine-ark-seedream"}, "seedream"},
		{Provider{Provider: "openai"}, "openai"},
		{Provider{Provider: "jimeng", Protocol: "volcengine-jimeng-image"}, "jimeng"},
		{Provider{Provider: "tencent", Protocol: "hunyuan-image"}, "hunyuan"},
		{Provider{Provider: "runninghub", Protocol: "runninghub"}, ""},
	}
	for _, c := range cases {
		if got := imageRoute(c.p); got != c.want {
			t.Fatalf("%s/%s: got %q want %q", c.p.Provider, c.p.Protocol, got, c.want)
		}
	}
}

// TestOpenAICompatURLKeepsVersionedBase 确认已带版本段的基础地址不再补 /v1，裸域名补上 /v1。
func TestOpenAICompatURLKeepsVersionedBase(t *testing.T) {
	cases := map[string]string{
		"https://api.openai.com":                            "https://api.openai.com/v1/images/generations",
		"https://api.openai.com/v1/":                        "https://api.openai.com/v1/images/generations",
		"https://open.bigmodel.cn/api/paas/v4":              "https://open.bigmodel.cn/api/paas/v4/images/generations",
		"https://dashscope.aliyuncs.com/compatible-mode/v1": "https://dashscope.aliyuncs.com/compatible-mode/v1/images/generations",
		"https://api.klingai.com":                           "https://api.klingai.com/v1/images/generations",
		"":                                                  "https://api.openai.com/v1/images/generations",
	}
	for base, want := range cases {
		if got := openAICompatURL(base, "/images/generations"); got != want {
			t.Fatalf("%q: got %q want %q", base, got, want)
		}
	}
}

// TestOpenAIVideoUsesProtocolAdapter 确认手写分支没有的 OpenAI 视频走协议适配器，认不出的厂商仍直接拒绝。
func TestOpenAIVideoUsesProtocolAdapter(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"task-1","status":"queued"}`))
	}))
	defer srv.Close()

	e := testEngine(t)
	e.HTTP = srv.Client()
	if e.Proto == nil {
		t.Fatal("protocol registry missing")
	}
	p, err := e.UpsertProvider(Provider{
		ServiceType: "video", Provider: "openai", Name: "Sora", BaseURL: srv.URL, Model: "sora-2", IsActive: true,
	}, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	j := &Job{Type: "video", Provider: p.Provider, VaultKey: p.VaultKey, Model: "sora-2", Prompt: "a cat"}
	if err := e.submitVideo(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/videos" {
		t.Fatalf("path %s", gotPath)
	}
	if j.RemoteID != "task-1" {
		t.Fatalf("remote %s", j.RemoteID)
	}
	if parseParams(j.Params).ProtocolRoute == "" {
		t.Fatal("protocol route was not recorded")
	}

	e.HTTP = rejectHTTP{t}
	mystery, err := e.UpsertProvider(Provider{
		ServiceType: "video", Provider: "mystery", Name: "Mystery", BaseURL: srv.URL, IsActive: true,
	}, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	err = e.submitVideo(context.Background(), &Job{Type: "video", Provider: mystery.Provider, VaultKey: mystery.VaultKey, Prompt: "nope"})
	if err == nil || !strings.Contains(err.Error(), "unsupported video provider") {
		t.Fatalf("mystery: %v", err)
	}
}

type rejectHTTP struct{ t *testing.T }

func (r rejectHTTP) Do(*http.Request) (*http.Response, error) {
	r.t.Fatal("handwritten miss must not call the network")
	return nil, nil
}
