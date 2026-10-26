// extract.go 按时间抽出片段、拼接，并加黑边改成目标画幅。

package cut

import (
	"fmt"
	"os"
	"path/filepath"
)

// Extract 把一段时间写到 outPath，并尽量保留音频。视频用 libx264、veryfast、crf 20，音频用 aac、48 kHz、160k。带音频的那次失败时改为不带音频再抽一次。时长小于等于 0 时返回 empty extract。
// src 作为 -i。
// start 格式化成三位小数后传给 -ss。
// dur 小于等于 0 时返回 empty extract，否则格式化成三位小数后传给 -t。
func (b Bin) Extract(src string, start, dur float64, outPath string) error {
	if err := b.ok(); err != nil {
		return err
	}
	if dur <= 0 {
		return fmt.Errorf("empty extract")
	}
	_, err := b.run("-y", "-ss", fmt.Sprintf("%.3f", start), "-t", fmt.Sprintf("%.3f", dur),
		"-i", src, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
		"-c:a", "aac", "-ar", "48000", "-b:a", "160k", "-movflags", "+faststart", outPath)
	if err != nil {
		_, err = b.run("-y", "-ss", fmt.Sprintf("%.3f", start), "-t", fmt.Sprintf("%.3f", dur),
			"-i", src, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
			"-an", "-movflags", "+faststart", outPath)
	}
	return err
}

// ConcatKeepAudio 用 concat 列表拼接文件。有音频时用 aac、48 kHz、192k；失败则去掉音频再拼。视频用 libx264、medium、crf 23。没有片段时返回 no clips。
// paths 长度为 0 时返回 no clips；否则逐条经 escapeConcat 写进 concat 列表。
// outPath 作为两次 b.run 的输出路径，它所在目录用来放 cut-concat.txt。
func (b Bin) ConcatKeepAudio(paths []string, outPath string) error {
	if err := b.ok(); err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no clips")
	}
	dir := filepath.Dir(outPath)
	list := filepath.Join(dir, "cut-concat.txt")
	var body string
	for _, p := range paths {
		slash := filepath.ToSlash(p)
		body += fmt.Sprintf("file '%s'\n", escapeConcat(slash))
	}
	if err := os.WriteFile(list, []byte(body), 0o600); err != nil {
		return err
	}
	_, err := b.run("-y", "-f", "concat", "-safe", "0", "-i", list,
		"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
		"-c:a", "aac", "-ar", "48000", "-b:a", "192k", "-movflags", "+faststart", outPath)
	if err != nil {
		_, err = b.run("-y", "-f", "concat", "-safe", "0", "-i", list,
			"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-an", "-movflags", "+faststart", outPath)
	}
	return err
}

// escapeConcat 把路径里的单引号写成 ffmpeg concat 列表能接受的形式。
func escapeConcat(p string) string {
	out := ""
	for _, r := range p {
		if r == '\'' {
			out += `'\''`
			continue
		}
		out += string(r)
	}
	return out
}

// PadAspect 加黑边到目标画幅。默认 1080×1920；16:9 是 1920×1080，1:1 是 1080×1080。先按比例缩小再居中填充。复制音频失败时改为不带音频。
// src 作为 -i。
// outPath 作为两次 b.run 的输出路径。
// aspect 为 16:9 时宽高是 1920 和 1080，为 1:1 时是 1080 和 1080，其余是 1080 和 1920。
func (b Bin) PadAspect(src, outPath, aspect string) error {
	if err := b.ok(); err != nil {
		return err
	}
	w, h := 1080, 1920
	switch aspect {
	case "16:9":
		w, h = 1920, 1080
	case "1:1":
		w, h = 1080, 1080
	}
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", w, h, w, h)
	_, err := b.run("-y", "-i", src, "-vf", vf, "-c:a", "copy", outPath)
	if err != nil {
		_, err = b.run("-y", "-i", src, "-vf", vf, "-an", outPath)
	}
	return err
}
