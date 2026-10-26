// remake.go 把有权使用的参考片读进现有剪辑工程，再换成操作者自己的画面。

package video

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// RemakeDraft 保存操作者有权使用的参考哈希，并把时长、镜头和字幕读进现有剪辑工程。标题空着时用「翻拍」。它不打开完成台，操作者要用明确动作再送过去。有 ffmpeg 时做分析；剪辑表还空且时长大于 0 时按 3 段、15-30 提议窗口。WHISPER 非空且已有源哈希时尝试转写。
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

// RemakeReplace 把画面指到操作者自己的媒体，并保留参考哈希和时间草稿。从参考片抄来的片段字节会被丢掉。只有 kind 为 remake 且已有参考哈希的工程接受替换。新哈希不能为空，也不能和参考哈希相同。
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

// retargetRemakeWindows 保留剪辑窗口的起止，清掉旧哈希。开始时间已经达到新时长的窗口丢掉，结束时间超出新时长的收到新时长。ffmpeg 可用时从新源片抽出窗口并写回哈希；抽出失败的窗口保持空哈希。
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
