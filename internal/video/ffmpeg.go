package video

import (
	"bytes"
	"context"
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
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func LookFFmpeg() (ffmpeg, ffprobe string) {
	return LookFFmpegIn(nil)
}

func LookFFmpegIn(extra []string) (ffmpeg, ffprobe string) {
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
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		cands = append(cands, filepath.Join(local, "yoyo", "video", "bin"))
	}
	cands = append(cands,
		`C:\ffmpeg\bin`,
		`C:\Program Files\ffmpeg\bin`,
		`/usr/local/bin`,
		`/opt/homebrew/bin`,
	)
	name, probe := "ffmpeg", "ffprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
		probe += ".exe"
	}
	if ffmpeg == "" {
		ffmpeg = lookFile(name, extra)
	}
	if ffprobe == "" {
		ffprobe = lookFile(probe, extra)
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
	return lookFile(name, dirs)
}

func lookFile(name string, dirs []string) string {
	for _, d := range dirs {
		if strings.TrimSpace(d) == "" {
			continue
		}
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
	return e.ConcatContext(context.Background(), paths)
}

func (e *Engine) ConcatContext(ctx context.Context, paths []string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.EnsureFFmpeg(ctx); err != nil {
		return nil, err
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
	cmd := exec.CommandContext(ctx, e.FFMPEG, "-y", "-f", "concat", "-safe", "0", "-i", list,
		"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
		"-c:a", "aac", "-ar", "48000", "-b:a", "192k", "-movflags", "+faststart", out)
	cmd.Dir = tmp
	if _, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		cmd = exec.CommandContext(ctx, e.FFMPEG, "-y", "-f", "concat", "-safe", "0", "-i", list,
			"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-an", "-movflags", "+faststart", out)
		cmd.Dir = tmp
		if outb, err := cmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("%s", ffmpegErr(outb, err))
		}
	}
	return os.ReadFile(out)
}

func (e *Engine) Poster(videoPath string) ([]byte, error) {
	if strings.TrimSpace(e.FFMPEG) == "" {
		return nil, fmt.Errorf("ffmpeg missing")
	}
	tmp, err := os.CreateTemp(e.Dir, "poster-*.jpg")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	// -ss before -i seeks to a keyframe instead of decoding from the start.
	// One small frame, no audio, and a hard timeout so a huge file cannot hang import.
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	attempts := [][]string{
		{"-y", "-ss", "0.3", "-i", videoPath, "-frames:v", "1", "-an", "-threads", "1", "-vf", "scale='min(640,iw)':-2", tmp.Name()},
		{"-y", "-ss", "0.3", "-i", videoPath, "-frames:v", "1", "-an", "-threads", "1", tmp.Name()},
	}
	var last error
	for _, args := range attempts {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		cmd := exec.CommandContext(ctx, e.FFMPEG, args...)
		if err := cmd.Run(); err != nil {
			last = err
			continue
		}
		b, err := os.ReadFile(tmp.Name())
		if err != nil || len(b) == 0 {
			if err == nil {
				err = fmt.Errorf("empty poster")
			}
			last = err
			continue
		}
		return b, nil
	}
	if last == nil {
		last = fmt.Errorf("poster failed")
	}
	return nil, last
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
