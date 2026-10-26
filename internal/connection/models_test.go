// 锁住模型名到能力的推断，以及单一能力连接不会把自定义模型名漏成聊天。

package connection

import "testing"

// TestCapOfModel 确认一批模型 id 被分到预期的能力。
func TestCapOfModel(t *testing.T) {
	cases := map[string]string{
		"gpt-4.1":                 CapChat,
		"qwen-plus":               CapChat,
		"gpt-image-1":             CapImage,
		"dall-e-3":                CapImage,
		"doubao-seedream-5-0":     CapImage,
		"sora-2":                  CapVideo,
		"doubao-seedance-2-0":     CapVideo,
		"wan3.0-video":            CapVideo,
		"gpt-4o-mini-tts":         CapSpeech,
		"speech-2.6-hd":           CapSpeech,
		"music-2.0":               CapMusic,
		"music-3.0":               CapMusic,
		"saliency_seg":            CapMatting,
		"removebg":                CapMatting,
		"stable-audio-2.5":        CapMusic,
		"lyria-002":               CapMusic,
		"eleven_text_to_sound_v2": CapMusic,
		"whisper-1":               CapASR,
		"gpt-4o-mini-transcribe":  CapASR,
		"paraformer-v2":           CapASR,
		"nova-3":                  CapASR,
	}
	for model, want := range cases {
		if got := CapOfModel(model); got != want {
			t.Fatalf("%s: got %s want %s", model, got, want)
		}
	}
}

// TestModelsForSplitsMultiCap 确认同时声明多种能力时，模型按能力拆开，能力列表含这些声明。
func TestModelsForSplitsMultiCap(t *testing.T) {
	c := Connection{
		Capabilities: []string{CapChat},
		Models:       []string{"gpt-4.1", "gpt-image-1", "sora-2"},
		DefaultModel: map[string]string{CapChat: "gpt-4.1"},
	}
	if !containsAll(ModelsFor(c, CapChat), "gpt-4.1") || contains(ModelsFor(c, CapChat), "gpt-image-1") {
		t.Fatalf("chat %v", ModelsFor(c, CapChat))
	}
	if !containsAll(ModelsFor(c, CapImage), "gpt-image-1") {
		t.Fatalf("image %v", ModelsFor(c, CapImage))
	}
	if !containsAll(ModelsFor(c, CapVideo), "sora-2") {
		t.Fatalf("video %v", ModelsFor(c, CapVideo))
	}
	caps := CapsOf(c)
	if !containsAll(caps, CapChat, CapImage, CapVideo) {
		t.Fatalf("caps %v", caps)
	}
}

// TestSingleCapKeepsCustomNames 确认只声明图像时，自定义模型名留在图像侧，不漏进聊天。
func TestSingleCapKeepsCustomNames(t *testing.T) {
	c := Connection{
		Capabilities: []string{CapImage},
		Models:       []string{"studio-custom"},
		DefaultModel: map[string]string{CapImage: "studio-custom"},
	}
	if !containsAll(ModelsFor(c, CapImage), "studio-custom") {
		t.Fatalf("image %v", ModelsFor(c, CapImage))
	}
	if len(ModelsFor(c, CapChat)) != 0 {
		t.Fatalf("chat leaked %v", ModelsFor(c, CapChat))
	}
}

// TestInferProtocolMusicStockAsr 确认音乐、素材和语音识别供应商推断出预期协议。
func TestInferProtocolMusicStockAsr(t *testing.T) {
	cases := []struct {
		vendor, cap, want string
	}{
		{"minimax", CapMusic, "minimax-music"},
		{"elevenlabs", CapMusic, "elevenlabs-sound"},
		{"stability", CapMusic, "stability-audio"},
		{"suno", CapMusic, "suno"},
		{"gemini", CapMusic, "google-lyria"},
		{"openai", CapMusic, "openai-audio"},
		{"unsplash", CapStock, "unsplash"},
		{"giphy", CapStock, "giphy"},
		{"pixabay", CapStock, "pixabay"},
		{"pexels", CapStock, "pexels"},
		{"deepgram", CapASR, "deepgram"},
		{"assemblyai", CapASR, "assemblyai"},
		{"azure", CapASR, "azure-stt"},
		{"aliyun", CapASR, "dashscope-asr"},
		{"groq", CapASR, "openai-audio"},
		{"jimeng", CapImage, "volcengine-jimeng-image"},
		{"jimeng", CapVideo, "volcengine-jimeng-video"},
		{"vidu", CapVideo, "vidu"},
		{"tencent", CapImage, "hunyuan-image"},
		{"tencent", CapVideo, "hunyuan-video"},
		{"tencent", CapASR, "tencent-asr"},
		{"mureka", CapMusic, "mureka"},
		{"volcengine", CapMatting, "volcengine-visual-matting"},
		{"removebg", CapMatting, "removebg"},
		{"photoroom", CapMatting, "photoroom"},
	}
	for _, c := range cases {
		if got := InferProtocol(c.vendor, c.cap); got != c.want {
			t.Fatalf("%s %s: got %s want %s", c.vendor, c.cap, got, c.want)
		}
	}
}

// contains 报告 have 里是否有 want。
func contains(have []string, want string) bool {
	for _, h := range have {
		if h == want {
			return true
		}
	}
	return false
}

// containsAll 要求每个 want 都在 have 里。
func containsAll(have []string, want ...string) bool {
	for _, w := range want {
		if !contains(have, w) {
			return false
		}
	}
	return true
}
