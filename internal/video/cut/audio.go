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

// MixGraph 只构造滤镜，方便不跑 ffmpeg 做测试。输入 0 是画面，1 是可选口播，2 是可选音乐。口播和音乐都在且 Duck 为真时，音乐经 sidechaincompress 压在人声下。只有口播且 ReplaceOrig 为真时不建滤镜，映射到 1:a:0。只有音乐且 ReplaceOrig 为真时音乐作为输入 1。两者都没有时滤镜和映射都空。
// hasSpeech 与 hasMusic 同为真时口播走输入 1；只有 hasSpeech 且 ReplaceOrig 为真时映射 1:a:0，否则和原声做 sidechaincompress。
// hasMusic 与 hasSpeech 同为真时 musicChain 的输入号是 2；只有 hasMusic 时输入号是 1。两者都为假时返回空滤镜和 nil。
func MixGraph(hasSpeech, hasMusic bool, o MixOpts) (filter string, maps []string) {
	o = NormalizeMix(o)
	switch {
	case hasSpeech && hasMusic:
		if o.Duck {
			filter = fmt.Sprintf("%s[bg];[bg][1:a]sidechaincompress=threshold=0.04:ratio=8:attack=40:release=280:makeup=1[ducked];[1:a]volume=%.2f[vo];[vo][ducked]amix=inputs=2:duration=first:dropout_transition=2:normalize=0[a]", musicChain(2, o), o.SpeechVol)
		} else {
			filter = fmt.Sprintf("[1:a]volume=%.2f[vo];%s[bg];[vo][bg]amix=inputs=2:duration=first:dropout_transition=2[a]", o.SpeechVol, musicChain(2, o))
		}
		return filter, []string{"-map", "0:v:0", "-map", "[a]"}
	case hasSpeech:
		if o.ReplaceOrig {
			return "", []string{"-map", "0:v:0", "-map", "1:a:0"}
		}
		filter = fmt.Sprintf("[0:a]volume=%.2f[bed];[bed][1:a]sidechaincompress=threshold=0.04:ratio=8:attack=40:release=280:makeup=1[ducked];[1:a]volume=%.2f[vo];[vo][ducked]amix=inputs=2:duration=first:normalize=0[a]", o.OrigVol, o.SpeechVol)
		return filter, []string{"-map", "0:v:0", "-map", "[a]"}
	case hasMusic:
		if o.ReplaceOrig {
			filter = musicChain(1, o) + "[a]"
			return filter, []string{"-map", "0:v:0", "-map", "[a]"}
		}
		filter = fmt.Sprintf("[0:a]volume=%.2f[bed];%s[bg];[bed][bg]amix=inputs=2:duration=first:dropout_transition=2[a]", o.OrigVol, musicChain(1, o))
		return filter, []string{"-map", "0:v:0", "-map", "[a]"}
	default:
		return "", nil
	}
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

// Mix 把口播和/或音乐混进画面。两者都空时返回 nothing to mix。画面复制，音频编成 aac、48 kHz，并加上 -shortest。滤镜失败时改走 mixFallback。
// video 作为第一个 -i。
// speech 去掉空白后非空才追加为 -i，并参与 MixGraph。
// music 去掉空白后非空才追加为 -i，并参与 MixGraph。
// outPath 接在 -shortest 后面；失败时原样传给 mixFallback。
func (b Bin) Mix(video, speech, music, outPath string, o MixOpts) error {
	if err := b.ok(); err != nil {
		return err
	}
	o = NormalizeMix(o)
	args := []string{"-y", "-i", video}
	hasSpeech := strings.TrimSpace(speech) != ""
	hasMusic := strings.TrimSpace(music) != ""
	if hasSpeech {
		args = append(args, "-i", speech)
	}
	if hasMusic {
		args = append(args, "-i", music)
	}
	if !hasSpeech && !hasMusic {
		return fmt.Errorf("nothing to mix")
	}
	// 没有口播时，音乐占输入 1。
	if !hasSpeech && hasMusic {
		filter, maps := MixGraph(false, true, o)
		if filter != "" {
			args = append(args, "-filter_complex", filter)
		}
		args = append(args, maps...)
	} else {
		filter, maps := MixGraph(hasSpeech, hasMusic, o)
		if filter != "" {
			args = append(args, "-filter_complex", filter)
		}
		args = append(args, maps...)
	}
	args = append(args, "-c:v", "copy", "-c:a", "aac", "-ar", "48000", "-shortest", outPath)
	if _, err := b.run(args...); err != nil {
		return b.mixFallback(video, speech, music, outPath, o)
	}
	return nil
}

// mixFallback 用音量混合重试，不再做闪避和淡出。画面复制，音频 aac，并加上 -shortest。
// video 作为第一个 -i。
// speech 非空才追加为 -i；与 music 同为非空时音量用 SpeechVol，单独非空时映射 1:a:0。
// music 非空才追加为 -i；单独非空时音量用 MusicVol。
// outPath 接在 -shortest 后面。
func (b Bin) mixFallback(video, speech, music, outPath string, o MixOpts) error {
	args := []string{"-y", "-i", video}
	if speech != "" {
		args = append(args, "-i", speech)
	}
	if music != "" {
		args = append(args, "-i", music)
	}
	switch {
	case speech != "" && music != "":
		filter := fmt.Sprintf("[1:a]volume=%.2f[vo];[2:a]volume=%.2f[bg];[vo][bg]amix=inputs=2:duration=first[a]", o.SpeechVol, o.MusicVol)
		args = append(args, "-filter_complex", filter, "-map", "0:v:0", "-map", "[a]")
	case speech != "":
		args = append(args, "-map", "0:v:0", "-map", "1:a:0")
	default:
		filter := fmt.Sprintf("[1:a]volume=%.2f[a]", o.MusicVol)
		args = append(args, "-filter_complex", filter, "-map", "0:v:0", "-map", "[a]")
	}
	args = append(args, "-c:v", "copy", "-c:a", "aac", "-shortest", outPath)
	_, err := b.run(args...)
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
