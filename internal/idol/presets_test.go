// 本文件锁住媒体地址拼接、端点比较、音色随模型收口，以及已知端点推断和自定义端点保留。
// 由 go test 调用。

package idol

import "testing"

// TestMediaURLJoinsOpenAIRoots 锁住没有版本的根会加上 /v1，已有 /v1 或 /v4 时不再加第二个版本。
func TestMediaURLJoinsOpenAIRoots(t *testing.T) {
	cases := []struct{ endpoint, suffix, want string }{
		{"https://api.openai.com", "audio/speech", "https://api.openai.com/v1/audio/speech"},
		{"https://api.openai.com/v1", "audio/speech", "https://api.openai.com/v1/audio/speech"},
		{"https://api.openai.com/v1/audio/speech", "audio/speech", "https://api.openai.com/v1/audio/speech"},
		{"https://api.groq.com/openai/v1/", "audio/transcriptions", "https://api.groq.com/openai/v1/audio/transcriptions"},
		{"https://open.bigmodel.cn/api/paas/v4", "images/generations", "https://open.bigmodel.cn/api/paas/v4/images/generations"},
		{"https://dashscope.aliyuncs.com/compatible-mode/v1", "audio/speech", "https://dashscope.aliyuncs.com/compatible-mode/v1/audio/speech"},
		{"https://api.minimax.io", "audio/speech", "https://api.minimax.io/v1/audio/speech"},
	}
	for _, tc := range cases {
		if got := MediaURL(tc.endpoint, tc.suffix); got != tc.want {
			t.Fatalf("MediaURL(%q, %q) = %q, want %q", tc.endpoint, tc.suffix, got, tc.want)
		}
	}
}

// TestSameEndpointIgnoresOnlyV1 锁住末尾 /v1 可以忽略，/v4 必须留在路径里，空端点不能对上。
func TestSameEndpointIgnoresOnlyV1(t *testing.T) {
	if !SameEndpoint("https://api.openai.com", "https://api.openai.com/v1/") {
		t.Fatal("openai root should match its /v1 preset")
	}
	if !SameEndpoint("https://api.groq.com/openai", "https://api.groq.com/openai/v1") {
		t.Fatal("groq root should match its /v1 preset")
	}
	if SameEndpoint("https://open.bigmodel.cn/api/paas/v4", "https://open.bigmodel.cn/api/paas") {
		t.Fatal("/v4 must stay part of the path")
	}
	if SameEndpoint("", "https://api.openai.com/v1") {
		t.Fatal("empty endpoint must not match")
	}
}

// TestVoicesFollowTheModel 锁住 tts-1 不提供 verse 和 ash，gpt-4o-mini-tts 提供 verse，tts-1-hd 提供 alloy。
func TestVoicesFollowTheModel(t *testing.T) {
	p, ok := Preset("openai-speech")
	if !ok {
		t.Fatal("missing openai speech preset")
	}
	for _, v := range VoicesFor(p, "tts-1") {
		if v.ID == "verse" || v.ID == "ash" {
			t.Fatalf("tts-1 must not offer %s", v.ID)
		}
	}
	if !voiceKnown(p, "gpt-4o-mini-tts", "verse") {
		t.Fatal("gpt-4o-mini-tts should offer verse")
	}
	if !voiceKnown(p, "tts-1-hd", "alloy") {
		t.Fatal("tts-1-hd should offer alloy")
	}
}

// TestApplyClampsVoiceToTheModel 锁住 openai-speech 配 tts-1 时，verse 会被收成该模型列表里的第一项 alloy，端点写成目录地址。
func TestApplyClampsVoiceToTheModel(t *testing.T) {
	got := ApplyServicePresets(Services{
		SpeechProvider: "openai-speech",
		SpeechModel:    "tts-1",
		SpeechVoice:    "verse",
	})
	if got.SpeechBaseURL != "https://api.openai.com/v1" {
		t.Fatalf("endpoint %q", got.SpeechBaseURL)
	}
	if got.SpeechModel != "tts-1" || got.SpeechVoice != "alloy" {
		t.Fatalf("model %q voice %q", got.SpeechModel, got.SpeechVoice)
	}
}

// TestApplyKeepsACustomEndpoint 锁住未知供应商 id 会被清掉，本地端点、模型和音色保持原样。
func TestApplyKeepsACustomEndpoint(t *testing.T) {
	got := ApplyServicePresets(Services{
		SpeechProvider: "nope",
		SpeechBaseURL:  "http://127.0.0.1:8080/v1",
		SpeechModel:    "local",
		SpeechVoice:    "warm",
	})
	if got.SpeechProvider != "" || got.SpeechBaseURL != "http://127.0.0.1:8080/v1" || got.SpeechModel != "local" || got.SpeechVoice != "warm" {
		t.Fatalf("%+v", got)
	}
}

// TestInferKnownPresetAndKeepPrivateModel 锁住已知端点加目录里的模型会填上供应商 id，私有模型和克隆音色保持自定义，只给 MiniMax 根地址时补上目录里的第一项模型和音色。
func TestInferKnownPresetAndKeepPrivateModel(t *testing.T) {
	known := InferProviders(Services{SpeechBaseURL: "https://api.openai.com", SpeechModel: "tts-1", SpeechVoice: "nova"})
	if known.SpeechProvider != "openai-speech" {
		t.Fatalf("provider %q", known.SpeechProvider)
	}
	full := NormalizeServices(known)
	if full.SpeechBaseURL != "https://api.openai.com/v1" || full.SpeechVoice != "nova" {
		t.Fatalf("%+v", full)
	}

	private := InferProviders(Services{SpeechBaseURL: "https://api.openai.com/v1", SpeechModel: "ft:private"})
	if private.SpeechProvider != "" {
		t.Fatalf("private model should stay custom, got %q", private.SpeechProvider)
	}
	cloned := InferProviders(Services{SpeechBaseURL: "https://api.openai.com", SpeechModel: "gpt-4o-mini-tts", SpeechVoice: "my-clone"})
	if cloned.SpeechProvider != "" {
		t.Fatalf("cloned voice should stay custom, got %q", cloned.SpeechProvider)
	}

	mini := NormalizeServices(Services{SpeechBaseURL: "https://api.minimax.io"})
	if mini.SpeechProvider != "minimax-speech" || mini.SpeechModel != "speech-2.6-hd" || mini.SpeechVoice != "male-qn-qingse" {
		t.Fatalf("%+v", mini)
	}
	if mini.SpeechBaseURL != "https://api.minimax.io/v1" {
		t.Fatalf("endpoint %q", mini.SpeechBaseURL)
	}
}

// TestMatchPresetPrefersEndpoint 锁住端点对上时优先于供应商名，端点为空时回落到供应商，两边都空则没有匹配。
func TestMatchPresetPrefersEndpoint(t *testing.T) {
	if got := MatchPreset("asr", "openai", "https://api.groq.com/openai/v1"); got != "groq-asr" {
		t.Fatalf("endpoint should win, got %q", got)
	}
	if got := MatchPreset("image", "zhipu", ""); got != "zhipu-image" {
		t.Fatalf("vendor fallback got %q", got)
	}
	if got := MatchPreset("speech", "", ""); got != "" {
		t.Fatalf("empty match got %q", got)
	}
}
