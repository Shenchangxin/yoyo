package video

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// Subtitle layout follows the Easel assemble.py rules (Apache-2.0):
// PlayRes equals the frame, font size is about 5% of the short side, and
// CJK lines wrap by measured width instead of relying on libass defaults.

func srtToASS(srt string, w, h int) string {
	if w < 16 {
		w = 1080
	}
	if h < 16 {
		h = 1920
	}
	cues := cut.ParseSRT(srt)
	if len(cues) == 0 {
		return ""
	}
	fs := int(math.Round(float64(min(w, h)) * 0.05))
	if fs < 18 {
		fs = 18
	}
	mv := int(math.Round(float64(h) * 0.07))
	mh := int(math.Round(float64(w) * 0.06))
	maxUnits := float64(w-2*mh) / float64(fs) * 0.94
	if maxUnits < 6 {
		maxUnits = 6
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Script Info]\nScriptType: v4.00+\nPlayResX: %d\nPlayResY: %d\nWrapStyle: 0\nScaledBorderAndShadow: yes\n\n", w, h)
	b.WriteString("[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	fmt.Fprintf(&b, "Style: Default,%s,%d,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,%d,%d,%d,1\n\n", subtitleFont(), fs, mh, mh, mv)
	b.WriteString("[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	for _, c := range cues {
		lines := wrapCaption(c.Text, maxUnits)
		if len(lines) == 0 {
			continue
		}
		text := assEscape(strings.Join(lines, "\n"))
		text = strings.ReplaceAll(text, "\n", `\N`)
		fmt.Fprintf(&b, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", assStamp(c.Start), assStamp(c.End), text)
	}
	return b.String()
}

func subtitleFont() string {
	switch runtime.GOOS {
	case "windows":
		return "Microsoft YaHei"
	case "darwin":
		return "PingFang SC"
	default:
		return "Noto Sans CJK SC"
	}
}

func assStamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	cs := int(sec*100 + 0.5)
	h := cs / 360000
	cs %= 360000
	m := cs / 6000
	cs %= 6000
	s := cs / 100
	cs %= 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func assEscape(s string) string {
	s = strings.ReplaceAll(s, "{", `\{`)
	s = strings.ReplaceAll(s, "}", `\}`)
	return s
}

func charUnits(r rune) float64 {
	if r > 0x2E80 {
		return 1
	}
	return 0.5
}

func wrapCaption(text string, maxUnits float64) []string {
	flat := strings.Join(strings.Fields(strings.ReplaceAll(text, `\N`, " ")), " ")
	if flat == "" {
		return nil
	}
	var out []string
	cur := ""
	curU := 0.0
	for _, r := range flat {
		u := charUnits(r)
		if cur != "" && curU+u > maxUnits {
			line := strings.TrimRight(cur, " ")
			if r != ' ' && strings.Contains(line, " ") {
				if i := strings.LastIndex(line, " "); i > 0 {
					out = append(out, line[:i])
					cur = strings.TrimSpace(line[i+1:])
					curU = 0
					for _, rr := range cur {
						curU += charUnits(rr)
					}
				} else {
					out = append(out, line)
					cur = ""
					curU = 0
				}
			} else {
				out = append(out, line)
				cur = ""
				curU = 0
			}
		}
		if r == ' ' && cur == "" {
			continue
		}
		cur += string(r)
		curU += u
	}
	if strings.TrimSpace(cur) != "" {
		out = append(out, strings.TrimRight(cur, " "))
	}
	if len(out) == 0 {
		return []string{flat}
	}
	return out
}

// alignScriptToCues keeps the approved spoken lines and borrows timing from whisper cues.
func alignScriptToCues(lines []string, cues []cut.Cue) []cut.Cue {
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	usable := make([]cut.Cue, 0, len(cues))
	for _, c := range cues {
		if c.End > c.Start {
			usable = append(usable, c)
		}
	}
	if len(clean) == 0 || len(usable) == 0 {
		return nil
	}
	if len(usable) == len(clean) {
		out := make([]cut.Cue, len(clean))
		for i := range clean {
			out[i] = usable[i]
			out[i].Text = clean[i]
		}
		return out
	}
	out := make([]cut.Cue, 0, len(clean))
	for i, line := range clean {
		a := i * len(usable) / len(clean)
		b := (i + 1) * len(usable) / len(clean)
		if b <= a {
			b = a + 1
		}
		if b > len(usable) {
			b = len(usable)
		}
		out = append(out, cut.Cue{Start: usable[a].Start, End: usable[b-1].End, Text: line})
	}
	return out
}

func normalizeMotion(motion string) string {
	if strings.EqualFold(strings.TrimSpace(motion), "ken-burns") {
		return "ken-burns"
	}
	return "static"
}

// shotFit: stills that hold text stay letterboxed. Ken Burns defaults to cover
// so the push does not enlarge the padded black bars.
func shotFit(sb StudioStoryboard, shot StudioShot, motion string) string {
	f := strings.ToLower(strings.TrimSpace(shot.Fit))
	if f == "" {
		f = strings.ToLower(strings.TrimSpace(sb.Fit))
	}
	switch f {
	case "contain":
		return "contain"
	case "cover":
		return "cover"
	}
	if motion == "static" {
		return "contain"
	}
	return "cover"
}

func frameFilter(w, h int, fit string) string {
	if fit == "cover" {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,setsar=1,fps=30", w, h, w, h)
	}
	return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30", w, h, w, h)
}

func stillFilter(w, h int, dur float64, motion, fit string) string {
	motion = normalizeMotion(motion)
	if motion == "ken-burns" && fit != "contain" {
		frames := int(dur * 30)
		if frames < 30 {
			frames = 30
		}
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,zoompan=z='min(zoom+0.0008,1.08)':d=%d:s=%dx%d:fps=30,setsar=1", w*2, h*2, w*2, h*2, frames, w, h)
	}
	return frameFilter(w, h, fit)
}

// clipShouldLoop reports whether a source shorter than the shot must be looped,
// and whether that loop repeats the picture more than once (worth telling the operator).
func clipShouldLoop(srcDur, want float64) (loop, warn bool) {
	if want <= 0 {
		return false, false
	}
	if srcDur <= 0 {
		return true, false
	}
	if srcDur+0.05 < want {
		return true, srcDur*2 < want
	}
	return false, false
}

func edgeRate(speed float64) string {
	speed = clampSpeed(speed)
	if speed == 0 || (speed >= 0.99 && speed <= 1.01) {
		return "+0%"
	}
	pct := int(math.Round((speed - 1) * 100))
	if pct > 50 {
		pct = 50
	}
	if pct < -50 {
		pct = -50
	}
	return fmt.Sprintf("%+d%%", pct)
}

func clampSpeed(speed float64) float64 {
	if speed <= 0 {
		return 0
	}
	if speed < 0.5 {
		return 0.5
	}
	if speed > 2 {
		return 2
	}
	return speed
}

func speechPayload(model, text, voice string, speed float64) map[string]any {
	if strings.TrimSpace(voice) == "" {
		voice = "alloy"
	}
	body := map[string]any{"model": model, "input": text, "voice": voice, "format": "mp3"}
	speed = clampSpeed(speed)
	if speed > 0 && (speed < 0.99 || speed > 1.01) {
		body["speed"] = speed
	}
	return body
}
