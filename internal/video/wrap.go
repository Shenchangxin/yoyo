package video

import (
	"fmt"
	"strings"
)

// WrapTalkingHead burns a title onto a CAS video. When the operator-installed
// Remotion project is ready, it renders the TitleCard composition; otherwise
// it falls back to FFmpeg SRT burn-in.
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
