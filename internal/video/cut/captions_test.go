// captions_test.go 检查 SRT 往返、按字数摊时长，以及 VTT 文件头。

package cut

import "testing"

// TestEncodeParseSRTRoundTrip 确认编码再解析后保留文本，起点落在 1.4 到 1.6 秒。
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

// TestCuesFromTextCoversDuration 确认三句话在 9 秒里分成三条，末条结束不早于 8 秒。
func TestCuesFromTextCoversDuration(t *testing.T) {
	cues := CuesFromText("第一句。第二句。第三句。", 9)
	if len(cues) != 3 {
		t.Fatalf("len %d", len(cues))
	}
	if cues[len(cues)-1].End < 8 {
		t.Fatalf("end %v", cues[len(cues)-1].End)
	}
}

// TestEncodeVTTHeader 确认输出以前六个字符 WEBVTT 开头。
func TestEncodeVTTHeader(t *testing.T) {
	raw := EncodeVTT([]Cue{{Start: 0, End: 1, Text: "hi"}})
	if raw[:6] != "WEBVTT" {
		t.Fatalf("%q", raw)
	}
}
