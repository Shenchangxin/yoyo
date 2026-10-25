package video

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// RemakeDraft stores a reference the operator has rights to and reads length,
// shots, and captions into the existing cut project. It does not open the
// finish desk. The operator sends it there with an explicit action.
func (e *Engine) RemakeDraft(hash, title, _ string) (CutProject, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return CutProject{}, fmt.Errorf("reference hash required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "翻拍"
	}
	p, err := e.CreateCutProject(CutProject{Title: title, Kind: "remake"})
	if err != nil {
		return CutProject{}, err
	}
	p, err = e.CutIngest(p.ID, hash, title)
	if err != nil {
		return p, err
	}
	p.Kind = "remake"
	p.ReferenceHash = hash
	p, err = e.saveCut(p)
	if err != nil {
		return p, err
	}
	e.refreshFFmpeg()
	if e.FFmpegOK() {
		analyzed, aerr := e.CutAnalyze(p.ID)
		if aerr != nil {
			p.Note = aerr.Error()
			p.Kind = "remake"
			p.ReferenceHash = hash
			return e.saveCut(p)
		}
		p = analyzed
	}
	if e.FFmpegOK() && p.Duration > 0 && len(p.EditList.Cuts) == 0 {
		proposed, perr := e.CutPropose(p.ID, 3, "15-30")
		if perr != nil {
			p.Note = perr.Error()
			p.Kind = "remake"
			p.ReferenceHash = hash
			return e.saveCut(p)
		}
		p = proposed
	}
	if strings.TrimSpace(e.WHISPER) != "" && p.SourceHash != "" {
		if spoken, terr := e.CutTranscribe(p.ID); terr == nil {
			p = spoken
		}
	}
	p.Kind = "remake"
	p.ReferenceHash = hash
	return e.saveCut(p)
}

// RemakeReplace points the picture at the operator's own media and keeps the
// reference hash plus the timed draft. Clip bytes copied from the reference
// are dropped.
func (e *Engine) RemakeReplace(id, hash string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if p.Kind != "remake" || strings.TrimSpace(p.ReferenceHash) == "" {
		return p, fmt.Errorf("only a remake draft accepts a replacement picture")
	}
	hash = strings.TrimSpace(hash)
	if hash == "" || hash == p.ReferenceHash {
		return p, fmt.Errorf("replacement hash required")
	}
	p.SourceHash = hash
	p.Stems.Original = hash
	p.ResultHash = ""
	p.Stale = false
	if poster, perr := e.posterFromHash(hash); perr == nil {
		p.PosterHash = poster
	}
	if d := e.probeCutDuration(hash); d > 0 {
		p.Duration = d
	}
	p, err = e.saveCut(p)
	if err != nil {
		return p, err
	}
	return e.retargetRemakeWindows(p)
}

func (e *Engine) retargetRemakeWindows(p CutProject) (CutProject, error) {
	windows := append([]cut.Clip(nil), p.EditList.Cuts...)
	if len(windows) == 0 {
		windows = append([]cut.Clip(nil), p.Breakdown.Clips...)
	}
	kept := make([]cut.Clip, 0, len(windows))
	for _, w := range windows {
		if p.Duration > 0 && w.Start >= p.Duration {
			continue
		}
		if p.Duration > 0 && w.End > p.Duration {
			w.End = p.Duration
		}
		w.Hash = ""
		kept = append(kept, w)
	}
	p.EditList.Cuts = kept
	p.Breakdown.Clips = kept
	p, err := e.saveCut(p)
	if err != nil || len(kept) == 0 {
		return p, err
	}
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		return p, nil
	}
	src, err := e.materialize(p.SourceHash, ".mp4")
	if err != nil {
		return p, nil
	}
	defer os.Remove(src)
	tmp, err := os.MkdirTemp(e.Dir, "remake-retarget-*")
	if err != nil {
		return p, nil
	}
	defer os.RemoveAll(tmp)
	out := make([]cut.Clip, 0, len(kept))
	for i, c := range kept {
		path := filepath.Join(tmp, fmt.Sprintf("c%d.mp4", i))
		if err := e.cutBin().Extract(src, c.Start, c.Dur(), path); err != nil {
			out = append(out, c)
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			out = append(out, c)
			continue
		}
		h, err := e.PutBytes(raw)
		if err != nil {
			out = append(out, c)
			continue
		}
		c.Hash = h
		out = append(out, c)
	}
	p.EditList.Cuts = out
	p.Breakdown.Clips = out
	return e.saveCut(p)
}
