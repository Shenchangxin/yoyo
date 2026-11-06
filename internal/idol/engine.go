// Package idol 保存偶像、对话、好感、礼物和立绘缓存。Turn、Reach、Narrate、React 调 Brain.Chat；她写了照片描述时调 Brain.Paint。本包不调用 runtime.Run。
// 状态在目录里的 state.json，图片在 blobs/。立绘是 Spine 的 data.json、data.atlas、data.png，StandMiddleware 挂在 /idol-stand/，缺文件时向 CDN 拉一次并写入缓存。
// 预设人设、事实和头像是数据，见 preset.go 与嵌入的 preset/catalog.json。
//
// 一轮话照 eidolon 的语义走：先由判断模型评用户这句话，她带着这句话给她的感受、按回合开始时的档位回话；
// 这一轮会让她告别时（越界太多、最近几轮伤她太深、开始冷却），她在回复里先回应再告别，打字时这段对话随后结束，通话里她随后挂断。
// 回复按段流式吐出，通话可以一边生成一边念；被打断时只记下已经写完的段。照片、记忆和标题都在回复记下之后另外进行，不耽误她开口。
package idol

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"time"
)

// Msg 是一条发给 Brain 的消息。Role 取 system、user、assistant。
type Msg struct {
	Role    string
	Content string
}

// Brain 是模型口。Chat 的 kind 取 chat、judge、title、reach、memory。Paint 按提示画一张图，返回图片字节。
type Brain interface {
	Chat(ctx context.Context, kind, cacheKey string, messages []Msg) (string, error)
	Paint(ctx context.Context, prompt string) ([]byte, error)
}

// StreamBrain 是能流式回话的 Brain。onDelta 依次收到模型新写的文字，返回值是全文。
type StreamBrain interface {
	ChatStream(ctx context.Context, kind, cacheKey string, messages []Msg, onDelta func(string)) (string, error)
}

// ErrStopped 表示这一轮在写完任何一段之前就被叫停，什么也没记下。
var ErrStopped = errors.New("stopped")

const (
	// photoTimeout 是一张照片最多画多久。
	photoTimeout = 150 * time.Second
	// photoMaxSide 是存下的照片长边最多多少像素。
	photoMaxSide = 1024
	// titleTimeout 和 memoryTimeout 是后台起标题、写记忆最多等多久。
	titleTimeout  = 30 * time.Second
	memoryTimeout = 60 * time.Second
)

// TurnInput 是一轮对话的输入。Call 为真表示通话，CallID 是这通电话的 id，告别按这通电话里的轮次累计。
// Work 为真表示用户勾选了让她去办事。Now 为零时用当前时间。
// Stop 关闭时不再往下生成，已经写完的段照常记下。OnJudged 在评判之后收到她的情绪；OnSegment 依次收到写完的段；OnPhoto 在她开始拍照时调用。
type TurnInput struct {
	IdolID    string
	Text      string
	Call      bool
	CallID    string
	Work      bool
	Now       time.Time
	Stop      <-chan struct{}
	OnJudged  func(emotion string)
	OnSegment func(index int, seg Segment)
	OnPhoto   func()
}

// TurnResult 是一轮对话写进状态之后的结果。Ended 为真时打字的这段对话结束；HungUp 为真时她挂断了这通电话。
// Change 是好感怎么变了；PhotoPending 为真时照片还在拍，之后挂到 RecordID 那条记录上。Stopped 为真时她被打断，只记下了写完的段。
type TurnResult struct {
	RecordID        string    `json:"record_id,omitempty"`
	ConversationID  string    `json:"conversation_id,omitempty"`
	Segments        []Segment `json:"segments,omitempty"`
	Speak           string    `json:"speak,omitempty"`
	Caption         string    `json:"caption,omitempty"`
	Emotion         string    `json:"emotion,omitempty"`
	Affinity        Affinity  `json:"affinity"`
	Change          *Change   `json:"change,omitempty"`
	Ended           bool      `json:"ended,omitempty"`
	EndReason       string    `json:"end_reason,omitempty"`
	HungUp          bool      `json:"hung_up,omitempty"`
	Warning         bool      `json:"warning,omitempty"`
	Work            bool      `json:"work,omitempty"`
	WorkInstruction string    `json:"work_instruction,omitempty"`
	WorkSession     string    `json:"work_session,omitempty"`
	WorkError       string    `json:"work_error,omitempty"`
	PhotoID         string    `json:"photo_id,omitempty"`
	PhotoPending    bool      `json:"photo_pending,omitempty"`
	Achievements    []string  `json:"achievements,omitempty"`
	Pending         bool      `json:"pending,omitempty"`
	Stopped         bool      `json:"stopped,omitempty"`
}

// Engine 用 Store 落盘，用 Brain 生成回复和照片。
// Defer 用来在后台跑照片、记忆和标题；为空时就地执行（测试用）。CanPaint 为空或返回真时她可以发照片。
type Engine struct {
	Store    *Store
	Brain    Brain
	Defer    func(func())
	CanPaint func() bool
}

// later 把 fn 交给 Defer；没有 Defer 时就地执行。
func (e *Engine) later(fn func()) {
	if e.Defer != nil {
		e.Defer(fn)
		return
	}
	fn()
}

// chat 调模型。Brain 能流式且 onDelta 非空时边收边交给 onDelta；否则整段返回后一次交给 onDelta。
func (e *Engine) chat(ctx context.Context, kind, cacheKey string, msgs []Msg, onDelta func(string)) (string, error) {
	if sb, ok := e.Brain.(StreamBrain); ok && onDelta != nil {
		return sb.ChatStream(ctx, kind, cacheKey, msgs, onDelta)
	}
	raw, err := e.Brain.Chat(ctx, kind, cacheKey, msgs)
	if err == nil && onDelta != nil {
		onDelta(raw)
	}
	return raw, err
}

// workAsked 是用户勾选了让她办事、判断模型却没看出来时补的一句。
const workAsked = "用户勾选了「让你去办事」：这句话是要你在这台电脑上替对方办的事。先写 <work>…</work>，再口头答应。"

// workHinted 是判断模型看出这句像是要她办事时补的一句。
const workHinted = "用户这句话像是要你在这台电脑上替对方办事；如果是，就先写 <work>…</work>，再口头答应；如果只是聊天，就照常回话。"

// Turn 走一轮对话：评判、按档位和感受回话、流式吐段、记下这一轮并改好感，再在后台拍照、写记忆、起标题。
// 冷却中或这段对话已结束时拒绝。ctx 取消时什么也不记；in.Stop 关闭时记下已经写完的段。
func (e *Engine) Turn(ctx context.Context, in TurnInput) (TurnResult, error) {
	if e == nil || e.Store == nil || e.Brain == nil {
		return TurnResult{}, errf("idol engine is not ready")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return TurnResult{}, errf("empty")
	}
	if len([]rune(text)) > maxInput {
		return TurnResult{}, errf("too long")
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	st := e.Store.View()
	st.ensureDaily(now)
	idol, ok := st.idol(in.IdolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	aff := st.affinityOf(idol.ID, now)
	if aff.InCooldown {
		return TurnResult{}, errf("cooldown")
	}
	con, has := currentOf(&st, idol.ID)
	if has && con.Ended {
		return TurnResult{}, errf("ended")
	}
	workOn := st.Services.WorkEnabled && idol.WorkEnabled
	canPaint := e.CanPaint == nil || e.CanPaint()
	ver := e.judge(ctx, idol, st.User, con.Records, text, in.Call, canPaint, workOn)
	if err := ctx.Err(); err != nil {
		return TurnResult{}, err
	}
	if in.OnJudged != nil {
		in.OnJudged(ver.Emotion)
	}

	cools := ver.Judged != nil && st.cools(idol.ID, *ver.Judged, now)
	outBefore, hurts := tally(con, in.CallID)
	ending := ends(ver.Judged, cools, outBefore, hurts)
	warning := warns(ver.Judged, cools, outBefore, hurts)
	way := "parts"
	if in.Call {
		way = "hangs_up"
	}
	parting, warnWay := "", ""
	if ending != "" {
		parting = way
	} else if warning {
		warnWay = way
	}
	var about []string
	photoOK := false
	switch {
	case !canPaint:
		about = append(about, cannotSend)
	case ver.Photo && aff.Tier == TierCold:
		about = append(about, photoRefused)
	case ver.Photo:
		about = append(about, photoAsked)
		photoOK = true
	}
	if workOn {
		if in.Work {
			about = append(about, workAsked)
		} else if ver.Work {
			about = append(about, workHinted)
		}
	}
	talking, lastContact := talkTyping, lastContactOf(&st, idol.ID)
	if in.Call {
		talking, lastContact = talkCall, time.Time{}
	}
	msgs := buildPrompt(promptInput{
		st: &st, idol: idol, con: con, text: text, talking: talking,
		affinity: affinityPrompt(aff.Tier, ver.Judged, parting, warnWay),
		about:    about, closing: closingFor(&st, idol, con, text),
		now: now, lastContact: lastContact, work: workOn,
	})

	stream := NewReplyStream(idol.SpokenLang, idol.SubtitleLang)
	var job *photoJob
	photoTag := ""
	handle := func(evs []StreamEvent) {
		for _, ev := range evs {
			if ev.Photo != "" {
				if photoOK && job == nil {
					photoTag = ev.Photo
					job = e.beginPhoto(idol, ev.Photo)
					if in.OnPhoto != nil {
						in.OnPhoto()
					}
				}
				continue
			}
			if ev.Segment != nil && in.OnSegment != nil {
				in.OnSegment(ev.Index, *ev.Segment)
			}
		}
	}
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()
	var stopped atomic.Bool
	if in.Stop != nil {
		go func() {
			select {
			case <-in.Stop:
				stopped.Store(true)
				cancelStream()
			case <-streamCtx.Done():
			}
		}()
	}
	_, err := e.chat(streamCtx, "chat", "idol:"+idol.ID, msgs, func(d string) { handle(stream.Feed(d)) })
	halted := stopped.Load()
	if err != nil && !halted {
		job.drop()
		return TurnResult{}, err
	}
	if err := ctx.Err(); err != nil {
		job.drop()
		return TurnResult{}, err
	}
	parsed, rest := stream.Finish(!halted)
	handle(rest)
	if len(parsed.Segments) == 0 {
		job.drop()
		if halted {
			return TurnResult{}, ErrStopped
		}
		return TurnResult{}, errf("empty reply")
	}
	doWork := workOn && (in.Work || parsed.Work != "")
	instruction := strings.TrimSpace(parsed.Work)
	if instruction == "" {
		instruction = text
	}
	emotion := ver.Emotion
	if m := CueEmotion(parsed.Segments[0].Cue); m != "" {
		emotion = m
	}

	var result TurnResult
	var rec Record
	var history []Record
	err = e.Store.Update(func(live *State) error {
		live.ensureDaily(now)
		cur, ok := live.idol(in.IdolID)
		if !ok {
			return errf("idol not found")
		}
		c := live.ensureCurrent(cur.ID, now)
		if c.Ended {
			return errf("ended")
		}
		history = append([]Record(nil), c.Records...)
		before := live.affinityOf(cur.ID, now)
		after := before
		hurt, out := 0.0, false
		if ver.Judged != nil {
			before, after = live.applyJudged(cur.ID, *ver.Judged, now)
			hurt, out = hurtOf(ver.Judged), ver.Judged.OutOfLine
		}
		rec = Record{
			ID: newID(), Kind: "turn", At: now, User: text, Segments: parsed.Segments, Emotion: emotion,
			PhotoTag: photoTag, Call: in.CallID, Hurt: hurt, OutOfLine: out, Delta: roundDelta(after.Score - before.Score),
		}
		if job != nil {
			rec.PhotoState = "pending"
		}
		if doWork {
			rec.WorkText = instruction
		}
		c.Records = append(c.Records, rec)
		endsTyped := ending != "" && !in.Call
		hangsUp := ending != "" && in.Call
		if endsTyped {
			c.Ended, c.EndReason = true, ending
			c.Records = append(c.Records, Record{ID: newID(), Kind: "event", Event: "ended", At: now})
		}
		if hangsUp {
			c.Records = append(c.Records, Record{ID: newID(), Kind: "event", Event: "hung_up", At: now, Call: in.CallID})
		}
		live.replaceConversation(c)
		for i := range live.Reaches {
			if live.Reaches[i].IdolID == cur.ID {
				live.Reaches[i].Answered = true
			}
		}
		live.markTask(now, "talk")
		result = TurnResult{
			RecordID: rec.ID, ConversationID: c.ID, Segments: parsed.Segments,
			Speak: SpeakLine(parsed.Segments), Caption: CaptionLine(parsed.Segments), Emotion: emotion,
			Affinity: after, Change: changeOf(before, after), Ended: endsTyped, EndReason: ending, HungUp: hangsUp,
			Warning: warning, Work: doWork, PhotoPending: job != nil, Achievements: live.unlock(now), Stopped: halted,
		}
		if doWork {
			result.WorkInstruction = instruction
		}
		return nil
	})
	if err != nil {
		job.drop()
		return TurnResult{}, err
	}
	if job != nil {
		recID := rec.ID
		e.later(func() { e.finishPhoto(recID, job) })
	}
	e.later(func() { e.writeMemories(idol, history, text, parsed.Segments, now) })
	conID := result.ConversationID
	e.later(func() { e.maybeTitle(conID) })
	return result, nil
}

// closingFor 是一轮提示最后的记忆和材料。查询用她最近说的那句（回复或先开口）和用户这句，「它」「那件事」指什么靠上一句说清。
func closingFor(st *State, idol Idol, con Conversation, text string) []string {
	query := strings.TrimSpace(herLatest(con.Records) + "\n" + text)
	var out []string
	if m := formatMemories(st, idol, st.recallMemories(idol, query)); m != "" {
		out = append(out, m)
	}
	if lore := retrieve(idol, st.Lore, query); lore != "" {
		out = append(out, lore)
	}
	return out
}

// roundDelta 把好感变化取整给界面看。
func roundDelta(d float64) int { return int(math.Round(d)) }

// lastContactOf 是用户和她最近一次说上话的时间：她最近一轮被记下，或最近一通接通的电话结束，取较晚的。礼物不算。
func lastContactOf(st *State, idolID string) time.Time {
	var last time.Time
	for _, c := range st.Conversations {
		if c.IdolID != idolID {
			continue
		}
		for _, r := range c.Records {
			if r.Kind == "turn" && r.At.After(last) {
				last = r.At
			}
		}
	}
	for _, c := range st.Calls {
		if c.IdolID == idolID && c.Connected && c.At.After(last) {
			last = c.At
		}
	}
	return last
}

// photoJob 是一张正在拍的照片。done 关闭后 id 或 err 有值。
type photoJob struct {
	done   chan struct{}
	cancel context.CancelFunc
	id     string
	err    error
}

// drop 放弃这张照片：这一轮没有记下时调用。nil 时什么也不做。
func (j *photoJob) drop() {
	if j != nil {
		j.cancel()
	}
}

// beginPhoto 按她写的画面描述开始拍一张照片，存成 blob。拍照用自己的上下文，不随这一轮的请求结束而取消。
func (e *Engine) beginPhoto(idol Idol, tag string) *photoJob {
	ctx, cancel := context.WithTimeout(context.Background(), photoTimeout)
	j := &photoJob{done: make(chan struct{}), cancel: cancel}
	e.later(func() {
		defer close(j.done)
		defer cancel()
		b, err := e.Brain.Paint(ctx, photoPrompt(idol, tag))
		if err == nil {
			b, err = ShrinkImage(b, photoMaxSide)
		}
		if err == nil {
			j.id, err = e.Store.SaveBlob(b)
		}
		j.err = err
	})
	return j
}

// finishPhoto 等照片拍完，挂到那条记录上；没拍成就把记录标成 failed，她之后会知道那张没发出去。
func (e *Engine) finishPhoto(recordID string, j *photoJob) {
	<-j.done
	_ = e.Store.Update(func(st *State) error {
		rec := st.record(recordID)
		if rec == nil {
			return nil
		}
		if j.err == nil && j.id != "" {
			rec.PhotoID, rec.PhotoState = j.id, ""
		} else {
			rec.PhotoState = "failed"
		}
		return nil
	})
}

// photoPrompt 拼出全年龄照片的提示：她是谁、她的样子、她写的画面。
func photoPrompt(idol Idol, scene string) string {
	var b strings.Builder
	b.WriteString("A single all-ages photograph, natural and candid, no text, no watermark, no logo. ")
	b.WriteString("人物：" + strings.TrimSpace(idol.Name))
	if intro := strings.TrimSpace(idol.Intro); intro != "" {
		b.WriteString("，" + clip(intro, 160))
	}
	b.WriteString("。画面：" + clip(scene, 400))
	return b.String()
}

// writeMemories 在后台读这一轮对话，写下用户透露的事。history 是这一轮之前的记录。失败就算了，不影响这一轮。
func (e *Engine) writeMemories(idol Idol, history []Record, text string, reply []Segment, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), memoryTimeout)
	defer cancel()
	st := e.Store.View()
	shown := st.shownFor(idol, herLatest(history)+"\n"+text)
	raw, err := e.Brain.Chat(ctx, "memory", "", memoryPrompt(idol, st.User, history, text, reply, shown, now))
	if err != nil {
		return
	}
	written, ok := parseMemories(raw)
	if !ok || len(written) == 0 {
		return
	}
	_ = e.Store.Update(func(live *State) error {
		cur, ok := live.idol(idol.ID)
		if !ok {
			return errf("idol not found")
		}
		live.addMemories(cur, written, shown, now)
		return nil
	})
}

// maybeTitle 在对话的第 3 轮记下之后、还没有标题时，用 kind 为 title 的 Chat 起一个不超过 16 个字的标题。
// 只在第 3 轮试这一次：失败或给出空标题就不再试，免得之后每一轮都多花一次调用。她的提示里从不带标题。
func (e *Engine) maybeTitle(conID string) {
	st := e.Store.View()
	con, ok := st.conversation(conID)
	if !ok || strings.TrimSpace(con.Title) != "" || countTurns(con) != 3 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), titleTimeout)
	defer cancel()
	raw, err := e.Brain.Chat(ctx, "title", "", []Msg{
		{Role: "system", Content: "用一句中文概括这段对话在聊什么，不超过12个字。不写她的名字，不加引号和标点，只输出标题本身。"},
		{Role: "user", Content: captionSample(con)},
	})
	if err != nil {
		return
	}
	title := strings.Trim(strings.TrimSpace(raw), "「」\"'《》“”")
	if title == "" {
		return
	}
	if r := []rune(title); len(r) > 16 {
		title = string(r[:16])
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

// Reach 在 reachDue 为真时让她先开口，记成一条 reach，不是一轮话（ADR-0026）。提示的形状和打字的一轮一样，只是「此刻」写你们没在说话，
// 带上用户最近的事，最后是先开口的规则。
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
	aff := st.affinityOf(idol.ID, now)
	con, _ := currentOf(&st, idol.ID)
	user, reply := lastUserLine(con.Records)
	closing := []string{formatMemories(&st, idol, st.reachMemories(idol, reply+"\n"+user, now)), reachRules}
	msgs := buildPrompt(promptInput{
		st: &st, idol: idol, con: con, text: wantsToReachOut, talking: talkApart,
		affinity: affinityPrompt(aff.Tier, nil, "", ""), closing: closing,
		now: now, lastContact: lastContactOf(&st, idol.ID), work: st.Services.WorkEnabled && idol.WorkEnabled,
	})
	return e.say(ctx, idol, msgs, "reach", now, func(live *State) error {
		cur, ok := live.idol(idolID)
		if !ok || !reachDue(live, cur, now) {
			return errf("not due")
		}
		live.Reaches = append(live.Reaches, ReachMark{IdolID: cur.ID, At: now})
		return nil
	})
}

// React 在她收到礼物、还没回过话时，让她回应一下，记成一条 reply，不是一轮话。冷却中或这段对话已结束时不回应。
func (e *Engine) React(ctx context.Context, idolID string, now time.Time) (TurnResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	st := e.Store.View()
	idol, ok := st.idol(idolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	aff := st.affinityOf(idol.ID, now)
	if aff.InCooldown {
		return TurnResult{}, errf("cooldown")
	}
	con, has := currentOf(&st, idol.ID)
	if !has || con.Ended {
		return TurnResult{}, errf("ended")
	}
	if !giftUnanswered(con.Records) {
		return TurnResult{}, errf("nothing to answer")
	}
	msgs := buildPrompt(promptInput{
		st: &st, idol: idol, con: con, text: wantsToRespond, talking: talkTyping,
		affinity: affinityPrompt(aff.Tier, nil, "", ""), closing: closingFor(&st, idol, con, ""),
		now: now, lastContact: lastContactOf(&st, idol.ID), work: st.Services.WorkEnabled && idol.WorkEnabled,
	})
	return e.say(ctx, idol, msgs, "reply", now, func(live *State) error {
		c, ok := currentOf(live, idolID)
		if !ok || c.Ended || !giftUnanswered(c.Records) {
			return errf("nothing to answer")
		}
		return nil
	})
}

// giftUnanswered 判断最后一次回合、先开口或回应之后有没有她还没回应的礼物。
func giftUnanswered(records []Record) bool {
	open := false
	for _, r := range records {
		switch r.Kind {
		case "turn", "reach", "reply":
			open = false
		case "event":
			if r.GiftID != "" || r.Event == "gift" {
				open = true
			}
		}
	}
	return open
}

// Narrate 用她的口吻转述一件办完的事，记成一条 work_said。workText 空白时用一句兜底，过长时截断。
func (e *Engine) Narrate(ctx context.Context, idolID, workText string, now time.Time) (TurnResult, error) {
	if now.IsZero() {
		now = time.Now()
	}
	workText = strings.TrimSpace(workText)
	if workText == "" {
		workText = "事情办完了，但助手没有留下可以转述的话。"
	}
	workText = clip(workText, 1200)
	st := e.Store.View()
	idol, ok := st.idol(idolID)
	if !ok {
		return TurnResult{}, errf("idol not found")
	}
	aff := st.affinityOf(idol.ID, now)
	con, _ := currentOf(&st, idol.ID)
	msgs := buildPrompt(promptInput{
		st: &st, idol: idol, con: con, text: workDoneLead + workText, talking: talkTyping,
		affinity: affinityPrompt(aff.Tier, nil, "", ""), now: now,
		lastContact: lastContactOf(&st, idol.ID), work: st.Services.WorkEnabled && idol.WorkEnabled,
	})
	return e.say(ctx, idol, msgs, "work_said", now, nil)
}

// say 让她说一段不是回合的话（先开口、回应礼物、转述结果），记成 kind 那种记录。check 在写入前于锁内再核对一次，返回错误则不记。
func (e *Engine) say(ctx context.Context, idol Idol, msgs []Msg, kind string, now time.Time, check func(*State) error) (TurnResult, error) {
	model := "chat"
	if kind == "reach" || kind == "reply" {
		model = "reach"
	}
	raw, err := e.chat(ctx, model, "idol:"+idol.ID, msgs, nil)
	if err != nil {
		return TurnResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return TurnResult{}, err
	}
	parsed := ParseReply(raw, idol.SpokenLang, idol.SubtitleLang)
	if len(parsed.Segments) == 0 {
		return TurnResult{}, errf("empty reply")
	}
	emotion := CueEmotion(parsed.Segments[0].Cue)
	if emotion == "" {
		emotion = "calm"
	}
	var result TurnResult
	err = e.Store.Update(func(live *State) error {
		if check != nil {
			if err := check(live); err != nil {
				return err
			}
		}
		c := live.ensureCurrent(idol.ID, now)
		rec := Record{ID: newID(), Kind: kind, At: now, Segments: parsed.Segments, Emotion: emotion}
		live.appendRecord(c.ID, rec)
		result = TurnResult{
			RecordID: rec.ID, ConversationID: c.ID, Segments: parsed.Segments,
			Speak: SpeakLine(parsed.Segments), Caption: CaptionLine(parsed.Segments),
			Emotion: emotion, Affinity: live.affinityOf(idol.ID, now),
		}
		return nil
	})
	return result, err
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

// ApplyReadings 按对照表把书面写法换成读音。显示用的原文不改，只改传入的 text。一行可以用等号或空格分开两边，# 开头的行跳过。
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
