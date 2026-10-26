// 一键成片。先起草或锁住口播，再配音，补上缺的静帧，最后合成。
// 点下去就在后台做，看 auto_phase。配音失败会改成不配音再继续。

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

// StartStudioAuto 开始一键成片。编号空着就拒绝。这一条已经在做就返回当前工程，不另起一次。
// 点这一下就开始用对话、语音和图像。函数立刻返回，后台接着做。
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

// runStudioAuto 按口播、配音、画面、合成的顺序做完。配音失败就记警告并跳过配音。任一步失败把阶段写成 failed。中途恐慌也收成失败，避免挂住。
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

// autoScript 保证有口播并锁定。口播是空的就先起草。然后按口播重排镜头，已有画面按字幕对上的会留下。锁定失败就返回错误。
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

// syncShotsToScript 按口播重写分段和镜头。字幕对得上、且原来有图或视频的镜头，会把画面、运镜和过期标记抄回来。第一段写成钩子，最后一段写成收束。
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

// autoVisuals 给缺画面的镜头排队生图，然后等到它们回来。画面已经齐就直接返回。排队失败且一镜都没有时返回错误。
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

// hasStudioMedia 判断至少有一镜已经有图或视频。
func hasStudioMedia(p StudioProject) bool {
	for _, s := range p.Storyboard.Shots {
		if s.ImageHash != "" || s.VideoHash != "" {
			return true
		}
	}
	return false
}

// waitStudioShots 等到每一镜都有可用画面，或没有还在跑的任务。最多等 12 分钟。引擎取消时返回已取消。超时或仍缺画面就返回错误。
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

// draftStudioScript 用已启用的对话连接写口播。没有连接、没有密钥，或返回是空的，就要求先自己写。
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

// clipRunes 按字数截断。超过 n 个字就只留前 n 个。n 小于等于 0 时不截。
func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n])
}

// cleanDraftScript 去掉模型输出外面的代码围栏，以及开头的「口播」标记。
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

// chatComplete 向对话接口要一段文本。端点空着用 api.openai.com。路径还没到 /chat/completions 就补上。45 秒超时。没有 choices 就返回错误。
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

// markAuto 写下当前阶段并清掉错误。工程读不到就什么都不做。
func (e *Engine) markAuto(id, phase string) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return
	}
	p.AutoPhase = phase
	p.AutoError = ""
	_, _ = e.saveStudio(p)
}

// finishAuto 写下结束阶段。成功时先打交付包，包打不上就记警告，阶段仍是 done，闸门改到交付。挂了选题且已有成片时，把选题标成完成。失败时只记下原因。
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
