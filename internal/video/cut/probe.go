// probe.go 用 ffmpeg 和 ffprobe 读时长、场景切点和静音区间。

package cut

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 这些正则从 ffmpeg 输出里取场景时间、静音起止和时长。
var (
	// reScenePTS 匹配 pts_time 后面的秒数。
	reScenePTS   = regexp.MustCompile(`pts_time:([0-9.]+)`)
	// reSilenceSt 匹配 silence_start 后面的秒数。
	reSilenceSt  = regexp.MustCompile(`silence_start:\s*([0-9.]+)`)
	// reSilenceEnd 匹配 silence_end 后面的秒数。
	reSilenceEnd = regexp.MustCompile(`silence_end:\s*([0-9.]+)`)
	// reDuration 匹配 Duration 行里的时、分、秒。
	reDuration   = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
)

// Duration 返回媒体秒数。ffprobe 路径非空且路径里不含 ffmpeg 时读 format=duration；否则从 ffmpeg -i 的 Duration 行解析时分秒。
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

// Scenes 用 ffmpeg 的场景检测返回切点秒数。阈值小于等于 0 时用 0.28。小于等于 0 的时间丢掉，按两位小数去重。没有切点时返回空切片。
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

// Silences 用 silencedetect=noise=-35dB:d=0.4 返回静音区间。开始和结束按出现顺序配对，只保留结束晚于开始的段。没有静音时返回空切片。
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

// Analyze 读时长，并用阈值 0.28 填场景切点。静音检测成功时也写进结果。时长失败则整次失败；场景或静音失败时对应字段留空。
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
