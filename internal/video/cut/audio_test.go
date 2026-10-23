package cut

import "testing"

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

func TestMixGraphFadeDoesNotDropDuck(t *testing.T) {
	filter, _ := MixGraph(true, true, MixOpts{Duck: true, MusicVol: 0.22, SpeechVol: 1, FadeOut: 3, Timeline: 10})
	if !contains(filter, "afade=") || !contains(filter, "sidechaincompress") {
		t.Fatalf("got %s", filter)
	}
}

func TestMixGraphSpeechReplace(t *testing.T) {
	filter, maps := MixGraph(true, false, MixOpts{ReplaceOrig: true})
	if filter != "" {
		t.Fatalf("unexpected filter %s", filter)
	}
	if maps[3] != "1:a:0" {
		t.Fatalf("maps %v", maps)
	}
}

func TestEnhanceFilterVoice(t *testing.T) {
	f := EnhanceFilter("voice")
	if !contains(f, "loudnorm") || !contains(f, "afftdn") {
		t.Fatalf("got %s", f)
	}
}

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
