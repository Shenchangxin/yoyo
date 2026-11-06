// gen_route_test.go 锁住生图按连接协议分发，以及 OpenAI 兼容地址不重复补版本段。

package video

import "testing"

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
