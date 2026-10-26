// 偶像桌面的唯一入口 IdolCall。对话、旁白和主动联系走偶像引擎；工作指令由 startIdolWork 交回会话的 StartSendOpts。立绘字节的读写在 idol_media.go 和存储的 blob。

package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// idolBrain 用 App 的聊天客户端实现偶像引擎的 Chat 和 Paint。
type idolBrain struct{ a *App }

// Chat 按 kind 选模型和输出上限：judge 为 320，title 为 48，其余 700。温度 0.7。角色只认 system、assistant 和 user。
// ctx 传给 client.Chat。
// cacheKey 写入 ChatRequest 的 CacheKey。
// messages 按 Role 和 Content 收成 runtime.Message。
func (b idolBrain) Chat(ctx context.Context, kind, cacheKey string, messages []idol.Msg) (string, error) {
	client, model, err := b.a.idolClient(kind)
	if err != nil {
		return "", err
	}
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
	max := 700
	switch kind {
	case "judge":
		max = 320
	case "title":
		max = 48
	}
	out, err := client.Chat(ctx, runtime.ChatRequest{
		Model: model, Messages: msgs, MaxTokens: max, CacheKey: cacheKey, Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}
	return out.Content, nil
}

// Paint 把提示交给 idolPaint。
// ctx 传给 idolPaint。
// prompt 传给 idolPaint。
func (b idolBrain) Paint(ctx context.Context, prompt string) ([]byte, error) {
	return b.a.idolPaint(ctx, prompt)
}

// idolEngine 用当前偶像存储和 idolBrain 组一个引擎。
func (a *App) idolEngine() *idol.Engine {
	return &idol.Engine{Store: a.Idols, Brain: idolBrain{a}}
}

// beginIdolTurn 取消上一轮偶像调用，再开一个 120 秒超时的上下文。
func (a *App) beginIdolTurn() (context.Context, context.CancelFunc) {
	a.idolMu.Lock()
	defer a.idolMu.Unlock()
	if a.idolCancel != nil {
		a.idolCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	a.idolCancel = cancel
	return ctx, cancel
}

// cancelIdolTurn 取消当前偶像回合并清掉取消函数。
func (a *App) cancelIdolTurn() {
	a.idolMu.Lock()
	defer a.idolMu.Unlock()
	if a.idolCancel != nil {
		a.idolCancel()
		a.idolCancel = nil
	}
}

// IdolCall 是偶像桌面的唯一入口，payload 是 JSON 对象。它按 op 读写偶像、圈子、记忆、礼物和 blob；turn 走 idolTurn，speak 走 idolSpeak。未知 op 返回 unknown idol op。
func (a *App) IdolCall(op, payload string) (string, error) {
	if a == nil || a.Idols == nil {
		return "", fmt.Errorf("idol store missing")
	}
	op = strings.TrimSpace(op)
	now := time.Now()
	switch op {
	case "snapshot":
		return a.idolSnapshot()
	case "settings_get":
		return idolJSON(a.idolSettingsView())
	case "settings_save":
		return a.idolSaveSettings(payload)
	case "user_save":
		var card idol.UserCard
		_ = json.Unmarshal([]byte(payload), &card)
		err := a.Idols.Update(func(st *idol.State) error {
			st.User = card
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(card)
	case "save_idol":
		return a.idolSave(payload)
	case "delete_idol":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Idol
			for _, it := range st.Idols {
				if it.ID != p.ID {
					next = append(next, it)
				}
			}
			if len(next) == len(st.Idols) {
				return fmt.Errorf("idol not found")
			}
			st.Idols = next
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "pin":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		if err := a.Idols.Pin(p.ID); err != nil {
			return "", err
		}
		return a.idolSnapshot()
	case "save_circle":
		return a.idolSaveCircle(payload)
	case "delete_circle":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Circle
			for _, c := range st.Circles {
				if c.ID != p.ID {
					next = append(next, c)
				}
			}
			st.Circles = next
			for i := range st.Idols {
				if st.Idols[i].CircleID == p.ID {
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
		return a.idolTurn(payload, now)
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
			Filename string `json:"filename"`
			Audio    string `json:"audio"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		raw, err := decodeIdolB64(p.Audio)
		if err != nil {
			return "", err
		}
		text, err := a.idolTranscribe(p.Filename, raw)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"text": text})
	case "speak":
		var p struct {
			IdolID string `json:"idol_id"`
			Text   string `json:"text"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		spoken := p.Text
		if id, ok := a.Idols.View().Idol(p.IdolID); ok {
			spoken = idol.ApplyReadings(p.Text, id.Readings)
		}
		b, ctype, err := a.idolSpeak(spoken, a.idolVoice(p.IdolID))
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"audio": base64.StdEncoding.EncodeToString(b), "content_type": ctype})
	case "narrate":
		return a.idolNarrate(payload, now)
	case "memories":
		var p struct {
			IdolID string `json:"idol_id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		st := a.Idols.View()
		return idolJSON(map[string]any{"memories": st.MemoriesFor(p.IdolID)})
	case "delete_memory":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		err := a.Idols.Update(func(st *idol.State) error {
			for i := range st.Memories {
				if st.Memories[i].SupersededBy == p.ID {
					st.Memories[i].SupersededBy = ""
				}
			}
			var next []idol.Memory
			for _, m := range st.Memories {
				if m.ID != p.ID {
					next = append(next, m)
				}
			}
			st.Memories = next
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "gifts":
		st := a.Idols.View()
		_ = a.Idols.Update(func(live *idol.State) error {
			live.EnsureDaily(now)
			return nil
		})
		st = a.Idols.View()
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
		err := a.Idols.Update(func(st *idol.State) error {
			var e error
			rec, aff, e = st.Give(p.IdolID, p.GiftID, now)
			return e
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"record": rec, "affinity": aff})
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
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Lore
			for _, d := range st.Lore {
				if d.ID != p.ID {
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
		st := a.Idols.View()
		return idolJSON(st.Stats())
	case "reach":
		return a.idolReach(payload, now)
	case "new_conversation":
		var p struct {
			IdolID string `json:"idol_id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		var id string
		err := a.Idols.Update(func(st *idol.State) error {
			if _, ok := st.Idol(p.IdolID); !ok {
				return fmt.Errorf("idol not found")
			}
			id = st.ForceNew(p.IdolID, now).ID
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"id": id})
	case "history":
		var p struct {
			IdolID string `json:"idol_id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		return idolJSON(a.Idols.View().History(p.IdolID))
	case "delete_conversation":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		err := a.Idols.Update(func(st *idol.State) error {
			var next []idol.Conversation
			for _, c := range st.Conversations {
				if c.ID == p.ID && c.Current {
					return fmt.Errorf("open a new conversation before deleting the current one")
				}
				if c.ID != p.ID {
					next = append(next, c)
				}
			}
			st.Conversations = next
			return nil
		})
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"ok": true})
	case "call_end":
		var p struct {
			IdolID    string `json:"idol_id"`
			Seconds   int    `json:"seconds"`
			Connected bool   `json:"connected"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
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
		id, err := a.Idols.SaveBlob(raw)
		if err != nil {
			return "", err
		}
		return idolJSON(map[string]any{"id": id})
	case "blob":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		b, err := a.Idols.ReadBlob(p.ID)
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

// idolTurn 在暂停时返回 paused。回合超时 120 秒。结果要求工作时调用 startIdolWork，并把会话 id 和错误记到该条记录。
// payload 解出 idol_id、text、call 和 work。
// now 传给 Turn。
func (a *App) idolTurn(payload string, now time.Time) (string, error) {
	if a.IsPaused() {
		return "", fmt.Errorf("paused")
	}
	var p struct {
		IdolID string `json:"idol_id"`
		Text   string `json:"text"`
		Call   bool   `json:"call"`
		Work   bool   `json:"work"`
	}
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", err
	}
	ctx, cancel := a.beginIdolTurn()
	defer cancel()
	res, err := a.idolEngine().Turn(ctx, idol.TurnInput{IdolID: p.IdolID, Text: p.Text, Call: p.Call, Work: p.Work, Now: now})
	if err != nil {
		return "", err
	}
	if res.Work {
		sid, werr := a.startIdolWork(p.IdolID, res.WorkInstruction)
		res.WorkSession = sid
		msg := ""
		if werr != nil {
			msg = werr.Error()
			res.WorkError = msg
		}
		_ = a.Idols.AttachWork(res.RecordID, sid, msg)
	}
	return idolJSON(res)
}

// idolNarrate 在会话仍在跑时返回 Pending。否则取最后一条助手正文，用 90 秒超时做旁白。
// payload 解出 idol_id 和 session。
// now 传给 Narrate。
func (a *App) idolNarrate(payload string, now time.Time) (string, error) {
	var p struct {
		IdolID  string `json:"idol_id"`
		Session string `json:"session"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	if p.Session != "" && a.Running(p.Session) {
		return idolJSON(idol.TurnResult{Pending: true})
	}
	text := a.lastAssistant(p.Session)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	res, err := a.idolEngine().Narrate(ctx, p.IdolID, text, now)
	if err != nil {
		return "", err
	}
	return idolJSON(res)
}

// idolReach 在偶像 id 为空时用当前钉住的偶像。超时 60 秒。引擎返回 not due 时响应 due 为 false。
// payload 解出 idol_id。
// now 传给 Reach。
func (a *App) idolReach(payload string, now time.Time) (string, error) {
	var p struct {
		IdolID string `json:"idol_id"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	if p.IdolID == "" {
		if id, ok := a.Idols.View().Pinned(); ok {
			p.IdolID = id.ID
		}
	}
	if p.IdolID == "" {
		return idolJSON(map[string]any{"due": false})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := a.idolEngine().Reach(ctx, p.IdolID, now)
	if err != nil && err.Error() == "not due" {
		return idolJSON(map[string]any{"due": false})
	}
	if err != nil {
		return "", err
	}
	return idolJSON(res)
}

// startIdolWork 复用偶像的工作会话，没有则新建并改名为偶像名。档案为空时用 assistant。然后 SetSessionProfile，并用 StartSendOpts 把指令交回该会话。
// idolID 用来取偶像，并在新建会话后传给 SetWorkSession。
// instruction 先 TrimSpace，空串则返回错误，否则写进交给会话的正文。
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
	msg := fmt.Sprintf("The operator is speaking through their idol %s. Do this on the machine with your tools. When you finish, write a short plain-language result they can hear. Do not role-play the refusal of tools.\n\n%s", it.Name, instruction)
	if err := a.StartSendOpts(sid, msg, false, nil); err != nil {
		return sid, err
	}
	return sid, nil
}

// lastAssistant 从轨迹里取最后一条非 delta 的助手正文。轨迹库或会话 id 为空返回空串。
// sessionID 为空或轨迹库为空则返回空串，否则传给 Traces.Read。
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
// v 传给 json.Marshal。
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
