package video

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

func parseSize(size, aspect string) (int, int) {
	size = strings.ToLower(strings.TrimSpace(size))
	if i := strings.Index(size, "x"); i > 0 {
		w, _ := strconv.Atoi(size[:i])
		h, _ := strconv.Atoi(size[i+1:])
		if w > 0 && h > 0 {
			return even(w), even(h)
		}
	}
	return aspectSize(aspect)
}

func even(n int) int {
	if n%2 != 0 {
		return n + 1
	}
	return n
}

func (e *Engine) materialize(hash, ext string) (string, error) {
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if src := e.casFile(hash); src != "" {
		in, err := os.Open(src)
		if err != nil {
			return "", err
		}
		defer in.Close()
		return writeTemp(e.Dir, ext, in)
	}
	// No disk object: the bytes are already in an in-memory store (tests).
	raw, err := e.GetBytes(hash)
	if err != nil {
		return "", err
	}
	return writeTemp(e.Dir, ext, bytes.NewReader(raw))
}

func writeTemp(dir, ext string, r io.Reader) (string, error) {
	f, err := os.CreateTemp(dir, "studio-*"+ext)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func (e *Engine) AssembleStudio(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	sb := p.Storyboard
	if p.NarrationHash != "" {
		sb.Narration = p.NarrationHash
	}
	if p.SubtitleHash != "" {
		sb.Subtitle = p.SubtitleHash
	}
	if p.BGMHash != "" {
		sb.BGM = p.BGMHash
	}
	if len(sb.Shots) == 0 {
		return StudioProject{}, fmt.Errorf("storyboard has no shots")
	}
	for _, shot := range sb.Shots {
		if shot.Stale && (strings.TrimSpace(shot.ImageHash) != "" || strings.TrimSpace(shot.VideoHash) != "") {
			return StudioProject{}, fmt.Errorf("shot %s is stale; regenerate before assemble", firstNonEmpty(shot.ID, shot.Caption))
		}
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return StudioProject{}, err
	}
	w, h := parseSize(sb.Size, p.Aspect)
	tmp, err := os.MkdirTemp(e.Dir, "assemble-*")
	if err != nil {
		return StudioProject{}, err
	}
	defer os.RemoveAll(tmp)

	var clips []string
	for i, shot := range sb.Shots {
		dur := shot.Duration
		if dur <= 0 {
			dur = 3
		}
		clip := filepath.Join(tmp, fmt.Sprintf("shot-%02d.mp4", i))
		motion := shot.Motion
		if motion == "" {
			motion = sb.ImageMotion
		}
		if shot.VideoHash != "" {
			src, err := e.materialize(shot.VideoHash, ".mp4")
			if err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
			defer os.Remove(src)
			vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30", w, h, w, h)
			cmd := exec.Command(e.FFMPEG, "-y", "-i", src, "-t", fmt.Sprintf("%.2f", dur), "-vf", vf,
				"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an", clip)
			if out, err := cmd.CombinedOutput(); err != nil {
				p.Status, p.Error = StudioFailed, ffmpegErr(out, err)
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, fmt.Errorf("%s", p.Error)
			}
		} else if shot.ImageHash != "" {
			src, err := e.materialize(shot.ImageHash, ".png")
			if err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
			defer os.Remove(src)
			vf := stillFilter(w, h, dur, motion)
			args := []string{"-y", "-loop", "1", "-i", src, "-t", fmt.Sprintf("%.2f", dur), "-vf", vf,
				"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an", clip}
			cmd := exec.Command(e.FFMPEG, args...)
			if out, err := cmd.CombinedOutput(); err != nil {
				p.Status, p.Error = StudioFailed, ffmpegErr(out, err)
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, fmt.Errorf("%s", p.Error)
			}
		} else {
			continue
		}
		clips = append(clips, clip)
	}
	if len(clips) == 0 {
		return StudioProject{}, fmt.Errorf("no usable shots (need image_hash or video_hash)")
	}
	merged, err := e.Concat(clips)
	if err != nil {
		p.Status, p.Error = StudioFailed, err.Error()
		_ = e.putDoc(colStudioProjects, p.ID, p)
		return p, err
	}
	base := filepath.Join(tmp, "base.mp4")
	if err := os.WriteFile(base, merged, 0o600); err != nil {
		return StudioProject{}, err
	}
	final := base
	if sb.Narration != "" || sb.BGM != "" {
		mixed, err := e.mixAudio(tmp, base, sb.Narration, sb.BGM, sb.BGMVolume)
		if err != nil {
			p.Status, p.Error = StudioFailed, err.Error()
			_ = e.putDoc(colStudioProjects, p.ID, p)
			return p, err
		}
		final = mixed
	}
	if sb.Subtitle != "" {
		burned, err := e.burnSubtitles(tmp, final, sb.Subtitle)
		if err == nil {
			final = burned
		}
	}
	raw, err := os.ReadFile(final)
	if err != nil {
		return StudioProject{}, err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return StudioProject{}, err
	}
	p.ResultHash = hash
	p.Status = StudioAssembled
	p.Error = ""
	p.Gate = GateCompose
	if poster, err := e.Poster(final); err == nil {
		if ph, err := e.PutBytes(poster); err == nil {
			p.PosterHash = ph
			if p.CoverHash == "" {
				p.CoverHash = ph
			}
		}
	}
	e.noteStudio(&p, "assemble", "cut ready")
	p.Versions = append(p.Versions, StudioSnapshot{
		N: len(p.Versions) + 1, ResultHash: p.ResultHash, PosterHash: p.PosterHash,
		NarrationHash: p.NarrationHash, SubtitleHash: p.SubtitleHash, CoverHash: p.CoverHash,
		Note: "assemble", CreatedAt: Now(),
	})
	saved, err := e.saveStudio(p)
	if err != nil {
		return p, err
	}
	published, err := e.PublishStudioAsset(saved)
	if err != nil {
		_ = e.putDoc(colStudioProjects, saved.ID, saved)
		return saved, err
	}
	return published, nil
}

func stillFilter(w, h int, dur float64, motion string) string {
	base := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30", w, h, w, h)
	if strings.ToLower(strings.TrimSpace(motion)) != "ken-burns" {
		return base
	}
	frames := int(dur * 30)
	if frames < 30 {
		frames = 30
	}
	return fmt.Sprintf("%s,zoompan=z='min(zoom+0.0008,1.12)':d=%d:s=%dx%d:fps=30", base, frames, w, h)
}

func (e *Engine) mixAudio(tmp, video, narrationHash, bgmHash string, bgmVol float64) (string, error) {
	if narrationHash == "" && bgmHash == "" {
		return video, nil
	}
	var narr, bgm string
	var err error
	if narrationHash != "" {
		narr, err = e.materialize(narrationHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(narr)
	}
	if bgmHash != "" {
		bgm, err = e.materialize(bgmHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(bgm)
	}
	if bgmVol <= 0 {
		bgmVol = 0.22
	}
	out := filepath.Join(tmp, "mixed.mp4")
	if err := e.cutBin().Mix(video, narr, bgm, out, cut.MixOpts{
		Duck:        narr != "" && bgm != "",
		ReplaceOrig: narr != "",
		MusicVol:    bgmVol,
		SpeechVol:   1,
	}); err != nil {
		return "", err
	}
	return out, nil
}

func (e *Engine) burnSubtitles(tmp, video, subHash string) (string, error) {
	src, err := e.materialize(subHash, ".srt")
	if err != nil {
		return "", err
	}
	defer os.Remove(src)
	out := filepath.Join(tmp, "sub.mp4")
	slash := filepath.ToSlash(src)
	slash = strings.ReplaceAll(slash, ":", `\:`)
	vf := "subtitles='" + strings.ReplaceAll(slash, "'", `\'`) + "'"
	cmd := exec.Command(e.FFMPEG, "-y", "-i", video, "-vf", vf, "-c:a", "copy", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s", ffmpegErr(b, err))
	}
	return out, nil
}

func (e *Engine) Reframe(hash, aspect string) (MediaRef, error) {
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return MediaRef{}, err
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.Remove(src)
	w, h := aspectSize(aspect)
	tmp, err := os.CreateTemp(e.Dir, "reframe-*.mp4")
	if err != nil {
		return MediaRef{}, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", w, h, w, h)
	cmd := exec.Command(e.FFMPEG, "-y", "-i", src, "-vf", vf, "-c:a", "copy", tmp.Name())
	if b, err := cmd.CombinedOutput(); err != nil {
		cmd = exec.Command(e.FFMPEG, "-y", "-i", src, "-vf", vf, "-an", tmp.Name())
		if b2, err2 := cmd.CombinedOutput(); err2 != nil {
			return MediaRef{}, fmt.Errorf("%s", ffmpegErr(append(b, b2...), err2))
		}
	}
	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return MediaRef{}, err
	}
	out, err := e.PutBytes(raw)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: out, MIME: "video/mp4", Width: w, Height: h, Source: "studio"}, nil
}

func (e *Engine) BurnInCaptions(hash, subHash string) (MediaRef, error) {
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return MediaRef{}, err
	}
	tmp, err := os.MkdirTemp(e.Dir, "burn-*")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.RemoveAll(tmp)
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.Remove(src)
	out, err := e.burnSubtitles(tmp, src, subHash)
	if err != nil {
		return MediaRef{}, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return MediaRef{}, err
	}
	cas, err := e.PutBytes(raw)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: cas, MIME: "video/mp4", Source: "studio"}, nil
}

func (e *Engine) Slideshow(hashes []string, aspect string, dur float64) (MediaRef, error) {
	p, err := e.CreateStudioProject(StudioProject{Title: "相册短片", Aspect: aspect})
	if err != nil {
		return MediaRef{}, err
	}
	if dur <= 0 {
		dur = 3
	}
	shots := make([]StudioShot, 0, len(hashes))
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		shots = append(shots, StudioShot{ImageHash: h, Duration: dur, Motion: "ken-burns"})
	}
	p.Storyboard.Shots = shots
	p.Storyboard.ImageMotion = "ken-burns"
	p, err = e.UpdateStudioProject(p)
	if err != nil {
		return MediaRef{}, err
	}
	p, err = e.AssembleStudio(p.ID)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: p.ResultHash, MIME: "video/mp4", PosterHash: p.PosterHash, Source: "studio", OriginID: p.ID}, nil
}

func ffmpegErr(out []byte, err error) string {
	tail := strings.TrimSpace(string(out))
	if n := strings.LastIndex(tail, "\n"); n >= 0 && len(tail) > 400 {
		lines := strings.Split(tail, "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		tail = strings.Join(lines, "\n")
	}
	if tail == "" {
		return err.Error()
	}
	return tail
}
