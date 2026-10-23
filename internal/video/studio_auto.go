package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

const autoShotWait = 12 * time.Minute

// StartStudioAuto drafts or locks the script, speaks, fills missing stills, and assembles.
// The click is the spend confirmation. Work continues in the background; poll auto_phase.
func (e *Engine) StartStudioAuto(id string) (StudioProject, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return StudioProject{}, fmt.Errorf("id required")
	}
	key := "studio-auto:" + id
	if _, loaded := e.inflight.LoadOrStore(key, true); loaded {
		return e.GetStudioProject(id)
	}
	p, err := e.GetStudioProject(id)
	if err != nil {
		e.inflight.Delete(key)
		return StudioProject{}, err
	}
	p.AutoPhase = "script"
	p.AutoError = ""
	e.noteStudio(&p, "auto", "started")
	saved, err := e.saveStudio(p)
	if err != nil {
		e.inflight.Delete(key)
		return StudioProject{}, err
	}
	go e.runStudioAuto(id)
	return saved, nil
}

func (e *Engine) runStudioAuto(id string) {
	defer e.inflight.Delete("studio-auto:" + id)
	defer func() {
		if recover() != nil {
			e.finishAuto(id, "failed", "一键成片中断了。可以再点一次。")
		}
	}()
	if err := e.autoScript(id); err != nil {
		e.finishAuto(id, "failed", err.Error())
		return
	}
	e.markAuto(id, "voice")
	if _, err := e.SpeakStudio(id, ""); err != nil {
		p, gerr := e.GetStudioProject(id)
		if gerr == nil {
			e.warnStudio(&p, GateAudio, "配音没成功，这一版先不配音。"+err.Error())
			_, _ = e.saveStudio(p)
		}
		if _, serr := e.SkipStudioAudio(id); serr != nil {
			e.finishAuto(id, "failed", err.Error())
			return
		}
	}
	e.markAuto(id, "visuals")
	if err := e.autoVisuals(id); err != nil {
		e.finishAuto(id, "failed", err.Error())
		return
	}
	e.markAuto(id, "compose")
	if _, err := e.AssembleStudio(id); err != nil {
		e.finishAuto(id, "failed", err.Error())
		return
	}
	e.finishAuto(id, "done", "")
}

func (e *Engine) autoScript(id string) error {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.Script) == "" {
		text, derr := e.draftStudioScript(p)
		if derr != nil {
			return derr
		}
		p.Script = text
		p.ScriptStatus = ScriptNeedsReview
	}
	e.syncShotsToScript(&p)
	if _, err := e.saveStudio(p); err != nil {
		return err
	}
	if _, err := e.ApproveStudioScript(id); err != nil {
		return fmt.Errorf("口播还锁不上：%s", err.Error())
	}
	return nil
}

func (e *Engine) syncShotsToScript(p *StudioProject) {
	if p == nil || strings.TrimSpace(p.Script) == "" {
		return
	}
	old := p.Storyboard.Shots
	p.Segments = parseSpokenSegments(p.Script, p.TargetDuration)
	next := shotsFromSegments(p.Segments)
	for i := range next {
		for _, o := range old {
			if strings.TrimSpace(o.Caption) != strings.TrimSpace(next[i].Caption) {
				continue
			}
			if o.ImageHash == "" && o.VideoHash == "" {
				continue
			}
			next[i].ImageHash = o.ImageHash
			next[i].VideoHash = o.VideoHash
			next[i].Motion = firstNonEmpty(o.Motion, next[i].Motion)
			next[i].Fit = o.Fit
			next[i].Stale = o.Stale
			if strings.TrimSpace(o.Visual) != "" {
				next[i].Visual = o.Visual
			}
			if strings.TrimSpace(o.Prompt) != "" {
				next[i].Prompt = o.Prompt
			}
			break
		}
	}
	p.Storyboard.Shots = next
	if p.Storyboard.ImageMotion == "" {
		p.Storyboard.ImageMotion = "ken-burns"
	}
	if len(p.Segments) > 0 {
		p.Hook = p.Segments[0].Narration
		p.CTA = p.Segments[len(p.Segments)-1].Narration
	}
}

func (e *Engine) autoVisuals(id string) error {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return err
	}
	if p.VisualsReady() {
		return nil
	}
	if _, err := e.GenerateMissingStudioShots(id); err != nil && !hasStudioMedia(p) {
		return fmt.Errorf("画面还没齐。上传一张图，或到设置 → 生成里接上生图。")
	}
	return e.waitStudioShots(id)
}

func hasStudioMedia(p StudioProject) bool {
	for _, s := range p.Storyboard.Shots {
		if s.ImageHash != "" || s.VideoHash != "" {
			return true
		}
	}
	return false
}

func (e *Engine) waitStudioShots(id string) error {
	deadline := time.Now().Add(autoShotWait)
	for {
		if e.ctx != nil && e.ctx.Err() != nil {
			return fmt.Errorf("一键成片已取消")
		}
		p, err := e.GetStudioProject(id)
		if err != nil {
			return err
		}
		if p.VisualsReady() {
			return nil
		}
		pending := false
		for _, s := range p.Storyboard.Shots {
			if (s.ImageHash != "" || s.VideoHash != "") && !s.Stale {
				continue
			}
			if s.JobID == "" {
				continue
			}
			j, err := e.GetJob(s.JobID)
			if err != nil {
				continue
			}
			switch j.Status {
			case "queued", "running", "polling":
				pending = true
			}
		}
		if !pending {
			return fmt.Errorf("还有镜头没有画面。上传一张图，或到设置 → 生成里接上生图。")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("画面生成超时。可以稍后在分镜里重试还缺的镜头。")
		}
		timer := time.NewTimer(800 * time.Millisecond)
		if e.ctx != nil {
			select {
			case <-e.ctx.Done():
				timer.Stop()
				return fmt.Errorf("一键成片已取消")
			case <-timer.C:
			}
		} else {
			<-timer.C
		}
	}
}

func (e *Engine) draftStudioScript(p StudioProject) (string, error) {
	if e.Conn == nil {
		return "", fmt.Errorf("先写几句口播，或到设置 → 生成里接上对话模型。")
	}
	c, err := e.Conn.Active(connection.CapChat, "")
	if err != nil {
		return "", fmt.Errorf("先写几句口播，或到设置 → 生成里接上对话模型。")
	}
	key, err := e.Conn.Lease(c)
	if err != nil || strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("对话模型还没有密钥。先写几句口播，或到设置里补上。")
	}
	model := connection.ModelFor(c, connection.CapChat)
	if model == "" {
		model = "gpt-4.1-mini"
	}
	dur := p.TargetDuration
	if dur <= 0 {
		dur = 30
	}
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = "一条短片"
	}
	system := "你是短片口播作者。只输出口播正文，不要标题，不要分镜，不要 markdown。80 到 180 个汉字。一句钩子，两句正文，一句收束。按目标时长收束，不要写成文章。"
	user := fmt.Sprintf("主题：%s\n给谁看：%s\n目标时长约 %d 秒。", title, strings.TrimSpace(p.Audience), dur)
	if brief := clipRunes(p.Brief, 240); brief != "" {
		user += "\n可参考，但不要照念：" + brief
	}
	raw, err := e.chatComplete(c.Endpoint, key, model, system, user)
	if err != nil {
		return "", fmt.Errorf("口播没写出来：%s", err.Error())
	}
	text := cleanDraftScript(raw)
	if text == "" {
		return "", fmt.Errorf("对话模型没有返回口播。可以自己在口播栏写几句。")
	}
	return text, nil
}

func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n])
}

func cleanDraftScript(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "\n"); i >= 0 && strings.HasPrefix(strings.ToLower(s[:i]), "口播") {
		s = strings.TrimSpace(s[i+1:])
	}
	s = strings.TrimPrefix(s, "口播：")
	s = strings.TrimPrefix(s, "口播:")
	return strings.TrimSpace(s)
}

func (e *Engine) chatComplete(endpoint, key, model, system, user string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	u := base
	if !strings.HasSuffix(u, "/chat/completions") {
		u += "/chat/completions"
	}
	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	ctx := context.Background()
	if e.ctx != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(e.ctx, 45*time.Second)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	httpDo := e.HTTP
	if httpDo == nil {
		httpDo = &http.Client{Timeout: 45 * time.Second}
	}
	res, err := httpDo.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		return "", fmt.Errorf("chat %d: %s", res.StatusCode, msg)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty chat response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func (e *Engine) markAuto(id, phase string) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return
	}
	p.AutoPhase = phase
	p.AutoError = ""
	_, _ = e.saveStudio(p)
}

func (e *Engine) finishAuto(id, phase, errMsg string) {
	var packErr error
	if phase == "done" && errMsg == "" {
		if _, packErr = e.PackageStudio(id); packErr != nil {
			packErr = fmt.Errorf("成片好了，交付包没打上：%s", packErr.Error())
		}
	}
	p, err := e.GetStudioProject(id)
	if err != nil {
		return
	}
	p.AutoPhase = phase
	p.AutoError = errMsg
	if errMsg != "" {
		e.noteStudio(&p, "auto", errMsg)
	} else {
		e.noteStudio(&p, "auto", "ready")
		p.Gate = GateDelivery
		if packErr != nil {
			e.warnStudio(&p, GateDelivery, packErr.Error())
		}
		if p.TopicID != "" && p.ResultHash != "" {
			if t, terr := e.GetTopic(p.TopicID); terr == nil {
				t.Status = TopicDone
				_, _ = e.SaveTopic(t)
			}
		}
	}
	_, _ = e.saveStudio(p)
}
