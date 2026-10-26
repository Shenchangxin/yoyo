// 短片字幕排版和静帧滤镜。SRT 转成 ASS 时按画面短边排字，中文按字宽折行。
// 运镜只认 ken-burns，其余都当静止。

package video

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// srtToASS 把 SRT 收成 ASS。排版沿用 Easel assemble.py 的规则（Apache-2.0）：PlayRes 等于画面，字号大约是短边的百分之五，中文按测出来的宽度折行，不靠 libass 的默认折行。
// 宽或高小于 16 时按 1080×1920。没有字幕条就返回空串。
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

// subtitleFont 按操作系统挑字幕字体。Windows 用微软雅黑，macOS 用苹方，其余用 Noto Sans CJK SC。
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

// assStamp 把秒写成 ASS 的时:分:秒.厘秒。负数当成 0。
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

// assEscape 把花括号转义，避免被 ASS 当成特效。
func assEscape(s string) string {
	s = strings.ReplaceAll(s, "{", `\{`)
	s = strings.ReplaceAll(s, "}", `\}`)
	return s
}

// charUnits 估算一个字占多宽。中日韩字符算 1，其余算 0.5。
func charUnits(r rune) float64 {
	if r > 0x2E80 {
		return 1
	}
	return 0.5
}

// wrapCaption 按宽度把一句字幕折成多行。英文尽量在空格处断开。折完是空的就退回整句。
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

// alignScriptToCues 保留已锁定的口播句子，时间向 whisper 的字幕借。条数相同就逐条对应。条数不同就按比例把识别条并进每一句。任一边是空的就返回 nil。
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

// normalizeMotion 只把 ken-burns 留成推拉。别的运镜，包括空的，都收成 static。
func normalizeMotion(motion string) string {
	if strings.EqualFold(strings.TrimSpace(motion), "ken-burns") {
		return "ken-burns"
	}
	return "static"
}

// shotFit 决定这一镜是铺满还是留边。镜头自己写了 cover 或 contain 就听镜头，否则听分镜。都没写时，静止留边，推拉铺满，避免把黑边放大。
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

// frameFilter 拼缩放滤镜。cover 先放大再裁切，其余先缩小再补边。输出 30 帧。
func frameFilter(w, h int, fit string) string {
	if fit == "cover" {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,setsar=1,fps=30", w, h, w, h)
	}
	return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30", w, h, w, h)
}

// stillFilter 给静帧拼滤镜。推拉且不是留边时用 zoompan，先放大再缓推。其余走 frameFilter。
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

// clipShouldLoop 判断素材比镜头短时要不要循环。循环后仍盖不住一半以上的时长时，第二个结果为真，合成时会提醒操作者。目标时长小于等于 0 就都不循环。素材时长未知时循环，但不提醒。
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

// edgeRate 把语速收成 edge-tts 的百分比。接近 1 或小于等于 0 写成 +0%。结果夹在 -50% 到 +50%。
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

// clampSpeed 把语速夹在 0.5 到 2。小于等于 0 返回 0，表示请求里不写语速。
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

// speechPayload 拼 OpenAI 语音请求，格式是 mp3。声音空着用 alloy。语速接近 1 或小于等于 0 时不写 speed。
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
