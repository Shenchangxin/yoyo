package connection

import "testing"

func TestCapOfModel(t *testing.T) {
	cases := map[string]string{
		"gpt-4.1":              CapChat,
		"qwen-plus":            CapChat,
		"gpt-image-1":          CapImage,
		"dall-e-3":             CapImage,
		"doubao-seedream-5-0":  CapImage,
		"sora-2":               CapVideo,
		"doubao-seedance-2-0":  CapVideo,
		"wan3.0-video":         CapVideo,
		"gpt-4o-mini-tts":      CapSpeech,
		"speech-2.6-hd":        CapSpeech,
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
