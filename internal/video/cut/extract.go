package cut

import (
	"fmt"
	"os"
	"path/filepath"
)

// Extract writes a time window to outPath, keeping audio.
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

// ConcatKeepAudio concatenates files with audio when present.
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

// PadAspect letterboxes to 9:16 / 16:9 / 1:1.
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
