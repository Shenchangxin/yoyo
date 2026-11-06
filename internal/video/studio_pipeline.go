// 短片的闸门、口播锁定、镜头时长和交付包。
// 能走到哪一闸，看口播、配音、画面和成片是否齐。合成本身不在这里。

package video

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// studioGateIndex 返回闸门在顺序里的位置。认不出的名字返回 0，也就是 GateSetup。
// g 去掉首尾空白后与 studioGateOrder 比较，没有相等项时返回 0。
func studioGateIndex(g string) int {
	g = strings.TrimSpace(g)
	for i, id := range studioGateOrder {
		if id == g {
			return i
		}
	}
	return 0
}

// FurthestGate 按现有材料给出最远能进的闸门。没锁定口播停在口播，没配音且没跳过停在配音，画面不齐停在画面，没成片停在合成，否则是交付。
func (p StudioProject) FurthestGate() string {
	if strings.TrimSpace(p.Script) == "" && p.ScriptStatus != ScriptApproved {
		return GateScript
	}
	if p.ScriptStatus != ScriptApproved {
		return GateScript
	}
	if p.NarrationHash == "" && !p.SkipAudio {
		return GateAudio
	}
	if !p.VisualsReady() {
		return GateVisuals
	}
	if p.ResultHash == "" {
		return GateCompose
	}
	return GateDelivery
}

// CanEnter 判断这个闸门没有超过 FurthestGate。
// gate 传给 studioGateIndex。
func (p StudioProject) CanEnter(gate string) bool {
	return studioGateIndex(gate) <= studioGateIndex(p.FurthestGate())
}

// VisualsReady 在每一镜都有图或视频、且没有过期时返回真。一镜都没有就返回假。
func (p StudioProject) VisualsReady() bool {
	shots := p.Storyboard.Shots
	if len(shots) == 0 {
		return false
	}
	for _, s := range shots {
		if strings.TrimSpace(s.ImageHash) == "" && strings.TrimSpace(s.VideoHash) == "" {
			return false
		}
		if s.Stale {
			return false
		}
	}
	return true
}

// ensureShotIDs 给缺编号的镜头补编号。运镜空着写成 ken-burns，画面说明空着就用提示，时长小于等于 0 就写成 3 秒。
// shots 逐项在 ID 去掉首尾空白后为空时用 NewID，Motion 为空时写成 ken-burns，Visual 为空时改用 Prompt，Duration 小于等于 0 时写成 3。
func ensureShotIDs(shots []StudioShot) []StudioShot {
	for i := range shots {
		if strings.TrimSpace(shots[i].ID) == "" {
			shots[i].ID = NewID()
		}
		if strings.TrimSpace(shots[i].Motion) == "" {
			shots[i].Motion = "ken-burns"
		}
		if strings.TrimSpace(shots[i].Visual) == "" {
			shots[i].Visual = shots[i].Prompt
		}
		if shots[i].Duration <= 0 {
			shots[i].Duration = 3
		}
	}
	return shots
}

// mergeStudioShots 把传进来的镜头并到现有镜头上。先按编号对；没带编号的再按字幕对，条数一样时按位置对。助手重写分镜时常常不带编号，这样已有的画面不会丢。
// 对上的镜头：没带画面说明或提示就沿用原来的；图和视频没换就留原来的；过期标记和任务编号以服务端为准，客户端手里的旧值不算。
// 换上了新的图或视频算手动换画面：清掉过期，任务编号记成 manualShotJob，之前排着的生图回来也不会盖掉它。画面说明变了而图或视频没换，就标成过期。
// 送回来的正是刚被生图换下的上一版时，说明客户端手里是旧数据，按没换处理，新落地的画面不被盖掉。
// cur 是现有镜头。
// next 是传进来的镜头，最后交给 ensureShotIDs 补齐编号和默认值。
func (e *Engine) mergeStudioShots(cur, next []StudioShot) []StudioShot {
	byID := make(map[string]int, len(cur))
	for i, s := range cur {
		if s.ID != "" {
			byID[s.ID] = i
		}
	}
	used := make([]bool, len(cur))
	sameCount := len(cur) == len(next)
	out := make([]StudioShot, 0, len(next))
	for k, s := range next {
		oi := -1
		if s.ID != "" {
			if i, ok := byID[s.ID]; ok && !used[i] {
				oi = i
			}
		} else {
			if caption := strings.TrimSpace(s.Caption); caption != "" {
				for i, o := range cur {
					if !used[i] && strings.TrimSpace(o.Caption) == caption {
						oi = i
						break
					}
				}
			}
			if oi < 0 && sameCount && !used[k] {
				oi = k
			}
		}
		if oi < 0 {
			s.Stale = false
			s.JobID = ""
			out = append(out, s)
			continue
		}
		used[oi] = true
		old := cur[oi]
		if s.ID == "" {
			s.ID = old.ID
		}
		if strings.TrimSpace(s.Visual) == "" {
			s.Visual = old.Visual
		}
		if strings.TrimSpace(s.Prompt) == "" {
			s.Prompt = old.Prompt
		}
		if strings.TrimSpace(s.Motion) == "" {
			s.Motion = old.Motion
		}
		if s.Duration <= 0 {
			s.Duration = old.Duration
		}
		visualChanged := strings.TrimSpace(s.Visual) != strings.TrimSpace(old.Visual) || strings.TrimSpace(s.Prompt) != strings.TrimSpace(old.Prompt)
		newImage := s.ImageHash != "" && s.ImageHash != old.ImageHash && s.ImageHash != old.PrevImage
		newVideo := s.VideoHash != "" && s.VideoHash != old.VideoHash && s.VideoHash != old.PrevVideo
		s.Stale, s.JobID = old.Stale, old.JobID
		s.PrevImage, s.PrevVideo = old.PrevImage, old.PrevVideo
		if newImage {
			s.PrevImage = firstNonEmpty(old.ImageHash, old.PrevImage)
		} else {
			s.ImageHash = old.ImageHash
		}
		if newVideo {
			s.PrevVideo = firstNonEmpty(old.VideoHash, old.PrevVideo)
		} else {
			s.VideoHash = old.VideoHash
		}
		switch {
		case newImage || newVideo:
			s.Stale = false
			s.JobID = manualShotJob
		case visualChanged && (s.ImageHash != "" || s.VideoHash != ""):
			s.Stale = true
		}
		out = append(out, s)
	}
	return ensureShotIDs(out)
}

// noteStudio 追加一条系统记录。超过 50 条就只留最新的 50 条。工程是空指针就什么都不做。
// p 为 nil 时直接返回；否则追加一条活动，超过 50 条时只留最后 50 条。
// kind 写入活动的 Kind。
// message 写入活动的 Message。
func (e *Engine) noteStudio(p *StudioProject, kind, message string) {
	if p == nil {
		return
	}
	p.Activities = append(p.Activities, StudioActivity{
		ID: NewID(), At: Now(), Actor: "system", Kind: kind, Message: message,
	})
	if len(p.Activities) > 50 {
		p.Activities = p.Activities[len(p.Activities)-50:]
	}
}

// flagStudio 记一条错误，级别是 error。
// p 传给 addStudioIssue。
// gate 传给 addStudioIssue。
// message 传给 addStudioIssue。
func (e *Engine) flagStudio(p *StudioProject, gate, message string) {
	e.addStudioIssue(p, "error", gate, message)
}

// warnStudio 记一条警告。同一闸、同一句话的旧警告会先去掉，避免重复。空消息不记。
// p 为 nil，或 message 去掉首尾空白后为空时直接返回。
// gate 用来丢掉同一闸、同一句的旧问题，并传给 addStudioIssue。
// message 去掉首尾空白后为空时直接返回，否则用来匹配旧问题并传给 addStudioIssue。
func (e *Engine) warnStudio(p *StudioProject, gate, message string) {
	if p == nil || strings.TrimSpace(message) == "" {
		return
	}
	kept := p.Issues[:0]
	for _, iss := range p.Issues {
		if iss.Gate == gate && iss.Message == message {
			continue
		}
		kept = append(kept, iss)
	}
	p.Issues = kept
	e.addStudioIssue(p, "warn", gate, message)
}

// clearStudioIssues 去掉某一闸上的旧问题。这一闸重新做成功时调用，旧的失败和提醒不再挂着。
func clearStudioIssues(p *StudioProject, gate string) {
	if p == nil {
		return
	}
	kept := p.Issues[:0]
	for _, iss := range p.Issues {
		if iss.Gate != gate {
			kept = append(kept, iss)
		}
	}
	p.Issues = kept
}

// addStudioIssue 追加一条问题。超过 20 条就只留最新的 20 条。空消息不记。
// p 为 nil，或 message 去掉首尾空白后为空时直接返回；否则追加问题，超过 20 条时只留最后 20 条。
// level 写入 Level。
// gate 写入 Gate。
// message 去掉首尾空白后为空时不记，否则写入 Message。
func (e *Engine) addStudioIssue(p *StudioProject, level, gate, message string) {
	if p == nil || strings.TrimSpace(message) == "" {
		return
	}
	p.Issues = append(p.Issues, StudioIssue{
		ID: NewID(), At: Now(), Level: level, Gate: gate, Message: message,
	})
	if len(p.Issues) > 20 {
		p.Issues = p.Issues[len(p.Issues)-20:]
	}
}

// saveStudio 把修订号加一并写回工程。编号必须已经有。只用在新建的工程，或已经拿着这条工程锁的地方；其余改动走 mutateStudio。
// p 把 Revision 加 1 并写入 UpdatedAt 后传给 putDoc。
func (e *Engine) saveStudio(p StudioProject) (StudioProject, error) {
	p.Revision++
	p.UpdatedAt = Now()
	if err := e.putDoc(colStudioProjects, p.ID, p); err != nil {
		return StudioProject{}, err
	}
	return p, nil
}

// mutateStudio 在这条短片的锁里读出最新的一份，交给 fn 改，再写回。fn 返回错误时不写，返回读到的那份和错误；返回 errUnchanged 时不写也不报错。
// 任务回调、一键成片和界面保存都走这里，免得后写的旧副本冲掉先落地的图、任务编号或口播。配音、生图、合成这类长活在锁外做完再用它写结果。fn 里不能再调用 mutateStudio。
// id 去掉首尾空白后用来加锁和读取。
func (e *Engine) mutateStudio(id string, fn func(p *StudioProject) error) (StudioProject, error) {
	id = strings.TrimSpace(id)
	unlock := e.lockDoc(colStudioProjects, id)
	defer unlock()
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if err := fn(&p); err != nil {
		if errors.Is(err, errUnchanged) {
			return p, nil
		}
		return p, err
	}
	return e.saveStudio(p)
}

// ApplyStudioPreset 把预设的画幅、平台和时长写上工程，并改分镜尺寸。预设不存在就拒绝。
// id 传给 GetStudioProject。
// presetID 传给 LookupStudioPreset，找不到时写进错误文本。
func (e *Engine) ApplyStudioPreset(id, presetID string) (StudioProject, error) {
	preset, ok := LookupStudioPreset(presetID)
	if !ok {
		return StudioProject{}, fmt.Errorf("没有这个预设：%s", presetID)
	}
	return e.mutateStudio(id, func(p *StudioProject) error {
		applyStudioPreset(p, preset)
		e.noteStudio(p, "preset", "locked "+preset.Name)
		return nil
	})
}

// applyStudioPreset 把预设的画幅、平台和时长写进工程，并按画幅重算分镜尺寸。
func applyStudioPreset(p *StudioProject, preset StudioPreset) {
	p.PresetID = preset.ID
	p.Aspect = preset.Aspect
	p.Platform = preset.Platform
	p.TargetDuration = preset.Duration
	w, h := aspectSize(p.Aspect)
	p.Storyboard.Size = fmt.Sprintf("%dx%d", w, h)
}

// ApplyStudioTemplate 先套模板上的预设，再写风格、平台、语气和身份。标题还是空的或「短片」时改成模板标题。模板不存在就拒绝。
// id 传给 GetStudioProject 和 ApplyStudioPreset。
// templateID 传给 LookupStudioTemplate，找不到时写进错误文本。
func (e *Engine) ApplyStudioTemplate(id, templateID string) (StudioProject, error) {
	tpl, ok := LookupStudioTemplate(templateID)
	if !ok {
		return StudioProject{}, fmt.Errorf("unknown template %q", templateID)
	}
	return e.mutateStudio(id, func(p *StudioProject) error {
		if preset, ok := LookupStudioPreset(tpl.PresetID); ok {
			applyStudioPreset(p, preset)
			e.noteStudio(p, "preset", "locked "+preset.Name)
		}
		p.TemplateID = tpl.ID
		p.Style = tpl.Style
		p.Platform = tpl.Platform
		p.Tone = tpl.Tone
		p.Identity = identityFromCatalog(tpl.CharacterID, tpl.Style)
		if p.Title == "" || p.Title == "短片" {
			p.Title = tpl.Title
		}
		e.noteStudio(p, "template", "started from "+tpl.Title)
		return nil
	})
}

// identityFromCatalog 用目录里的角色和风格拼身份。风格的约定和禁止会接在角色的后面。两边都没有就用默认口播主持。
// characterID 传给 LookupStudioCharacter，找到时写入角色编号、名字、Preserve 和 Prohibit。
// styleID 传给 LookupStudioStyle，找到时写入 StyleID，并把 Contract、Forbid 接在后面。
func identityFromCatalog(characterID, styleID string) StudioIdentity {
	ident := DefaultStudioIdentity()
	if c, ok := LookupStudioCharacter(characterID); ok {
		ident.CharacterID = c.ID
		ident.Name = c.Name
		ident.Preserve = c.Preserve
		ident.Prohibit = c.Prohibit
	}
	if s, ok := LookupStudioStyle(styleID); ok {
		ident.StyleID = s.ID
		ident.Preserve = strings.TrimSpace(ident.Preserve + "; " + s.Contract)
		ident.Prohibit = strings.TrimSpace(ident.Prohibit + "; " + s.Forbid)
	}
	return ident
}

// SetStudioIdentity 按目录重写身份。风格编号非空时也写到工程的 Style。
// id 传给 GetStudioProject。
// characterID 传给 identityFromCatalog。
// styleID 传给 identityFromCatalog，非空时同时写入工程的 Style。
func (e *Engine) SetStudioIdentity(id, characterID, styleID string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		p.Identity = identityFromCatalog(characterID, styleID)
		if styleID != "" {
			p.Style = styleID
		}
		e.noteStudio(p, "identity", p.Identity.Name)
		return nil
	})
}

// ApproveStudioScript 锁住工程里现有的口播。见 ApproveStudioScriptText。
// id 传给 ApproveStudioScriptText。
func (e *Engine) ApproveStudioScript(id string) (StudioProject, error) {
	return e.ApproveStudioScriptText(id, "")
}

// ApproveStudioScriptText 锁住口播。script 非空时先换成这份正文再锁，界面不必先等失焦保存落地。正文是空的就拒绝。
// 分段跟不上口播就重新按句拆。没有镜头就按分段生成；镜头是按旧口播自动拆出来的，就按新口播重拆，字幕对得上的旧镜头留下画面和提示。操作者或助手写的镜头不动。钩子和收束空着时用第一段和最后一段。闸门改到配音。
// id 传给 mutateStudio。
// script 去掉首尾空白后非空时写成新的口播。
func (e *Engine) ApproveStudioScriptText(id, script string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		if text := strings.TrimSpace(script); text != "" {
			e.setStudioScript(p, text)
		}
		p.Script = strings.TrimSpace(p.Script)
		if p.Script == "" {
			return fmt.Errorf("write a spoken script first")
		}
		e.followScript(p)
		p.ScriptStatus = ScriptApproved
		if studioGateIndex(p.Gate) < studioGateIndex(GateAudio) {
			p.Gate = GateAudio
		}
		e.noteStudio(p, "approve", "script locked")
		return nil
	})
}

// setStudioScript 换上新口播，并把正文放进内容库。正文变了而原来已锁定时退回待审。比较前两边都去掉首尾空白，只多一个换行不算改稿。
// p 写入 Script、ScriptHash，必要时改 ScriptStatus。
// text 是新正文，去掉首尾空白后写入。
func (e *Engine) setStudioScript(p *StudioProject, text string) {
	text = strings.TrimSpace(text)
	if strings.TrimSpace(p.Script) != text && p.ScriptStatus == ScriptApproved {
		p.ScriptStatus = ScriptNeedsReview
	}
	p.Script = text
	if text == "" {
		p.ScriptHash = ""
		p.ScriptStatus = ScriptNeedsReview
		return
	}
	if e.CAS != nil {
		if h, err := e.PutBytes([]byte(text)); err == nil {
			p.ScriptHash = h
		}
	}
}

// followScript 让分段和自动拆出的镜头跟上口播。分段和口播的句子对不上就重拆。没有镜头就按分段生成；镜头是按另一份口播自动拆的就重拆，字幕对得上的旧镜头留下画面、提示、运镜和适配。手写的镜头不动。钩子和收束空着时用第一段和最后一段。
// p 的 Script 为空时什么都不做。
func (e *Engine) followScript(p *StudioProject) {
	if p == nil || strings.TrimSpace(p.Script) == "" {
		return
	}
	if !segmentsMatchScript(p.Segments, p.Script) {
		p.Segments = parseSpokenSegments(p.Script, p.TargetDuration)
	}
	sum := scriptSum(p.Script)
	switch {
	case len(p.Storyboard.Shots) == 0:
		p.Storyboard.Shots = shotsFromSegments(p.Segments)
		p.ShotsFrom = sum
	case p.ShotsFrom != "" && p.ShotsFrom != sum:
		p.Storyboard.Shots = carryShots(p.Storyboard.Shots, shotsFromSegments(p.Segments))
		p.ShotsFrom = sum
	}
	if p.Storyboard.ImageMotion == "" {
		p.Storyboard.ImageMotion = "ken-burns"
	}
	if p.Hook == "" && len(p.Segments) > 0 {
		p.Hook = p.Segments[0].Narration
	}
	if p.CTA == "" && len(p.Segments) > 0 {
		p.CTA = p.Segments[len(p.Segments)-1].Narration
	}
}

// segmentsMatchScript 判断分段是不是按这份口播的句子拆出来的。条数和每一句都要对上。
func segmentsMatchScript(segs []StudioSegment, script string) bool {
	parts := splitSpeak(script)
	if len(parts) == 0 || len(parts) != len(segs) {
		return false
	}
	for i, part := range parts {
		if strings.TrimSpace(segs[i].Narration) != part {
			return false
		}
	}
	return true
}

// scriptSum 给口播正文一个短摘要，用来判断自动拆出的镜头是不是跟着这份口播。
func scriptSum(script string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(script)))
	return hex.EncodeToString(sum[:8])
}

// carryShots 用新拆出的镜头替换旧镜头。字幕对得上的旧镜头把图、视频、画面说明、提示、运镜、适配和过期标记抄过来，每一镜只抄一次。
// old 是现有镜头。
// next 是按新口播拆出的镜头。
func carryShots(old, next []StudioShot) []StudioShot {
	used := make([]bool, len(old))
	for i := range next {
		caption := strings.TrimSpace(next[i].Caption)
		for j, o := range old {
			if used[j] || strings.TrimSpace(o.Caption) != caption {
				continue
			}
			used[j] = true
			next[i].ImageHash = o.ImageHash
			next[i].VideoHash = o.VideoHash
			next[i].JobID = o.JobID
			next[i].Stale = o.Stale
			next[i].Fit = o.Fit
			next[i].Motion = firstNonEmpty(o.Motion, next[i].Motion)
			if strings.TrimSpace(o.Visual) != "" {
				next[i].Visual = o.Visual
			}
			if strings.TrimSpace(o.Prompt) != "" {
				next[i].Prompt = o.Prompt
			}
			break
		}
	}
	return next
}

// SkipStudioAudio 在口播已锁定后跳过配音，闸门改到画面。还没锁定就拒绝。
// id 传给 mutateStudio。
func (e *Engine) SkipStudioAudio(id string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		if p.ScriptStatus != ScriptApproved {
			return fmt.Errorf("先锁定口播")
		}
		p.SkipAudio = true
		if studioGateIndex(p.Gate) < studioGateIndex(GateVisuals) {
			p.Gate = GateVisuals
		}
		e.noteStudio(p, "skip-audio", "operator skipped voice")
		return nil
	})
}

// parseSpokenSegments 按句子拆口播。第一段是钩子，最后一段是收束，中间是正文。目标秒数空着按 60 秒，再平分到每段，每段至少 2 秒。
// script 传给 splitSpeak，结果为空时返回 nil。
// targetSec 小于等于 0 时写成 60，再除以句子数，每段小于 2 时写成 2。
func parseSpokenSegments(script string, targetSec int) []StudioSegment {
	parts := splitSpeak(script)
	if len(parts) == 0 {
		return nil
	}
	if targetSec <= 0 {
		targetSec = 60
	}
	each := float64(targetSec) / float64(len(parts))
	if each < 2 {
		each = 2
	}
	out := make([]StudioSegment, 0, len(parts))
	for i, text := range parts {
		kind := "body"
		if i == 0 {
			kind = "hook"
		} else if i == len(parts)-1 {
			kind = "cta"
		}
		out = append(out, StudioSegment{
			ID: NewID(), Kind: kind, Narration: text,
			Visual: text, Duration: each,
		})
	}
	return out
}

// shotsFromSegments 按分段生成镜头。字幕和画面说明都用这段口播，运镜是 ken-burns。时长小于等于 0 就写成 3 秒。
// segs 逐段读取 Narration、Visual 和 Duration，Duration 小于等于 0 时写成 3，最后传给 ensureShotIDs。
func shotsFromSegments(segs []StudioSegment) []StudioShot {
	out := make([]StudioShot, 0, len(segs))
	for _, s := range segs {
		dur := s.Duration
		if dur <= 0 {
			dur = 3
		}
		out = append(out, StudioShot{
			ID: NewID(), Caption: s.Narration, Visual: s.Visual, Prompt: s.Visual,
			Duration: dur, Motion: "ken-burns",
		})
	}
	return ensureShotIDs(out)
}

// ApplyStudioTiming 让镜头时长跟着口播走。还没有配音就拒绝。
// 字幕条数和镜头一样多时，每镜从这句开口持续到下一句开口，停顿也算进去；对不上时按镜头字幕的字数分。探到配音总长就整体对齐到这个长度，画面和口播一样长。
// 字幕是按字数估算的，就把字幕时间也按配音总长重排，烧进画面的字才跟得上声音。闸门还没到画面时改到画面。
// 读字幕和探时长在锁外做；写回时配音已经换过就拒绝。
// id 传给 GetStudioProject 和 mutateStudio。
func (e *Engine) ApplyStudioTiming(id string) (StudioProject, error) {
	snap, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if snap.NarrationHash == "" {
		return snap, fmt.Errorf("speak first")
	}
	total, _ := e.ProbeSeconds(snap.NarrationHash)
	cues := studioCues(snap, e)
	subtitle := ""
	if snap.SubtitleEstimated && total > 0 && len(cues) > 0 {
		if srt := cut.EncodeSRT(scaleCues(cues, total)); strings.TrimSpace(srt) != "" {
			if h, err := e.PutBytes([]byte(srt)); err == nil {
				subtitle = h
			}
		}
	}
	return e.mutateStudio(id, func(p *StudioProject) error {
		if p.NarrationHash != snap.NarrationHash {
			return fmt.Errorf("配音刚换过，再跟随一次口播")
		}
		if len(p.Storyboard.Shots) == 0 {
			p.Storyboard.Shots = shotsFromSegments(p.Segments)
		}
		durs := shotDurations(p.Storyboard.Shots, cueSpans(cues, total), total)
		if len(durs) == 0 {
			return fmt.Errorf("could not read narration timing")
		}
		for i := range p.Storyboard.Shots {
			p.Storyboard.Shots[i].Duration = durs[i]
		}
		if subtitle != "" {
			p.SubtitleHash = subtitle
			p.Storyboard.Subtitle = subtitle
		}
		if studioGateIndex(p.Gate) < studioGateIndex(GateVisuals) {
			p.Gate = GateVisuals
		}
		e.noteStudio(p, "timing", "shot lengths follow voice")
		return nil
	})
}

// studioCues 读出工程字幕的每一条。没有字幕或读不到就返回 nil。
func studioCues(p StudioProject, e *Engine) []cut.Cue {
	if p.SubtitleHash == "" || e == nil {
		return nil
	}
	raw, err := e.GetBytes(p.SubtitleHash)
	if err != nil {
		return nil
	}
	return cut.ParseSRT(string(raw))
}

// cueSpans 把字幕切成首尾相接的时间段。第一段从 0 开始，每段到下一条开口为止，最后一段到配音结束；总长不知道时到最后一条字幕结束。
// cues 按出现顺序排列。
// total 大于最后一条字幕的结束时间时作为终点。
func cueSpans(cues []cut.Cue, total float64) []float64 {
	if len(cues) == 0 {
		return nil
	}
	end := cues[len(cues)-1].End
	if total > end {
		end = total
	}
	out := make([]float64, len(cues))
	for i := range cues {
		start := 0.0
		if i > 0 {
			start = cues[i].Start
		}
		stop := end
		if i+1 < len(cues) {
			stop = cues[i+1].Start
		}
		if d := stop - start; d > 0 {
			out[i] = d
		}
	}
	return out
}

// scaleCues 把字幕时间整体缩放，让最后一条在 total 结束。
func scaleCues(cues []cut.Cue, total float64) []cut.Cue {
	if len(cues) == 0 || total <= 0 {
		return cues
	}
	last := cues[len(cues)-1].End
	if last <= 0 {
		return cues
	}
	k := total / last
	out := make([]cut.Cue, len(cues))
	for i, c := range cues {
		c.Start *= k
		c.End *= k
		out[i] = c
	}
	return out
}

// shotDurations 给每一镜一个时长。spans 和镜头一样多时用它，否则按镜头字幕的字数分，探不到总长时无从分配就返回 nil。探到总长时整体缩放到这个长度。每镜至少 1 秒。
// shots 提供镜头数和字幕。
// spans 是 cueSpans 的结果。
// total 是配音秒数，小于等于 0 表示没探到。
func shotDurations(shots []StudioShot, spans []float64, total float64) []float64 {
	n := len(shots)
	if n == 0 {
		return nil
	}
	var base []float64
	switch {
	case len(spans) == n:
		base = append([]float64(nil), spans...)
	case total > 0:
		base = make([]float64, n)
		for i, s := range shots {
			w := float64(utf8.RuneCountInString(strings.TrimSpace(s.Caption)))
			if w < 4 {
				w = 4
			}
			base[i] = w
		}
	default:
		return nil
	}
	if total > 0 {
		var sum float64
		for _, d := range base {
			sum += d
		}
		if sum > 0 {
			for i := range base {
				base[i] = base[i] * total / sum
			}
		}
	}
	for i := range base {
		if base[i] < 1 {
			base[i] = 1
		}
	}
	return base
}

// parseSRTDurations 读出 SRT 里每一条的时长。没有箭头的行会跳过。结束不大于开始的也跳过。
func parseSRTDurations(s string) []float64 {
	var out []float64
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "-->") {
			continue
		}
		parts := strings.Split(line, "-->")
		if len(parts) != 2 {
			continue
		}
		a := parseSRTStamp(strings.TrimSpace(parts[0]))
		b := parseSRTStamp(strings.TrimSpace(parts[1]))
		if b > a {
			out = append(out, b-a)
		}
	}
	return out
}

// parseSRTStamp 把 SRT 时间戳换成秒。逗号会先换成小数点。
func parseSRTStamp(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	var h, m int
	var sec float64
	fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	return float64(h*3600+m*60) + sec
}

// ProbeSeconds 用 ffprobe 读一份内容库媒体的时长。没有 ffprobe 或取不出文件就返回错误。
// hash 以扩展名 .bin 传给 materialize。
func (e *Engine) ProbeSeconds(hash string) (float64, error) {
	if e.FFProbe == "" {
		return 0, fmt.Errorf("ffprobe missing")
	}
	path, err := e.materialize(hash, ".bin")
	if err != nil {
		return 0, err
	}
	defer osRemove(path)
	return probeFileSeconds(e.FFProbe, path)
}

// StudioPreflight 是进某一闸之前，还缺什么、会用到什么。
type StudioPreflight struct {
	Gate string `json:"gate"`
	Ok   bool   `json:"ok"`
	// Spends 是这一闸可能会用到的能力。
	Spends []string `json:"spends,omitempty"`
	// Missing 是这一闸还没配上的能力。
	Missing []string `json:"missing,omitempty"`
	Message string   `json:"message,omitempty"`
	// Furthest 是这则工程当前最远能到的闸门。
	Furthest string `json:"furthest"`
}

// StudioPreflight 检查进这一闸还缺什么。闸门空着就用 FurthestGate。
// 配音看语音连接，画面看图像连接，合成看 ffmpeg，交付看成片在不在。缺了就把 Ok 设成假。
// id 传给 GetStudioProject。
// gate 去掉首尾空白，空串时改用 FurthestGate，并写入结果的 Gate。
func (e *Engine) StudioPreflight(id, gate string) (StudioPreflight, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioPreflight{}, err
	}
	gate = strings.TrimSpace(gate)
	if gate == "" {
		gate = p.FurthestGate()
	}
	out := StudioPreflight{Gate: gate, Furthest: p.FurthestGate(), Ok: true}
	var missingLabels []string
	// need 记下这一步要花的账号。Spends 是给人看的叫法；Missing 是能力名，界面靠它打开对应的设置页。
	need := func(cap, label string) {
		out.Spends = append(out.Spends, label)
		ok := e.Conn != nil
		if ok {
			_, err := e.ActiveProvider(cap, "")
			ok = err == nil
		}
		if !ok {
			out.Missing = append(out.Missing, cap)
			missingLabels = append(missingLabels, label)
			out.Ok = false
		}
	}
	switch gate {
	case GateAudio:
		if _, err := e.ActiveProvider("speech", ""); err != nil && edgeTTSAvailable() {
			out.Spends = append(out.Spends, "本机 edge-tts")
			out.Message = "没有启用语音账号，会用本机 edge-tts 配音"
		} else {
			need("speech", "语音账号")
		}
	case GateVisuals:
		if !p.VisualsReady() {
			need("image", "生图账号")
		}
	case GateCompose:
		out.Spends = append(out.Spends, "本机合成")
		e.refreshFFmpeg()
		if !e.FFmpegOK() {
			out.Message = "第一次合成会先下载 ffmpeg"
		}
	case GateDelivery:
		if p.ResultHash == "" {
			out.Missing = append(out.Missing, "video")
			out.Ok = false
			out.Message = "先合成一条短片。"
		}
	}
	if !out.Ok && len(missingLabels) > 0 {
		out.Message = "到 设置 → 生成 里启用" + strings.Join(missingLabels, "、") + "。"
	}
	return out, nil
}

// PackageStudio 打交付包。还没有成片就拒绝。文案空着就按平台写一版，封面空着就用海报。
// 成片和文案都在时标成可发布。封面仍缺会写在备注里。空的文件项不放进包里。成功后闸门改到交付，并追加一个版本。
// id 传给 mutateStudio。
func (e *Engine) PackageStudio(id string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		if p.ResultHash == "" {
			return fmt.Errorf("先合成一条短片")
		}
		if p.CopyText == "" {
			p.CopyText = defaultPublishCopy(*p)
		}
		if p.CopyHash == "" && e.CAS != nil {
			if h, err := e.PutBytes([]byte(p.CopyText)); err == nil {
				p.CopyHash = h
			}
		}
		if p.CoverHash == "" {
			p.CoverHash = p.PosterHash
		}
		var files []StudioDeliveryFile
		for _, f := range []StudioDeliveryFile{
			{Kind: "video", Hash: p.ResultHash, MIME: "video/mp4"},
			{Kind: "subtitle", Hash: p.SubtitleHash, MIME: "application/x-subrip"},
			{Kind: "cover", Hash: p.CoverHash, MIME: "image/jpeg"},
			{Kind: "copy", Hash: p.CopyHash, MIME: "text/markdown"},
		} {
			if f.Hash == "" {
				continue
			}
			f.Bytes = e.mediaBytes(f.Hash)
			files = append(files, f)
		}
		status := DeliveryDraft
		note := ""
		if p.CopyHash != "" {
			status = DeliveryPublishable
			if p.CoverHash == "" {
				note = "还缺封面：生成一张，或用合成时截的海报"
			}
		} else {
			note = "成片好了，还缺封面或文案"
		}
		p.Delivery = StudioDelivery{
			Status: status, VideoHash: p.ResultHash, SubtitleHash: p.SubtitleHash,
			CoverHash: p.CoverHash, CopyHash: p.CopyHash, CopyText: p.CopyText,
			Files: files, Note: note,
		}
		p.Gate = GateDelivery
		p.Versions = append(p.Versions, StudioSnapshot{
			N: len(p.Versions) + 1, ResultHash: p.ResultHash, PosterHash: p.PosterHash,
			NarrationHash: p.NarrationHash, SubtitleHash: p.SubtitleHash,
			CoverHash: p.CoverHash, CopyHash: p.CopyHash, Note: "delivery", CreatedAt: Now(),
		})
		e.noteStudio(p, "package", status)
		return nil
	})
}

// mediaBytes 返回内容库里这份对象的字节数。磁盘上有文件就只看大小，不把整段成片读进内存。读不到时返回 0。
func (e *Engine) mediaBytes(hash string) int64 {
	if path := e.casFile(hash); path != "" {
		if st, err := os.Stat(path); err == nil {
			return st.Size()
		}
	}
	if raw, err := e.GetBytes(hash); err == nil {
		return int64(len(raw))
	}
	return 0
}

// RestoreStudioVersion 把成片、海报、口播、字幕、封面和文案编号恢复成第 n 版，分镜里指着口播和字幕的编号一起换回。不改口播正文和镜头。找不到这一版就拒绝。
// id 传给 mutateStudio。
func (e *Engine) RestoreStudioVersion(id string, n int) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		var snap StudioSnapshot
		for _, v := range p.Versions {
			if v.N == n {
				snap = v
				break
			}
		}
		if snap.N == 0 {
			return fmt.Errorf("version %d not found", n)
		}
		p.ResultHash = snap.ResultHash
		p.PosterHash = snap.PosterHash
		p.NarrationHash = snap.NarrationHash
		p.SubtitleHash = snap.SubtitleHash
		p.Storyboard.Narration = snap.NarrationHash
		p.Storyboard.Subtitle = snap.SubtitleHash
		p.CoverHash = snap.CoverHash
		p.CopyHash = snap.CopyHash
		e.noteStudio(p, "restore", fmt.Sprintf("V%d", n))
		return nil
	})
}

// defaultPublishCopy 用标题、口播和平台标签拼一段发布文案。口播超过 180 个字就截断。平台空着按抖音。
func defaultPublishCopy(p StudioProject) string {
	platform := p.Platform
	if platform == "" {
		platform = "douyin"
	}
	title := strings.TrimSpace(p.Title)
	body := strings.TrimSpace(p.Script)
	if utf8.RuneCountInString(body) > 180 {
		runes := []rune(body)
		body = string(runes[:180]) + "…"
	}
	tags := "#短视频"
	switch platform {
	case "douyin":
		tags = "#抖音 #短视频"
	case "xhs":
		tags = "#小红书 #短视频"
	case "bilibili":
		tags = "#B站 #短片"
	case "channels":
		tags = "#视频号"
	}
	return "# " + title + "\n\n" + body + "\n\n" + tags + "\n"
}

// osRemove 删掉临时文件。路径空着或删除失败都忽略。
// path 非空时传给 os.Remove。
func osRemove(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
