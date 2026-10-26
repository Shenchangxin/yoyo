// wrap.go 给口播成片烧上标题。

package video

import (
	"fmt"
	"strings"
)

// WrapTalkingHead 把标题烧进一条内容寻址视频。托管的 Remotion 工程就绪时渲染 TitleCard；渲染失败或工程未就绪时，退回 FFmpeg 烧字幕。
func (e *Engine) WrapTalkingHead(hash, title string) (MediaRef, error) {
	hash = strings.TrimSpace(hash)
	title = strings.TrimSpace(title)
	if hash == "" {
		return MediaRef{}, fmt.Errorf("hash required")
	}
	if title == "" {
		return MediaRef{}, fmt.Errorf("title required")
	}
	if e.managedRemotionReady() {
		ref, err := e.RenderTitleCard(hash, title, "")
		if err == nil {
			return ref, nil
		}
	}
	return e.wrapFFmpeg(hash, title)
}

// wrapFFmpeg 把标题做成一条 SRT，再调用 BurnInCaptions，来源记为 studio-wrap。
func (e *Engine) wrapFFmpeg(hash, title string) (MediaRef, error) {
	srt := simpleSRT(title)
	if srt == "" {
		return MediaRef{}, fmt.Errorf("empty title")
	}
	sh, err := e.PutBytes([]byte(srt))
	if err != nil {
		return MediaRef{}, err
	}
	ref, err := e.BurnInCaptions(hash, sh)
	if err != nil {
		return MediaRef{}, err
	}
	ref.Source = "studio-wrap"
	return ref, nil
}
