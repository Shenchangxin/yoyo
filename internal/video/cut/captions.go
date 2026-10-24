package cut

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var reSRTTime = regexp.MustCompile(`(\d+):(\d+):(\d+)[,.](\d+)\s*-->\s*(\d+):(\d+):(\d+)[,.](\d+)`)

func EncodeSRT(cues []Cue) string {
	var b strings.Builder
	for i, c := range cues {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, srtStamp(c.Start), srtStamp(c.End), strings.TrimSpace(c.Text))
	}
	return b.String()
}

func EncodeVTT(cues []Cue) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, c := range cues {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n", vttStamp(c.Start), vttStamp(c.End), strings.TrimSpace(c.Text))
	}
	return b.String()
}

func ParseSRT(raw string) []Cue {
	blocks := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n\n")
	var out []Cue
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 2 {
			continue
		}
		idx := 0
		if _, err := strconv.Atoi(strings.TrimSpace(lines[0])); err == nil {
			idx = 1
		}
		if idx >= len(lines) {
			continue
		}
		m := reSRTTime.FindStringSubmatch(lines[idx])
		if m == nil {
			continue
		}
		text := strings.TrimSpace(strings.Join(lines[idx+1:], "\n"))
		out = append(out, Cue{
			Start: srtSeconds(m[1], m[2], m[3], m[4]),
			End:   srtSeconds(m[5], m[6], m[7], m[8]),
			Text:  text,
		})
	}
	return out
}

func CuesFromText(text string, dur float64) []Cue {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := splitSentences(text)
	if len(parts) == 0 {
		return nil
	}
	if dur <= 0 {
		dur = float64(len(parts)) * 2.4
	}
	total := 0
	weights := make([]int, len(parts))
	for i, p := range parts {
		n := utf8.RuneCountInString(p)
		if n < 4 {
			n = 4
		}
		weights[i] = n
		total += n
	}
	var cues []Cue
	t := 0.0
	for i, p := range parts {
		span := dur * float64(weights[i]) / float64(total)
		if span < 1.2 {
			span = 1.2
		}
		cues = append(cues, Cue{Start: t, End: t + span, Text: p})
		t += span
	}
	return cues
}

func splitSentences(s string) []string {
	s = strings.ReplaceAll(s, "。", "。\n")
	s = strings.ReplaceAll(s, "！", "！\n")
	s = strings.ReplaceAll(s, "？", "？\n")
	s = strings.ReplaceAll(s, ". ", ".\n")
	var out []string
	for _, p := range strings.Split(s, "\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func srtStamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	ms := int(sec*1000 + 0.5)
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func vttStamp(sec float64) string {
	return strings.Replace(srtStamp(sec), ",", ".", 1)
}

func srtSeconds(h, m, s, frac string) float64 {
	hh, _ := strconv.Atoi(h)
	mm, _ := strconv.Atoi(m)
	ss, _ := strconv.Atoi(s)
	if len(frac) > 3 {
		frac = frac[:3]
	}
	for len(frac) < 3 {
		frac += "0"
	}
	ms, _ := strconv.Atoi(frac)
	return float64(hh*3600+mm*60+ss) + float64(ms)/1000
}

func (b Bin) BurnCaptions(video, srtPath, outPath string) error {
	if err := b.ok(); err != nil {
		return err
	}
	slash := filepath.ToSlash(srtPath)
	slash = strings.ReplaceAll(slash, `\`, `/`)
	slash = strings.ReplaceAll(slash, ":", `\:`)
	vf := "subtitles='" + strings.ReplaceAll(slash, "'", `\'`) + "'"
	_, err := b.run("-y", "-i", video, "-vf", vf, "-c:a", "copy", outPath)
	return err
}

// Transcribe runs whisper.cpp when present and returns cues + raw text.
func (b Bin) Transcribe(src, workDir string) (cues []Cue, text string, err error) {
	if strings.TrimSpace(b.Whisper) == "" {
		return nil, "", fmt.Errorf("whisper is not available — set WHISPER_BIN or install whisper.cpp")
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, "", err
	}
	out := filepath.Join(workDir, "out")
	cmd := quiet(exec.Command(b.Whisper, "-f", src, "-otxt", "-osrt", "-of", out))
	if raw, runErr := cmd.CombinedOutput(); runErr != nil {
		return nil, "", fmt.Errorf("whisper: %s", strings.TrimSpace(string(raw)))
	}
	txt, _ := os.ReadFile(out + ".txt")
	text = strings.TrimSpace(string(txt))
	srt, _ := os.ReadFile(out + ".srt")
	cues = ParseSRT(string(srt))
	if len(cues) == 0 && text != "" {
		dur, _ := b.Duration(src)
		cues = CuesFromText(text, dur)
	}
	return cues, text, nil
}
