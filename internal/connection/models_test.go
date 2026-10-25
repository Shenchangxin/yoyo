package connection

import "testing"

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

func contains(have []string, want string) bool {
	for _, h := range have {
		if h == want {
			return true
		}
	}
	return false
}

func containsAll(have []string, want ...string) bool {
	for _, w := range want {
		if !contains(have, w) {
			return false
		}
	}
	return true
}
