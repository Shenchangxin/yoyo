package cut

import "testing"

func TestEncodeParseSRTRoundTrip(t *testing.T) {
	in := []Cue{{Start: 1.5, End: 3.2, Text: "先听这一句。"}, {Start: 3.2, End: 5, Text: "再切下一段。"}}
	raw := EncodeSRT(in)
	got := ParseSRT(raw)
	if len(got) != 2 {
		t.Fatalf("len %d raw %q", len(got), raw)
	}
	if got[0].Text != in[0].Text {
		t.Fatalf("text %q", got[0].Text)
	}
	if got[0].Start < 1.4 || got[0].Start > 1.6 {
		t.Fatalf("start %v", got[0].Start)
	}
}

func TestCuesFromTextCoversDuration(t *testing.T) {
	cues := CuesFromText("第一句。第二句。第三句。", 9)
	if len(cues) != 3 {
		t.Fatalf("len %d", len(cues))
	}
	if cues[len(cues)-1].End < 8 {
		t.Fatalf("end %v", cues[len(cues)-1].End)
	}
}

func TestEncodeVTTHeader(t *testing.T) {
	raw := EncodeVTT([]Cue{{Start: 0, End: 1, Text: "hi"}})
	if raw[:6] != "WEBVTT" {
		t.Fatalf("%q", raw)
	}
}
