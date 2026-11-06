// audio.go 构造混音滤镜，并把口播或音乐混进画面。

package cut

import (
	"fmt"
	"strings"
)

// MixOpts 是混音音量、是否闪避、是否替换原声，以及音乐淡出和时间线长度。
type MixOpts struct {
	SpeechVol   float64
	MusicVol    float64
	OrigVol     float64
	Duck        bool
	ReplaceOrig bool
	// FadeOut 在时间线末尾把音乐淡出这么多秒。
	FadeOut  float64
	Timeline float64
	// Pad 为真时音轨末尾补静音，成片长度跟着画面走。只有输出另外限定了时长时才打开，否则补出来的静音没有尽头。
	Pad bool
}

// NormalizeMix 把小于等于 0 的语音音量改成 1，把小于等于 0 的音乐和原声音量改成 0.22。
func NormalizeMix(o MixOpts) MixOpts {
	if o.SpeechVol <= 0 {
		o.SpeechVol = 1
	}
	if o.MusicVol <= 0 {
		o.MusicVol = 0.22
	}
	if o.OrigVol <= 0 {
		o.OrigVol = 0.22
	}
	return o
}

// MixGraph 只构造滤镜，方便不跑 ffmpeg 做测试。输入 0 是画面，1 是可选口播，2 是可选音乐（没有口播时音乐是 1）。
// 垫底是原声和音乐：ReplaceOrig 为假时原声按 OrigVol 留着，音乐按 MusicVol 叠上。有口播时垫底在 Duck 为真时经 sidechaincompress 压在人声下，再和人声混在一起。Pad 为真时最后补静音。
// 输出标签是 [a]，映射画面 0:v:0 和 [a]。口播和音乐都没有时返回空滤镜和 nil。
// hasSpeech 为真时口播走输入 1。
// hasMusic 为真时音乐走输入 2；没有口播时走输入 1。
func MixGraph(hasSpeech, hasMusic bool, o MixOpts) (filter string, maps []string) {
	o = NormalizeMix(o)
	if !hasSpeech && !hasMusic {
		return "", nil
	}
	var parts []string
	bed := ""
	if !o.ReplaceOrig {
		parts = append(parts, fmt.Sprintf("[0:a]volume=%.2f[orig]", o.OrigVol))
		bed = "orig"
	}
	if hasMusic {
		in := 1
		if hasSpeech {
			in = 2
		}
		parts = append(parts, musicChain(in, o)+"[bg]")
		if bed == "" {
			bed = "bg"
		} else {
			parts = append(parts, "[orig][bg]amix=inputs=2:duration=longest:dropout_transition=0:normalize=0[bed]")
			bed = "bed"
		}
	}
	out := bed
	if hasSpeech {
		switch {
		case bed == "":
			parts = append(parts, fmt.Sprintf("[1:a]volume=%.2f[vo]", o.SpeechVol))
			out = "vo"
		case o.Duck:
			parts = append(parts,
				fmt.Sprintf("[1:a]volume=%.2f,asplit=2[vo][key]", o.SpeechVol),
				fmt.Sprintf("[%s][key]sidechaincompress=threshold=0.04:ratio=8:attack=40:release=280:makeup=1[ducked]", bed),
				"[vo][ducked]amix=inputs=2:duration=longest:dropout_transition=0:normalize=0[mix]",
			)
			out = "mix"
		default:
			parts = append(parts,
				fmt.Sprintf("[1:a]volume=%.2f[vo]", o.SpeechVol),
				fmt.Sprintf("[vo][%s]amix=inputs=2:duration=longest:dropout_transition=0:normalize=0[mix]", bed),
			)
			out = "mix"
		}
	}
	tail := "anull"
	if o.Pad {
		tail = "apad"
	}
	parts = append(parts, fmt.Sprintf("[%s]%s[a]", out, tail))
	return strings.Join(parts, ";"), []string{"-map", "0:v:0", "-map", "[a]"}
}

// musicChain 是带标签的音乐支路。只有 FadeOut 大于 0 且 Timeline 比淡出长出 0.05 秒以上才加 afade，否则只调音量。
// input 写进滤镜标签的输入号。
// o 的 FadeOut 大于 0 且 Timeline 大于 FadeOut 加 0.05 时加 afade，音量用 MusicVol。
func musicChain(input int, o MixOpts) string {
	if o.FadeOut > 0 && o.Timeline > o.FadeOut+0.05 {
		st := o.Timeline - o.FadeOut
		return fmt.Sprintf("[%d:a]afade=t=out:st=%.2f:d=%.2f,volume=%.2f", input, st, o.FadeOut, o.MusicVol)
	}
	return fmt.Sprintf("[%d:a]volume=%.2f", input, o.MusicVol)
}

// Mix 把口播和/或音乐混进画面。两者都空时返回 nothing to mix。画面复制，音频编成 aac、48 kHz。
// 能读出画面时长时：音乐循环到底，音轨补静音，输出按画面时长截止，配音短也不会把片子剪短。读不出时长时退回 -shortest。
// 画面本身没有音轨时不垫原声。滤镜失败时改走 mixFallback。
// video 作为第一个 -i。
// speech 去掉空白后非空才追加为 -i，并参与 MixGraph。
// music 去掉空白后非空才追加为 -i，并参与 MixGraph。
// outPath 是输出；失败时原样传给 mixFallback。
func (b Bin) Mix(video, speech, music, outPath string, o MixOpts) error {
	if err := b.ok(); err != nil {
		return err
	}
	o = NormalizeMix(o)
	hasSpeech := strings.TrimSpace(speech) != ""
	hasMusic := strings.TrimSpace(music) != ""
	if !hasSpeech && !hasMusic {
		return fmt.Errorf("nothing to mix")
	}
	if !o.ReplaceOrig && !b.HasAudio(video) {
		o.ReplaceOrig = true
	}
	length, _ := b.Duration(video)
	if length > 0 {
		o.Pad = true
		if o.Timeline <= 0 {
			o.Timeline = length
		}
	}
	args := []string{"-y", "-i", video}
	if hasSpeech {
		args = append(args, "-i", speech)
	}
	if hasMusic {
		if length > 0 {
			args = append(args, "-stream_loop", "-1")
		}
		args = append(args, "-i", music)
	}
	filter, maps := MixGraph(hasSpeech, hasMusic, o)
	args = append(args, "-filter_complex", filter)
	args = append(args, maps...)
	args = append(args, "-c:v", "copy", "-c:a", "aac", "-ar", "48000")
	args = append(args, lengthArgs(length)...)
	args = append(args, outPath)
	if _, err := b.run(args...); err != nil {
		if b.Ctx != nil && b.Ctx.Err() != nil {
			return err
		}
		if ferr := b.mixFallback(video, speech, music, outPath, o, length); ferr != nil {
			return fmt.Errorf("%v; fallback: %v", err, ferr)
		}
	}
	return nil
}

// lengthArgs 有时长时用 -t 截到画面长度，否则用 -shortest。
func lengthArgs(length float64) []string {
	if length > 0 {
		return []string{"-t", fmt.Sprintf("%.3f", length)}
	}
	return []string{"-shortest"}
}

// mixFallback 用音量混合重试，不再做闪避和淡出。画面复制，音频 aac，长度规则和 Mix 一样。
// video 作为第一个 -i。
// speech 非空才追加为 -i；与 music 同为非空时音量用 SpeechVol，单独非空时映射 1:a:0。
// music 非空才追加为 -i；单独非空时音量用 MusicVol。
// length 是画面秒数，传给 lengthArgs。
func (b Bin) mixFallback(video, speech, music, outPath string, o MixOpts, length float64) error {
	args := []string{"-y", "-i", video}
	if speech != "" {
		args = append(args, "-i", speech)
	}
	if music != "" {
		args = append(args, "-i", music)
	}
	switch {
	case speech != "" && music != "":
		filter := fmt.Sprintf("[1:a]volume=%.2f[vo];[2:a]volume=%.2f[bg];[vo][bg]amix=inputs=2:duration=longest[a]", o.SpeechVol, o.MusicVol)
		args = append(args, "-filter_complex", filter, "-map", "0:v:0", "-map", "[a]")
	case speech != "":
		args = append(args, "-map", "0:v:0", "-map", "1:a:0")
	default:
		filter := fmt.Sprintf("[1:a]volume=%.2f[a]", o.MusicVol)
		args = append(args, "-filter_complex", filter, "-map", "0:v:0", "-map", "[a]")
	}
	args = append(args, "-c:v", "copy", "-c:a", "aac")
	args = append(args, lengthArgs(length)...)
	_, err := b.run(append(args, outPath)...)
	return err
}

// EnhanceFilter 返回 denoise、loudnorm、loudness、voice、enhance 的音频滤镜。其他预设用 loudnorm，目标 I=-16、TP=-1.5、LRA=11。
// preset 去掉空白并转小写后匹配 denoise、loudnorm、loudness、voice、enhance，其余走 default。
func EnhanceFilter(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "denoise":
		return "afftdn=nr=10:nf=-25"
	case "loudnorm", "loudness":
		return "loudnorm=I=-16:TP=-1.5:LRA=11"
	case "voice", "enhance":
		return "highpass=f=80,afftdn=nr=8:nf=-25,loudnorm=I=-16:TP=-1.5:LRA=11"
	default:
		return "loudnorm=I=-16:TP=-1.5:LRA=11"
	}
}

// ExtractAudio 丢掉画面，把音轨抽成 aac、48 kHz、192k。
// src 作为 -i。
// outPath 作为输出路径。
func (b Bin) ExtractAudio(src, outPath string) error {
	if err := b.ok(); err != nil {
		return err
	}
	_, err := b.run("-y", "-i", src, "-vn", "-acodec", "aac", "-ar", "48000", "-b:a", "192k", outPath)
	return err
}

// EnhanceAudio 对音轨套用 EnhanceFilter 给出的滤镜。
// src 作为 -i。
// outPath 作为输出路径。
// preset 传给 EnhanceFilter。
func (b Bin) EnhanceAudio(src, outPath, preset string) error {
	if err := b.ok(); err != nil {
		return err
	}
	_, err := b.run("-y", "-i", src, "-af", EnhanceFilter(preset), outPath)
	return err
}
