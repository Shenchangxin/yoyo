// captions_test.go 检查 SRT 往返、按字数摊时长，以及 VTT 文件头。

package cut

import (
	"math"
	"testing"
)

// TestMapCuesFollowsTheCut 确认源片字幕按剪辑表搬到成片：剪掉的部分丢掉，跨剪辑点的切开，略过的不要。
func TestMapCuesFollowsTheCut(t *testing.T) {
	cues := []Cue{
		{Start: 1, End: 3, Text: "留下"},
		{Start: 4, End: 6, Text: "剪掉"},
		{Start: 9, End: 11, Text: "跨过"},
		{Start: 12, End: 13, Text: "略过", Omit: true},
	}
	cuts := []Clip{{Start: 0, End: 3.5}, {Start: 10, End: 14}}
	got := MapCues(cues, cuts)
	if len(got) != 2 {
		t.Fatalf("cues %+v", got)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if got[0].Text != "留下" || !near(got[0].Start, 1) || !near(got[0].End, 3) {
		t.Fatalf("first %+v", got[0])
	}
	if got[1].Text != "跨过" || !near(got[1].Start, 3.5) || !near(got[1].End, 4.5) {
		t.Fatalf("second %+v", got[1])
	}
}

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
