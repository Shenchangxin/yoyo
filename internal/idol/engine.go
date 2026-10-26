// Package idol 保存偶像、对话、好感、礼物和立绘缓存。Turn、Reach、Narrate 调 Brain.Chat；要照片时调 Brain.Paint。本包不调用 runtime.Run。
// 状态在目录里的 state.json，图片在 blobs/。立绘是 Spine 的 data.json、data.atlas、data.png，StandMiddleware 挂在 /idol-stand/，缺文件时向 CDN 拉一次并写入缓存。
// 预设人设、事实和头像是数据，见 preset.go 与嵌入的 preset/catalog.json。
package idol

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// Msg 是一条发给 Brain 的消息。Role 取 system、user、assistant。
type Msg struct {
	Role    string
	Content string
}

// Brain 是模型口。Chat 的 kind 取 chat、judge、title、reach。Paint 按提示画一张图，返回图片字节。
type Brain interface {
	Chat(ctx context.Context, kind, cacheKey string, messages []Msg) (string, error)
	Paint(ctx context.Context, prompt string) ([]byte, error)
}

// TurnInput 是一轮对话的输入。Call 为真表示通话，Work 为真表示用户要求去操作这台电脑。Now 为零时 Turn 用当前时间。
type TurnInput struct {
	IdolID string
	Text   string
	Call   bool
	Work   bool
	Now    time.Time
}

// TurnResult 是一轮对话写进状态之后的结果。Ended 为真时这段对话结束；HungUp 仅在通话中结束时为真。
type TurnResult struct {
	RecordID        string    `json:"record_id,omitempty"`
	ConversationID  string    `json:"conversation_id,omitempty"`
	Segments        []Segment `json:"segments,omitempty"`
	Speak           string    `json:"speak,omitempty"`
	Caption         string    `json:"caption,omitempty"`
	Emotion         string    `json:"emotion,omitempty"`
	Affinity        Affinity  `json:"affinity"`
	Ended           bool      `json:"ended,omitempty"`
	HungUp          bool      `json:"hung_up,omitempty"`
	Warning         bool      `json:"warning,omitempty"`
	Work            bool      `json:"work,omitempty"`
	WorkInstruction string    `json:"work_instruction,omitempty"`
	WorkSession     string    `json:"work_session,omitempty"`
	WorkError       string    `json:"work_error,omitempty"`
	PhotoID         string    `json:"photo_id,omitempty"`
	Achievements    []string  `json:"achievements,omitempty"`
	Pending         bool      `json:"pending,omitempty"`
}

// Engine 用 Store 落盘，用 Brain 生成回复和照片。
type Engine struct {
	Store *Store
	Brain Brain
}

// Turn 走一轮对话。先裁决好感、越界、照片和工作，再调 Brain.Chat。越界两次、分数降到 12 及以下，或连续三次负向，这段对话结束。
// 好感变化被夹在 -25 到 4。工作通道打开且裁决或回复里有工作时，Work 为真。冷淡档不发照片。
func (e *Engine) Turn(ctx context.Context, in TurnInput) (TurnResult, error) {
	if e == nil || e.Store == nil || e.Brain == nil {
		return TurnResult{}, errf("idol engine is not ready")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return TurnResult{}, errf("empty")
	}
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	st := e.Store.View()
	st.ensureDaily(in.Now)
	idol, ok := st.idol(in.IdolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	aff := st.affinityOf(idol.ID)
	if aff.Cooling(in.Now) {
		return TurnResult{}, errf("cooldown")
	}
	con, has := currentOf(&st, idol.ID)
	if has && con.Ended {
		return TurnResult{}, errf("ended")
	}
	ver := e.judge(ctx, idol, text, aff.Tier)
	workOn := st.Services.WorkEnabled || idol.WorkEnabled
	doWork := workOn && (in.Work || ver.Work)
	if ver.OutOfLine {
		ver.Affinity = -25
	}
	if ver.Affinity > 4 {
		ver.Affinity = 4
	}
	if ver.Affinity < -25 {
		ver.Affinity = -25
	}
	emo := cleanEmotion(ver.Emotion)
	msgs := buildPrompt(&st, idol, con, text, in.Call, aff, in.Now)
	if doWork {
		msgs = append(msgs[:len(msgs)-1], Msg{Role: "system", Content: "用户要你去操作这台电脑。你只口头答应一两句，具体的事交给工作通道。不要描述工具。"}, msgs[len(msgs)-1])
	}
	raw, err := e.Brain.Chat(ctx, "chat", "idol:"+idol.ID, msgs)
	if err != nil {
		return TurnResult{}, err
	}
	if ctx.Err() != nil {
		return TurnResult{}, ctx.Err()
	}
	parsed := ParseReply(raw, idol.SpokenLang, idol.SubtitleLang)
	if len(parsed.Segments) == 0 {
		return TurnResult{}, errf("empty reply")
	}
	if strings.TrimSpace(parsed.Work) != "" {
		doWork = workOn
	}
	instruction := text
	if strings.TrimSpace(parsed.Work) != "" {
		instruction = parsed.Work
	}
	var photoID string
	if ver.Photo && aff.Tier != TierCold && emo != "" {
		scene := parsed.Photo
		if scene == "" {
			scene = text
		}
		if b, perr := e.Brain.Paint(ctx, photoPrompt(idol, scene)); perr == nil && len(b) > 0 {
			if id, serr := e.Store.SaveBlob(b); serr == nil {
				photoID = id
			}
		}
	}
	var result TurnResult
	err = e.Store.Update(func(live *State) error {
		live.ensureDaily(in.Now)
		idol, ok = live.idol(in.IdolID)
		if !ok {
			return errf("idol not found")
		}
		con = live.ensureCurrent(idol.ID, in.Now)
		if con.Ended {
			return errf("ended")
		}
		for i := range live.Reaches {
			if live.Reaches[i].IdolID == idol.ID {
				live.Reaches[i].Answered = true
			}
		}
		if ver.OutOfLine {
			con.OutOfLine++
		}
		if ver.Affinity < 0 {
			con.NegStreak++
		} else if ver.Affinity > 0 {
			con.NegStreak = 0
		}
		aff = live.ApplyDelta(idol.ID, ver.Affinity, in.Now, 0)
		rec := Record{
			ID: newID(), Kind: "turn", At: in.Now, User: text,
			Segments: parsed.Segments, Emotion: emo, PhotoID: photoID, Delta: ver.Affinity,
		}
		endReason := ""
		if con.OutOfLine >= 2 {
			endReason = "out of line"
		} else if aff.Score <= 12 {
			endReason = "cold"
		} else if con.NegStreak >= 3 {
			endReason = "hurt"
		}
		warning := (con.OutOfLine == 1 && ver.OutOfLine) || con.NegStreak == 2
		if endReason != "" {
			con.Ended = true
			con.EndReason = endReason
			con.Records = append(con.Records, rec, Record{
				ID: newID(), Kind: "event", At: in.Now,
				Segments: []Segment{{Spoken: "这段对话先到这里", Subtitle: "这段对话先到这里"}},
			})
		} else {
			con.Records = append(con.Records, rec)
		}
		if strings.TrimSpace(con.Title) == "" && countTurns(con) >= 3 {
			con.Title = ""
		}
		live.replaceConversation(con)
		if strings.TrimSpace(ver.Memory) != "" {
			live.remember(idol, ver.Memory, in.Now)
		}
		live.markTask(in.Now, idolTZ(live.User), "talk")
		achs := live.unlock(in.Now)
		result = TurnResult{
			RecordID: rec.ID, ConversationID: con.ID, Segments: parsed.Segments,
			Speak: SpeakLine(parsed.Segments), Caption: CaptionLine(parsed.Segments),
			Emotion: emo, Affinity: aff, Ended: endReason != "", HungUp: endReason != "" && in.Call,
			Warning: warning, Work: doWork, WorkInstruction: instruction, PhotoID: photoID,
			Achievements: achs,
		}
		if !doWork {
			result.WorkInstruction = ""
		}
		return nil
	})
	if err != nil {
		return TurnResult{}, err
	}
	if result.ConversationID != "" {
		e.maybeTitle(ctx, result.ConversationID)
	}
	return result, nil
}

// maybeTitle 在当前对话满 3 个 turn 且还没有标题时，用 kind 为 title 的 Chat 起一个不超过 16 个字的标题。
func (e *Engine) maybeTitle(ctx context.Context, conID string) {
	st := e.Store.View()
	con, ok := st.conversation(conID)
	if !ok || strings.TrimSpace(con.Title) != "" || countTurns(con) < 3 {
		return
	}
	raw, err := e.Brain.Chat(ctx, "title", "", []Msg{
		{Role: "system", Content: "用不超过12个字的中文给这段对话起标题，只输出标题。"},
		{Role: "user", Content: captionSample(con)},
	})
	if err != nil {
		return
	}
	title := strings.TrimSpace(raw)
	if title == "" {
		return
	}
	if len([]rune(title)) > 16 {
		title = string([]rune(title)[:16])
	}
	_ = e.Store.Update(func(live *State) error {
		c, ok := live.conversation(conID)
		if !ok || c.Title != "" {
			return nil
		}
		c.Title = title
		live.replaceConversation(c)
		return nil
	})
}

// Reach 在 reachDue 为真时让偶像先说一句，kind 是 reach，并记下一次 ReachMark。情绪固定为 calm。
func (e *Engine) Reach(ctx context.Context, idolID string, now time.Time) (TurnResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	st := e.Store.View()
	idol, ok := st.idol(idolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	if !reachDue(&st, idol, now) {
		return TurnResult{}, errf("not due")
	}
	aff := st.affinityOf(idol.ID)
	con, _ := currentOf(&st, idol.ID)
	msgs := buildPrompt(&st, idol, con, "（用户还没说话。你先轻轻开口，一句就好，不要提问连环。）", false, aff, now)
	raw, err := e.Brain.Chat(ctx, "reach", "idol:"+idol.ID, msgs)
	if err != nil {
		return TurnResult{}, err
	}
	parsed := ParseReply(raw, idol.SpokenLang, idol.SubtitleLang)
	if len(parsed.Segments) == 0 {
		return TurnResult{}, errf("empty reply")
	}
	var result TurnResult
	err = e.Store.Update(func(live *State) error {
		idol, ok = live.idol(idolID)
		if !ok {
			return errf("idol not found")
		}
		if !reachDue(live, idol, now) {
			return errf("not due")
		}
		con = live.ensureCurrent(idol.ID, now)
		rec := Record{ID: newID(), Kind: "reach", At: now, Segments: parsed.Segments, Emotion: "calm"}
		live.appendRecord(con.ID, rec)
		live.Reaches = append(live.Reaches, ReachMark{IdolID: idol.ID, At: now})
		result = TurnResult{
			RecordID: rec.ID, ConversationID: con.ID, Segments: parsed.Segments,
			Speak: SpeakLine(parsed.Segments), Caption: CaptionLine(parsed.Segments),
			Emotion: "calm", Affinity: live.affinityOf(idol.ID),
		}
		return nil
	})
	return result, err
}

// Narrate 用偶像的口吻转述一件已办完的事，写入 kind 为 work_said 的记录。workText 空白时用一句兜底，长于 500 个字符时截断。
func (e *Engine) Narrate(ctx context.Context, idolID, workText string, now time.Time) (TurnResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	workText = strings.TrimSpace(workText)
	if workText == "" {
		workText = "事情办完了，但没有留下可以转述的话。"
	}
	if len([]rune(workText)) > 500 {
		workText = string([]rune(workText)[:500])
	}
	st := e.Store.View()
	idol, ok := st.idol(idolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	aff := st.affinityOf(idol.ID)
	con, _ := currentOf(&st, idol.ID)
	msgs := buildPrompt(&st, idol, con, "用你的口吻，一两句告诉用户这件事的结果，不要复述工具名：\n"+workText, false, aff, now)
	raw, err := e.Brain.Chat(ctx, "chat", "idol:"+idol.ID, msgs)
	if err != nil {
		return TurnResult{}, err
	}
	parsed := ParseReply(raw, idol.SpokenLang, idol.SubtitleLang)
	if len(parsed.Segments) == 0 {
		return TurnResult{}, errf("empty reply")
	}
	var result TurnResult
	err = e.Store.Update(func(live *State) error {
		con = live.ensureCurrent(idolID, now)
		rec := Record{ID: newID(), Kind: "work_said", At: now, Segments: parsed.Segments}
		live.appendRecord(con.ID, rec)
		result = TurnResult{
			RecordID: rec.ID, ConversationID: con.ID, Segments: parsed.Segments,
			Speak: SpeakLine(parsed.Segments), Caption: CaptionLine(parsed.Segments),
			Affinity: live.affinityOf(idolID),
		}
		return nil
	})
	return result, err
}

// verdict 是裁决模型返回的 JSON。Affinity 随后会被夹到 -25 到 4；越界时 Turn 改成 -25。
type verdict struct {
	Affinity  int    `json:"affinity"`
	OutOfLine bool   `json:"out_of_line"`
	Photo     bool   `json:"photo"`
	Work      bool   `json:"work"`
	Emotion   string `json:"emotion"`
	Memory    string `json:"memory"`
}

// judge 用 kind 为 judge 的 Chat 读用户这句话。解析失败时情绪是 calm，其余字段为零值。记忆写成 NONE 或 null 时清空。
func (e *Engine) judge(ctx context.Context, idol Idol, text, tier string) verdict {
	raw, err := e.Brain.Chat(ctx, "judge", "", []Msg{
		{Role: "system", Content: "你只输出 JSON。"},
		{Role: "user", Content: judgePrompt(text, tier)},
	})
	if err != nil {
		return verdict{Emotion: "calm"}
	}
	var v verdict
	blob := extractJSON(raw)
	if json.Unmarshal([]byte(blob), &v) != nil {
		return verdict{Emotion: "calm"}
	}
	v.Emotion = cleanEmotion(v.Emotion)
	v.Memory = strings.TrimSpace(v.Memory)
	if strings.EqualFold(v.Memory, "NONE") || strings.EqualFold(v.Memory, "null") {
		v.Memory = ""
	}
	return v
}

// extractJSON 取出第一个 { 到最后一个 } 之间的文本。没有成对括号时原样返回。
func extractJSON(s string) string {
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i >= 0 && j > i {
		return s[i : j+1]
	}
	return s
}

// cleanEmotion 只保留 happy、shy、surprised、sad、angry、thinking、calm。其余变成 calm。
func cleanEmotion(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "happy", "shy", "surprised", "sad", "angry", "thinking", "calm":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "calm"
	}
}

// photoPrompt 拼出全年龄照片的英文提示，带上偶像名字、简介和场景。
func photoPrompt(idol Idol, scene string) string {
	return "one all-ages photograph, no text, no watermark, of " + idol.Name + ". " +
		strings.TrimSpace(idol.Intro) + " scene: " + scene
}

// countTurns 数一段对话里 kind 为 turn 的记录。
func countTurns(c Conversation) int {
	n := 0
	for _, r := range c.Records {
		if r.Kind == "turn" {
			n++
		}
	}
	return n
}

// captionSample 取最后 6 条记录的用户原文和字幕，给起标题用。
func captionSample(c Conversation) string {
	var b strings.Builder
	for _, r := range tailRecords(c.Records, 6) {
		if r.User != "" {
			b.WriteString(r.User)
			b.WriteString("\n")
		}
		if line := CaptionLine(r.Segments); line != "" {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ApplyReadings 按对照表把书面名字换成读音。显示用的原文不改，只改传入的 text。一行可以用等号或空格分开两边，# 开头的行跳过。
func ApplyReadings(text, table string) string {
	for _, ln := range strings.Split(table, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		parts := strings.SplitN(ln, "=", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(ln, " ", 2)
		}
		if len(parts) != 2 {
			continue
		}
		from, to := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if from != "" && to != "" {
			text = strings.ReplaceAll(text, from, to)
		}
	}
	return text
}
