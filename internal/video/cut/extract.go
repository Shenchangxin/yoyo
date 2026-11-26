// extract.go 按时间抽出片段、拼接，并加黑边改成目标画幅。

package cut

import (
	"fmt"
	"os"
	"path/filepath"
)

// Extract 把一段时间写到 outPath，并保留音频。视频用 libx264、veryfast、crf 20，音频用 aac、48 kHz、160k。源片本来就没有音轨时才不带音频；有音轨却抽失败就报错，不悄悄出一段无声的片子。时长小于等于 0 时返回 empty extract。
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
	audio := []string{"-c:a", "aac", "-ar", "48000", "-b:a", "160k"}
	if !b.HasAudio(src) {
		audio = []string{"-an"}
	}
	args := append([]string{"-y", "-ss", fmt.Sprintf("%.3f", start), "-t", fmt.Sprintf("%.3f", dur),
		"-i", src, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20"}, audio...)
	_, err := b.run(append(args, "-movflags", "+faststart", outPath)...)
	return err
}

// ConcatKeepAudio 用 concat 列表拼接文件。有音频时用 aac、48 kHz、192k；第一段本来就没有音轨时才不带音频。视频用 libx264、medium、crf 23。没有片段时返回 no clips。
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
	audio := []string{"-c:a", "aac", "-ar", "48000", "-b:a", "192k"}
	if !b.HasAudio(paths[0]) {
		audio = []string{"-an"}
	}
	args := append([]string{"-y", "-f", "concat", "-safe", "0", "-i", list,
		"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23"}, audio...)
	_, err := b.run(append(args, "-movflags", "+faststart", outPath)...)
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
