package idol

import "testing"

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
