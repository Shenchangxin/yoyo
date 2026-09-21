package video

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func LookFFmpeg() (ffmpeg, ffprobe string) {
	if v := os.Getenv("FFMPEG_BIN"); v != "" {
		ffmpeg = v
	}
	if v := os.Getenv("FFPROBE_BIN"); v != "" {
		ffprobe = v
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	cands := []string{dir, filepath.Join(dir, "bin"), filepath.Join(dir, "resources")}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, filepath.Join(home, ".yoyo", "video", "bin"))
	}
	name, probe := "ffmpeg", "ffprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
		probe += ".exe"
	}
	if ffmpeg == "" {
		ffmpeg = look(name, cands)
	}
	if ffprobe == "" {
		ffprobe = look(probe, cands)
	}
	return ffmpeg, ffprobe
}

func look(name string, dirs []string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, d := range dirs {
		p := filepath.Join(d, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func (e *Engine) FFmpegOK() bool {
	return e.FFMPEG != ""
}

func (e *Engine) Concat(paths []string) ([]byte, error) {
	if !e.FFmpegOK() {
		return nil, fmt.Errorf("ffmpeg is not available — set FFMPEG_BIN or install ffmpeg")
	}
	for i, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("clip %d is missing on disk", i+1)
		}
	}
	tmp, err := os.MkdirTemp(e.Dir, "merge-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	list := filepath.Join(tmp, "list.txt")
	var b strings.Builder
	for _, p := range paths {
		slash := filepath.ToSlash(p)
		fmt.Fprintf(&b, "file '%s'\n", strings.ReplaceAll(slash, `'`, `'\''`))
	}
	if err := os.WriteFile(list, []byte(b.String()), 0o600); err != nil {
		return nil, err
	}
	out := filepath.Join(tmp, "out.mp4")
	cmd := exec.Command(e.FFMPEG, "-y", "-f", "concat", "-safe", "0", "-i", list, "-c:v", "libx264", "-crf", "23", "-c:a", "aac", "-b:a", "192k", "-movflags", "+faststart", out)
	cmd.Dir = tmp
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg concat failed")
	}
	return os.ReadFile(out)
}

func (e *Engine) Poster(videoPath string) ([]byte, error) {
	if e.FFProbe == "" && e.FFMPEG == "" {
		return nil, fmt.Errorf("ffmpeg missing")
	}
	tmp, err := os.CreateTemp(e.Dir, "poster-*.jpg")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	cmd := exec.Command(e.FFMPEG, "-y", "-i", videoPath, "-ss", "00:00:00.3", "-vframes", "1", tmp.Name())
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp.Name())
}

func CompressJPEG(src []byte, maxEdge int, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty image")
	}
	if maxEdge > 0 && (w > maxEdge || h > maxEdge) {
		if w > h {
			h = h * maxEdge / w
			w = maxEdge
		} else {
			w = w * maxEdge / h
			h = maxEdge
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}
	if quality <= 0 {
		quality = 68
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DataURLJPEG(jpegBytes []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes)
}
