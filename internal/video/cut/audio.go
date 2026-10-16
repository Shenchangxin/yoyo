package cut

import (
	"fmt"
	"strings"
)

type MixOpts struct {
	SpeechVol   float64
	MusicVol    float64
	OrigVol     float64
	Duck        bool
	ReplaceOrig bool
}

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

// MixGraph is a pure filter builder so mix can be tested without FFmpeg.
// Inputs: 0=video, 1=speech (optional), 2=music (optional).
func MixGraph(hasSpeech, hasMusic bool, o MixOpts) (filter string, maps []string) {
	o = NormalizeMix(o)
	switch {
	case hasSpeech && hasMusic:
		if o.Duck {
			filter = fmt.Sprintf("[2:a]volume=%.2f[bg];[bg][1:a]sidechaincompress=threshold=0.04:ratio=8:attack=40:release=280:makeup=1[ducked];[1:a]volume=%.2f[vo];[vo][ducked]amix=inputs=2:duration=first:dropout_transition=2:normalize=0[a]", o.MusicVol, o.SpeechVol)
		} else {
			filter = fmt.Sprintf("[1:a]volume=%.2f[vo];[2:a]volume=%.2f[bg];[vo][bg]amix=inputs=2:duration=first:dropout_transition=2[a]", o.SpeechVol, o.MusicVol)
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
			filter = fmt.Sprintf("[1:a]volume=%.2f[a]", o.MusicVol)
			return filter, []string{"-map", "0:v:0", "-map", "[a]"}
		}
		filter = fmt.Sprintf("[0:a]volume=%.2f[bed];[1:a]volume=%.2f[bg];[bed][bg]amix=inputs=2:duration=first:dropout_transition=2[a]", o.OrigVol, o.MusicVol)
		return filter, []string{"-map", "0:v:0", "-map", "[a]"}
	default:
		return "", nil
	}
}

// Mix mixes speech and/or music onto video. Duck compresses the bed under voice.
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
	// Shift music to input 1 when there is no speech.
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

// ExtractAudio dumps the soundtrack.
func (b Bin) ExtractAudio(src, outPath string) error {
	if err := b.ok(); err != nil {
		return err
	}
	_, err := b.run("-y", "-i", src, "-vn", "-acodec", "aac", "-ar", "48000", "-b:a", "192k", outPath)
	return err
}

// EnhanceAudio applies a named filter chain.
func (b Bin) EnhanceAudio(src, outPath, preset string) error {
	if err := b.ok(); err != nil {
		return err
	}
	_, err := b.run("-y", "-i", src, "-af", EnhanceFilter(preset), outPath)
	return err
}
