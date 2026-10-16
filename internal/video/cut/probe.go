package cut

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	reScenePTS   = regexp.MustCompile(`pts_time:([0-9.]+)`)
	reSilenceSt  = regexp.MustCompile(`silence_start:\s*([0-9.]+)`)
	reSilenceEnd = regexp.MustCompile(`silence_end:\s*([0-9.]+)`)
	reDuration   = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
)

// Duration returns media length in seconds.
func (b Bin) Duration(path string) (float64, error) {
	if err := b.probeOK(); err != nil {
		return 0, err
	}
	probe := strings.TrimSpace(b.FFprobe)
	if probe != "" && !strings.Contains(strings.ToLower(probe), "ffmpeg") {
		out, err := b.probe("-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
		if err == nil {
			s := strings.TrimSpace(string(out))
			s = strings.TrimSuffix(s, "\r")
			if i := strings.IndexByte(s, '\n'); i >= 0 {
				s = s[:i]
			}
			d, err := strconv.ParseFloat(s, 64)
			if err == nil && d > 0 {
				return d, nil
			}
		}
	}
	if err := b.ok(); err != nil {
		return 0, err
	}
	out, _ := b.run("-hide_banner", "-i", path)
	m := reDuration.FindSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("bad duration")
	}
	h, _ := strconv.Atoi(string(m[1]))
	min, _ := strconv.Atoi(string(m[2]))
	sec, _ := strconv.ParseFloat(string(m[3]), 64)
	d := float64(h*3600+min*60) + sec
	if d <= 0 {
		return 0, fmt.Errorf("bad duration")
	}
	return d, nil
}

// Scenes returns cut points (seconds) from ffmpeg scene detect.
func (b Bin) Scenes(path string, threshold float64) ([]float64, error) {
	if err := b.ok(); err != nil {
		return nil, err
	}
	if threshold <= 0 {
		threshold = 0.28
	}
	filter := fmt.Sprintf("select='gt(scene,%.3f)',showinfo", threshold)
	out, _ := b.run("-hide_banner", "-i", path, "-vf", filter, "-an", "-f", "null", "-")
	var times []float64
	seen := map[string]bool{}
	for _, m := range reScenePTS.FindAllSubmatch(out, -1) {
		t, err := strconv.ParseFloat(string(m[1]), 64)
		if err != nil || t <= 0 {
			continue
		}
		key := fmt.Sprintf("%.2f", t)
		if seen[key] {
			continue
		}
		seen[key] = true
		times = append(times, t)
	}
	if times == nil {
		times = []float64{}
	}
	return times, nil
}

// Silences returns muted ranges from silencedetect.
func (b Bin) Silences(path string) ([]Silence, error) {
	if err := b.ok(); err != nil {
		return nil, err
	}
	out, _ := b.run("-hide_banner", "-i", path, "-af", "silencedetect=noise=-35dB:d=0.4", "-f", "null", "-")
	text := string(out)
	starts := reSilenceSt.FindAllStringSubmatch(text, -1)
	ends := reSilenceEnd.FindAllStringSubmatch(text, -1)
	n := len(starts)
	if len(ends) < n {
		n = len(ends)
	}
	var outS []Silence
	for i := 0; i < n; i++ {
		st, _ := strconv.ParseFloat(starts[i][1], 64)
		en, _ := strconv.ParseFloat(ends[i][1], 64)
		if en > st {
			outS = append(outS, Silence{Start: st, End: en})
		}
	}
	if outS == nil {
		outS = []Silence{}
	}
	return outS, nil
}

// Analyze fills a Breakdown from one file.
func (b Bin) Analyze(path string) (Breakdown, error) {
	bd := Breakdown{}
	d, err := b.Duration(path)
	if err != nil {
		return bd, err
	}
	bd.Duration = d
	if sc, err := b.Scenes(path, 0.28); err == nil {
		bd.Scenes = sc
	}
	if sil, err := b.Silences(path); err == nil {
		bd.Silences = sil
	}
	return bd, nil
}
