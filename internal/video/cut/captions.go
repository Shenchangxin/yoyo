// captions.go 编码、解析和烧录字幕，并在有 whisper.cpp 时转写音轨。

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

// reSRTTime 匹配 SRT 的起止时间，毫秒分隔符可以是逗号或点。
var reSRTTime = regexp.MustCompile(`(\d+):(\d+):(\d+)[,.](\d+)\s*-->\s*(\d+):(\d+):(\d+)[,.](\d+)`)

// EncodeSRT 写成 SRT。空白文本跳过，序号从 1 起，毫秒用逗号。
// cues 逐条跳过空白文本，序号用循环下标加 1，起止传给 srtStamp。
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

// EncodeVTT 写成以 WEBVTT 开头的文本，毫秒用点号。空白文本跳过。
// cues 逐条跳过空白文本，起止传给 vttStamp。
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

// ParseSRT 按空行分块。块首如果是序号就跳过，下一行解析时间轴，其余行是文本。
// raw 把 \r\n 换成 \n 后按空行分块。
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

// CuesFromText 按句切分并把时长按字数摊开。每句至少 1.2 秒，不足 4 个字按 4 个字计。dur 小于等于 0 时用句数乘 2.4 秒。空文本返回 nil。
// text 去掉首尾空白，空串返回 nil，否则传给 splitSentences。
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

// splitSentences 在中文句号、叹号、问号以及英文句点加空格处断开。
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

// srtStamp 把秒格式化成时:分:秒,毫秒。负秒当 0，毫秒四舍五入。
// sec 小于 0 时改成 0，再乘 1000 加 0.5 后取整成毫秒。
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

// vttStamp 把 SRT 时间戳里的逗号换成点。
// sec 传给 srtStamp，再把结果里的逗号换成点。
func vttStamp(sec float64) string {
	return strings.Replace(srtStamp(sec), ",", ".", 1)
}

// srtSeconds 把时、分、秒和小数段合成秒。小数段超过 3 位就截断，不足 3 位就补 0。
// h 转成整数后乘 3600。
// m 转成整数后乘 60。
// frac 长于 3 位就截到 3 位，短于 3 位就补 0，再除以 1000。
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

// BurnCaptions 用 subtitles 滤镜把 SRT 烧进画面，音频复制。路径里的反斜杠、冒号和单引号会转义。
// video 作为 -i。
// srtPath 经 ToSlash，反斜杠、冒号和单引号转义后放进 subtitles 滤镜。
// outPath 作为输出路径。
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

// Transcribe 在 whisper 路径非空时运行 whisper.cpp，参数是 -f、-otxt、-osrt 和 -of。读出 out.txt 与 out.srt。没有字幕但有文本时，用成片时长把文本切成字幕。路径为空时返回错误，提示设置 WHISPER_BIN 或安装 whisper.cpp。
// src 传给 whisper 的 -f；没有字幕但有文本时再传给 Duration。
// workDir 传给 MkdirAll，并与 out 拼成输出前缀。
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
