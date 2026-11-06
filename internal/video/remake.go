// remake.go 把有权使用的参考片读进现有剪辑工程，再换成操作者自己的画面。

package video

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// RemakeDraft 保存操作者有权使用的参考哈希，并把时长、镜头和字幕读进现有剪辑工程。标题空着时用「翻拍」。它不打开完成台，操作者要用明确动作再送过去。有 ffmpeg 时做分析；剪辑表还空且时长大于 0 时按 3 段、15-30 提议窗口。WHISPER 非空且已有源哈希时尝试转写。分析或提议失败时把原因写进备注，草稿照样返回。
// hash 去掉首尾空白，空串时返回错误 reference hash required；否则传给 CutIngest，并写入 ReferenceHash。
// title 去掉首尾空白，空串时写成「翻拍」，再写入 CreateCutProject 的 Title，并传给 CutIngest。
func (e *Engine) RemakeDraft(hash, title, _ string) (CutProject, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return CutProject{}, fmt.Errorf("reference hash required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "翻拍"
	}
	p, err := e.CreateCutProject(CutProject{Title: title, Kind: "remake", ReferenceHash: hash})
	if err != nil {
		return CutProject{}, err
	}
	p, err = e.CutIngest(p.ID, hash, title)
	if err != nil {
		return p, err
	}
	note := func(msg string) (CutProject, error) {
		return e.mutateCut(p.ID, func(cur *CutProject) error {
			cur.Note = msg
			return nil
		})
	}
	e.refreshFFmpeg()
	if e.FFmpegOK() {
		analyzed, aerr := e.CutAnalyze(p.ID)
		if aerr != nil {
			return note(aerr.Error())
		}
		p = analyzed
	}
	if e.FFmpegOK() && p.Duration > 0 && len(p.EditList.Cuts) == 0 {
		proposed, perr := e.CutPropose(p.ID, 3, "15-30")
		if perr != nil {
			return note(perr.Error())
		}
		p = proposed
	}
	if strings.TrimSpace(e.WHISPER) != "" && p.SourceHash != "" {
		if spoken, terr := e.CutTranscribe(p.ID); terr == nil {
			p = spoken
		}
	}
	return p, nil
}

// RemakeReplace 把画面指到操作者自己的媒体，并保留参考哈希和时间草稿。从参考片抄来的片段字节会被丢掉，成片清空。只有 kind 为 remake 且已有参考哈希的工程接受替换。新哈希不能为空，也不能和参考哈希相同。封面和时长在锁外读。
// id 传给 GetCutProject 和 mutateCut。
// hash 去掉首尾空白，空串或等于 ReferenceHash 时返回错误 replacement hash required；否则写入 SourceHash 和 Stems.Original，并传给 posterFromHash 和 probeCutDuration。
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
	poster, _ := e.posterFromHash(hash)
	dur := e.probeCutDuration(hash)
	p, err = e.mutateCut(id, func(cur *CutProject) error {
		cur.SourceHash = hash
		cur.Stems.Original = hash
		cur.ResultHash = ""
		cur.Stale = false
		if poster != "" {
			cur.PosterHash = poster
		}
		if dur > 0 {
			cur.Duration = dur
		}
		cur.EditList.Cuts = retargetWindows(*cur)
		cur.Breakdown.Clips = cur.EditList.Cuts
		return nil
	})
	if err != nil {
		return p, err
	}
	return e.extractRemakeWindows(p)
}

// retargetWindows 保留剪辑窗口的起止，清掉旧哈希。开始时间已经达到新时长的窗口丢掉，结束时间超出新时长的收到新时长。
func retargetWindows(p CutProject) []cut.Clip {
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
	return kept
}

// extractRemakeWindows 在 ffmpeg 可用时从新源片抽出每个窗口并写回哈希。抽出失败的窗口保持空哈希，合成时再从源片截。抽取在锁外做。
func (e *Engine) extractRemakeWindows(p CutProject) (CutProject, error) {
	if len(p.EditList.Cuts) == 0 {
		return p, nil
	}
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		return p, nil
	}
	src, cleanup, err := e.mediaPath(p.SourceHash, ".mp4")
	if err != nil {
		return p, nil
	}
	defer cleanup()
	tmp, err := os.MkdirTemp(e.Dir, "remake-retarget-*")
	if err != nil {
		return p, nil
	}
	defer os.RemoveAll(tmp)
	hashes := make(map[int]string, len(p.EditList.Cuts))
	for i, c := range p.EditList.Cuts {
		path := filepath.Join(tmp, fmt.Sprintf("c%d.mp4", i))
		if err := e.cutBin().Extract(src, c.Start, c.Dur(), path); err != nil {
			continue
		}
		if h, _, err := e.PutPath(path); err == nil {
			hashes[i] = h
		}
	}
	source := p.SourceHash
	return e.mutateCut(p.ID, func(cur *CutProject) error {
		if cur.SourceHash != source || len(cur.EditList.Cuts) != len(p.EditList.Cuts) {
			return errUnchanged
		}
		for i := range cur.EditList.Cuts {
			if h := hashes[i]; h != "" && cur.EditList.Cuts[i].Hash == "" {
				cur.EditList.Cuts[i].Hash = h
			}
		}
		cur.Breakdown.Clips = append([]cut.Clip(nil), cur.EditList.Cuts...)
		return nil
	})
}
