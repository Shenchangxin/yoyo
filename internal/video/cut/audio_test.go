// audio_test.go 检查混音滤镜里的闪避、淡出和人声增强。

package cut

import "testing"

// TestMixGraphDucksMusicUnderSpeech 确认 Duck 为真时滤镜含 sidechaincompress，并带有映射参数。
func TestMixGraphDucksMusicUnderSpeech(t *testing.T) {
	filter, maps := MixGraph(true, true, MixOpts{Duck: true, MusicVol: 0.2, SpeechVol: 1})
	if filter == "" {
		t.Fatal("empty filter")
	}
	if !contains(filter, "sidechaincompress") {
		t.Fatalf("want duck, got %s", filter)
	}
	if len(maps) < 4 {
		t.Fatalf("maps %v", maps)
	}
}

// TestMixGraphFadeDoesNotDropDuck 确认淡出和闪避同时出现在滤镜里。
func TestMixGraphFadeDoesNotDropDuck(t *testing.T) {
	filter, _ := MixGraph(true, true, MixOpts{Duck: true, MusicVol: 0.22, SpeechVol: 1, FadeOut: 3, Timeline: 10})
	if !contains(filter, "afade=") || !contains(filter, "sidechaincompress") {
		t.Fatalf("got %s", filter)
	}
}

// TestMixGraphSpeechReplace 确认只有口播且 ReplaceOrig 为真时滤镜为空，音频映射到 1:a:0。
func TestMixGraphSpeechReplace(t *testing.T) {
	filter, maps := MixGraph(true, false, MixOpts{ReplaceOrig: true})
	if filter != "" {
		t.Fatalf("unexpected filter %s", filter)
	}
	if maps[3] != "1:a:0" {
		t.Fatalf("maps %v", maps)
	}
}

// TestEnhanceFilterVoice 确认 voice 预设同时含 loudnorm 和 afftdn。
func TestEnhanceFilterVoice(t *testing.T) {
	f := EnhanceFilter("voice")
	if !contains(f, "loudnorm") || !contains(f, "afftdn") {
		t.Fatalf("got %s", f)
	}
}

// contains 报告 s 是否包含 sub。
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
