package video

import "testing"

func TestNormalizeAspect(t *testing.T) {
	if NormalizeAspect("9:16") != "9:16" || NormalizeAspect("1:1") != "1:1" || NormalizeAspect("adaptive") != "adaptive" {
		t.Fatal("keep known ratios")
	}
	if NormalizeAspect("") != "16:9" || NormalizeAspect("21:9") != "16:9" {
		t.Fatal("unknown falls back to 16:9")
	}
}

func TestImageSizeFor(t *testing.T) {
	if ImageSizeFor("character", "9:16") != "1920x1080" {
		t.Fatal("character sheet stays landscape")
	}
	if ImageSizeFor("scene", "9:16") != "1080x1920" {
		t.Fatal("scene follows episode")
	}
	if ImageSizeFor("prop", "16:9") != "1024x1024" {
		t.Fatal("prop is square")
	}
}

func TestOpenAIImageSize(t *testing.T) {
	if OpenAIImageSize("1920x1080") != "1536x1024" {
		t.Fatal("landscape")
	}
	if OpenAIImageSize("1080x1920") != "1024x1536" {
		t.Fatal("portrait")
	}
	if OpenAIImageSize("1024x1024") != "1024x1024" {
		t.Fatal("square")
	}
}

func TestGeminiAspectFromSize(t *testing.T) {
	if GeminiAspectFromSize("1080x1920", "16:9") != "9:16" {
		t.Fatal("from size")
	}
	if GeminiAspectFromSize("", "adaptive") != "16:9" {
		t.Fatal("adaptive")
	}
}

func TestIsNarrator(t *testing.T) {
	if !IsNarrator("旁白", "") || !IsNarrator("VO", "narrator") {
		t.Fatal("should skip")
	}
	if IsNarrator("林小雨", "主角") {
		t.Fatal("on-screen")
	}
}

func TestClampProviderDuration(t *testing.T) {
	if ClampProviderDuration(8, "aliyun") != 8 {
		t.Fatal("wan keeps 8s")
	}
	if ClampProviderDuration(25, "aliyun") != 25 {
		t.Fatal("wan 25")
	}
	if ClampProviderDuration(20, "volcengine") != 15 {
		t.Fatal("seedance max 15")
	}
	if ClampSegment(8, "narrative") != 10 {
		t.Fatal("planning clamp still 10–15")
	}
}
