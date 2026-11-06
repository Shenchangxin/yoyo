// 偶像桌面的唯一入口 IdolCall。对话、旁白、先开口和回应礼物走偶像引擎；工作指令由 startIdolWork 交回会话的 StartSendOpts。
// 一轮话在后台跑（idol_live.go），界面用 turn_next 长轮询拿写完的段。语音、识别和生图的厂商协议在 idol_vendor.go。

package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// idolBrain 用 App 的聊天客户端实现偶像引擎的 Chat、ChatStream 和 Paint。
type idolBrain struct{ a *App }

// idolChatShape 按用途给出输出上限和温度：评判、记忆、标题要稳，用低温度；回话和先开口要像人，温度高一些。
func idolChatShape(kind string) (int, float64) {
	switch kind {
	case "judge":
		return 200, 0.2
	case "memory":
		return 600, 0.2
	case "title":
		return 48, 0.3
	case "reach":
		return 500, 0.9
	default:
		return 800, 0.8
	}
}

// request 组一次补全请求。角色只认 system、assistant 和 user。
func (b idolBrain) request(kind, cacheKey, model string, messages []idol.Msg) runtime.ChatRequest {
	msgs := make([]runtime.Message, 0, len(messages))
	for _, m := range messages {
		role := runtime.RoleUser
		switch m.Role {
		case "system":
			role = runtime.RoleSystem
		case "assistant":
			role = runtime.RoleAssistant
		}
		msgs = append(msgs, runtime.Message{Role: role, Content: m.Content})
	}
	max, temp := idolChatShape(kind)
	return runtime.ChatRequest{Model: model, Messages: msgs, MaxTokens: max, CacheKey: cacheKey, Temperature: temp}
}

// Chat 按 kind 选模型（评判、记忆、标题优先用评判模型）并整段返回。
func (b idolBrain) Chat(ctx context.Context, kind, cacheKey string, messages []idol.Msg) (string, error) {
	client, model, err := b.a.idolClient(kind)
	if err != nil {
		return "", err
	}
	out, err := client.Chat(ctx, b.request(kind, cacheKey, model, messages))
	if err != nil {
		return "", err
	}
	return out.Content, nil
}

// ChatStream 和 Chat 一样，只是把模型新写的文字依次交给 onDelta。
func (b idolBrain) ChatStream(ctx context.Context, kind, cacheKey string, messages []idol.Msg, onDelta func(string)) (string, error) {
	client, model, err := b.a.idolClient(kind)
	if err != nil {
		return "", err
	}
	out, err := client.ChatStream(ctx, b.request(kind, cacheKey, model, messages), func(c runtime.StreamChunk) error {
		if c.Type == runtime.ChunkTextDelta && c.Text != "" && onDelta != nil {
			onDelta(c.Text)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return out.Content, nil
}

// Paint 把提示交给 idolPaint。
func (b idolBrain) Paint(ctx context.Context, prompt string) ([]byte, error) {
	return b.a.idolPaint(ctx, prompt)
}

// idolEngine 用当前偶像存储和 idolBrain 组一个引擎。照片、记忆和标题在后台 goroutine 里做。
func (a *App) idolEngine() *idol.Engine {
	return &idol.Engine{
		Store:    a.Idols,
		Brain:    idolBrain{a},
		Defer:    func(fn func()) { go fn() },
		CanPaint: a.idolCanPaint,
	}
}

// cancelIdolTurn 取消所有还在跑的偶像回合，并清掉旧的取消函数。
func (a *App) cancelIdolTurn() {
	cancelIdolTurns("")
	a.idolMu.Lock()
	defer a.idolMu.Unlock()
	if a.idolCancel != nil {
		a.idolCancel()
		a.idolCancel = nil
	}
}

// idolID 是只带偶像或记录 id 的载荷。
type idolID struct {
	ID     string `json:"id"`
	IdolID string `json:"idol_id"`
}

// IdolCall 是偶像桌面的唯一入口，payload 是 JSON 对象。它按 op 读写偶像、圈子、记忆、礼物和 blob，开一轮话、合成、识别和画图。未知 op 返回 unknown idol op。
func (a *App) IdolCall(op, payload string) (string, error) {
	if a == nil || a.Idols == nil {
		return "", fmt.Errorf("idol store missing")
	}
	op = strings.TrimSpace(op)
	now := time.Now()
	var ref idolID
	_ = json.Unmarshal([]byte(payload), &ref)
	switch op {
	case "snapshot":
		return a.idolSnapshot()
	case "settings_get":
		return idolJSON(a.idolSettingsView())
	case "settings_save":
		return a.idolSaveSettings(payload)
	case "user_save":
		var card idol.UserCard
		if err := json.Unmarshal([]byte(payload), &card); err != nil {
			return "", err
		}
		if err := a.Idols.Update(func(st *idol.State) error {
			st.User = card
			return nil
		}); err != nil {
			return "", err
		}
		return idolJSON(card)
	case "save_idol":
		return a.idolSave(payload)
	case "delete_idol":
		cancelIdolTurns(ref.ID)
		if err := a.Idols.DeleteIdol(ref.ID); err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "pin":
		if err := a.Idols.Pin(ref.ID); err != nil {
			return "", err
		}
		return a.idolSnapshot()
	case "save_circle":
		return a.idolSaveCircle(payload)
	case "delete_circle":
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Circle
			for _, c := range st.Circles {
				if c.ID != ref.ID {
					next = append(next, c)
				}
			}
			st.Circles = next
			for i := range st.Idols {
				if st.Idols[i].CircleID == ref.ID {
					st.Idols[i].CircleID = ""
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "turn":
		return a.idolTurn(payload)
	case "turn_open":
		in, err := idolTurnInput(payload)
		if err != nil {
			return "", err
		}
		t, err := a.openIdolTurn(in)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"id": t.id})
	case "turn_next":
		var p struct {
			ID     string `json:"id"`
			After  int    `json:"after"`
			WaitMs int    `json:"wait_ms"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		t := idolLive(p.ID)
		if t == nil {
			return "", fmt.Errorf("turn not found")
		}
		return idolJSON(t.wait(p.After, idolWaitFor(p.WaitMs)))
	case "turn_stop":
		stopIdolTurn(ref.ID)
		return idolJSON(map[string]any{"ok": true})
	case "turn_cancel":
		if t := idolLive(ref.ID); t != nil {
			t.cancel()
		}
		return idolJSON(map[string]any{"ok": true})
	case "cancel":
		a.cancelIdolTurn()
		return idolJSON(map[string]any{"ok": true})
	case "truncate":
		var p struct {
			ID   string `json:"id"`
			Keep int    `json:"keep"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		if err := a.Idols.Truncate(p.ID, p.Keep); err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "transcribe":
		var p struct {
			IdolID   string `json:"idol_id"`
			Filename string `json:"filename"`
			Audio    string `json:"audio"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		raw, err := decodeIdolB64(p.Audio)
		if err != nil {
			return "", err
		}
		ctx, cancel := context.WithTimeout(context.Background(), idolHearTimeout)
		defer cancel()
		text, err := a.idolTranscribe(ctx, p.Filename, raw, a.Idols.View().HearingHint(p.IdolID))
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"text": text})
	case "speak":
		var p struct {
			IdolID string `json:"idol_id"`
			Text   string `json:"text"`
			Cue    string `json:"cue"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		ctx, cancel := context.WithTimeout(context.Background(), idolSpeakTimeout)
		defer cancel()
		b, ctype, err := a.idolSpeak(ctx, p.IdolID, p.Text, p.Cue)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"audio": base64.StdEncoding.EncodeToString(b), "content_type": ctype})
	case "narrate":
		return a.idolNarrate(payload)
	case "photo_wait":
		var p struct {
			RecordID string `json:"record_id"`
			WaitMs   int    `json:"wait_ms"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		photo, state := a.waitIdolPhoto(p.RecordID, idolWaitFor(p.WaitMs))
		return idolJSON(map[string]any{"photo_id": photo, "state": state})
	case "memories":
		return idolJSON(map[string]any{"memories": a.Idols.View().MemoriesFor(ref.IdolID)})
	case "delete_memory":
		if err := a.Idols.DeleteMemory(ref.ID); err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "gifts":
		_ = a.Idols.Update(func(live *idol.State) error {
			live.EnsureDaily(now)
			return nil
		})
		st := a.Idols.View()
		return idolJSON(map[string]any{
			"catalog": idol.Catalog, "inventory": st.Inventory,
			"draws": st.Draws + st.BonusDraws, "reset_cards": st.ResetCards,
		})
	case "draw":
		rng := rand.New(rand.NewPCG(uint64(now.UnixNano()), 1))
		var res idol.DrawResult
		err := a.Idols.Update(func(st *idol.State) error {
			st.EnsureDaily(now)
			var e error
			res, e = st.Draw(rng)
			return e
		})
		if err != nil {
			return "", err
		}
		return idolJSON(res)
	case "give":
		var p struct {
			IdolID string `json:"idol_id"`
			GiftID string `json:"gift_id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		var rec idol.Record
		var aff idol.Affinity
		var change *idol.Change
		err := a.Idols.Update(func(st *idol.State) error {
			var e error
			rec, aff, change, e = st.Give(p.IdolID, p.GiftID, now)
			return e
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"record": rec, "affinity": aff, "change": change})
	case "react":
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		res, err := a.idolEngine().React(ctx, ref.IdolID, now)
		if err != nil {
			switch err.Error() {
			case "nothing to answer", "cooldown", "ended":
				return idolJSON(map[string]any{"skipped": true, "reason": err.Error()})
			}
			return "", err
		}
		return idolJSON(res)
	case "reset":
		var n int
		err := a.Idols.Update(func(st *idol.State) error {
			var e error
			n, e = st.UseReset(now)
			return e
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"reset": n})
	case "save_lore":
		return a.idolSaveLore(payload)
	case "delete_lore":
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Lore
			for _, d := range st.Lore {
				if d.ID != ref.ID {
					next = append(next, d)
				}
			}
			st.Lore = next
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "stats":
		return idolJSON(a.Idols.View().Stats(now))
	case "reach":
		return a.idolReach(ref.IdolID, now)
	case "new_conversation":
		var id string
		err := a.Idols.Update(func(st *idol.State) error {
			if _, ok := st.Idol(ref.IdolID); !ok {
				return fmt.Errorf("idol not found")
			}
			id = st.ForceNew(ref.IdolID, now).ID
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"id": id})
	case "history":
		return idolJSON(a.Idols.View().History(ref.IdolID))
	case "delete_conversation":
		var photos []string
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Conversation
			for _, c := range st.Conversations {
				if c.ID == ref.ID && c.Current {
					return fmt.Errorf("open a new conversation before deleting the current one")
				}
				if c.ID == ref.ID {
					for _, r := range c.Records {
						if r.PhotoID != "" {
							photos = append(photos, r.PhotoID)
						}
					}
					continue
				}
				next = append(next, c)
			}
			st.Conversations = next
			return nil
		})
		if err != nil {
			return "", err
		}
		a.Idols.RemoveBlobs(photos)
		return idolJSON(map[string]any{"ok": true})
	case "call_start":
		noteIdolCall(ref.IdolID, true)
		return idolJSON(map[string]any{"ok": true})
	case "call_end":
		var p struct {
			IdolID    string `json:"idol_id"`
			Seconds   int    `json:"seconds"`
			Connected bool   `json:"connected"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		noteIdolCall(p.IdolID, false)
		err := a.Idols.Update(func(st *idol.State) error {
			st.Calls = append(st.Calls, idol.CallLog{IdolID: p.IdolID, At: now, Seconds: p.Seconds, Connected: p.Connected})
			if p.Connected {
				st.MarkTask(now, "call")
			}
			st.Unlock(now)
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "save_blob":
		var p struct {
			Data string `json:"data"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		raw, err := decodeIdolB64(p.Data)
		if err != nil {
			return "", err
		}
		if raw, err = idol.ShrinkImage(raw, 1024); err != nil {
			return "", err
		}
		id, err := a.Idols.SaveBlob(raw)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"id": id})
	case "blob":
		b, err := a.Idols.ReadBlob(ref.ID)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"data": base64.StdEncoding.EncodeToString(b)})
	case "ping":
		return a.idolPing(payload)
	case "profiles":
		if a.Profiles == nil {
			return idolJSON([]any{})
		}
		return idolJSON(a.Profiles.List())
	default:
		return "", fmt.Errorf("unknown idol op %s", op)
	}
}

// idolTurnInput 解出一轮的输入：偶像、这句话、是否通话和通话 id、是否勾选了办事。
func idolTurnInput(payload string) (idol.TurnInput, error) {
	var p struct {
		IdolID string `json:"idol_id"`
		Text   string `json:"text"`
		Call   bool   `json:"call"`
		CallID string `json:"call_id"`
		Work   bool   `json:"work"`
	}
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return idol.TurnInput{}, err
	}
	return idol.TurnInput{IdolID: p.IdolID, Text: p.Text, Call: p.Call, CallID: strings.TrimSpace(p.CallID), Work: p.Work}, nil
}

// idolTurn 开一轮并等它跑完再返回结果（打字用）。暂停时返回 paused。
func (a *App) idolTurn(payload string) (string, error) {
	in, err := idolTurnInput(payload)
	if err != nil {
		return "", err
	}
	t, err := a.openIdolTurn(in)
	if err != nil {
		return "", err
	}
	for {
		v := t.wait(1<<30, idolWaitCap)
		if !v.Done {
			continue
		}
		if v.Error != "" {
			return "", errors.New(v.Error)
		}
		return idolJSON(v.Result)
	}
}

// idolNarrate 取她交给助手那件事的转述：发起那一轮的记录 id 或会话 id 有人在等时，最多等 wait_ms；
// 没人在等（比如应用重启过）时现在开始等。还没办完返回 pending。
func (a *App) idolNarrate(payload string) (string, error) {
	var p struct {
		IdolID   string `json:"idol_id"`
		Session  string `json:"session"`
		RecordID string `json:"record_id"`
		WaitMs   int    `json:"wait_ms"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	w := idolWork(p.RecordID, p.Session)
	if w == nil {
		if p.Session == "" || p.IdolID == "" {
			return "", fmt.Errorf("nothing to narrate")
		}
		w = a.watchIdolWork(p.IdolID, p.Session, p.RecordID)
	}
	timer := time.NewTimer(idolWaitFor(p.WaitMs))
	defer timer.Stop()
	select {
	case <-w.done:
	case <-timer.C:
		return idolJSON(idol.TurnResult{Pending: true})
	}
	if w.result == nil {
		return "", errors.New(firstNonEmpty(w.errText, "narration failed"))
	}
	return idolJSON(w.result)
}

// idolReach 在偶像 id 为空时用当前钉住的偶像，超时 60 秒。还没到她开口的时候、暂停时、用户正在和她打字或通话时，返回 due 为 false。
func (a *App) idolReach(idolID string, now time.Time) (string, error) {
	if idolID == "" {
		if it, ok := a.Idols.View().Pinned(); ok {
			idolID = it.ID
		}
	}
	if idolID == "" || a.IsPaused() || idolTalking(idolID, now) {
		return idolJSON(map[string]any{"due": false})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := a.idolEngine().Reach(ctx, idolID, now)
	if err != nil && err.Error() == "not due" {
		return idolJSON(map[string]any{"due": false})
	}
	if err != nil {
		return "", err
	}
	return idolJSON(map[string]any{"due": true, "idol_id": idolID, "record_id": res.RecordID, "speak": res.Speak, "caption": res.Caption, "emotion": res.Emotion})
}

// startIdolWork 复用偶像的工作会话，没有则新建并改名为偶像名。档案为空时用 assistant。然后 SetSessionProfile，并用 StartSendOpts 把指令交给该会话。
// 会话正忙时这件事排在后面，也算交出去了。
func (a *App) startIdolWork(idolID, instruction string) (string, error) {
	if a.IsPaused() {
		return "", fmt.Errorf("paused")
	}
	st := a.Idols.View()
	it, ok := st.Idol(idolID)
	if !ok {
		return "", fmt.Errorf("idol not found")
	}
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return "", fmt.Errorf("empty work")
	}
	profileID := it.WorkProfileID
	if profileID == "" {
		profileID = st.Services.WorkProfileID
	}
	if profileID == "" {
		profileID = "assistant"
	}
	sid := it.WorkSessionID
	if sid != "" {
		if _, err := a.GetSession(sid); err != nil {
			sid = ""
		}
	}
	if sid == "" {
		meta, err := a.NewSession("")
		if err != nil {
			return "", err
		}
		sid = meta.ID
		_ = a.RenameSession(sid, it.Name)
		_ = a.Idols.SetWorkSession(idolID, sid)
	}
	_, _ = a.SetSessionProfile(sid, profileID)
	msg := fmt.Sprintf("The operator asked their idol %s to get this done on the machine. Do it with your tools. When you finish, write a short plain-language result (what was done, what changed, anything they must decide) that she can retell; if you could not do it, say why. Do not role-play.\n\n%s", it.Name, instruction)
	if err := a.StartSendOpts(sid, msg, false, nil); err != nil && !errors.Is(err, ErrQueued) {
		return sid, err
	}
	return sid, nil
}

// lastAssistant 从轨迹里取最后一条非 delta 的助手正文。轨迹库或会话 id 为空返回空串。
func (a *App) lastAssistant(sessionID string) string {
	if a.Traces == nil || sessionID == "" {
		return ""
	}
	evs, err := a.Traces.Read(sessionID)
	if err != nil {
		return ""
	}
	text := ""
	for _, ev := range evs {
		if ev.Type != trace.TypeAssistant {
			continue
		}
		if delta, _ := ev.Payload["delta"].(bool); delta {
			continue
		}
		if t, _ := ev.Payload["text"].(string); strings.TrimSpace(t) != "" {
			text = t
		}
	}
	return text
}

// idolJSON 把值编码成 JSON 字符串。
func idolJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeIdolB64 去掉 data URL 前缀后按标准 base64 解码。
func decodeIdolB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}
