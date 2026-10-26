// 暂停、页面、档案、日程重试、连接器提及和语音。暂停会停掉正在跑的回合。视频频道和子会话不套用档案工具过滤。

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/profile"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// IsPaused 在 App 非空且 Config.Paused 为真时返回真。
func (a *App) IsPaused() bool {
	return a != nil && a.Config.Paused
}

// SetPaused 写入配置。变为暂停时 StopTurn 每一个正在跑的会话。两种方向都记 journal，并向 Hub 发 pause 事件，然后 SaveConfig。
func (a *App) SetPaused(paused bool) error {
	a.Config.Paused = paused
	if paused {
		for _, id := range a.RunningIDs() {
			a.StopTurn(id)
		}
		if a.Journal != nil {
			_, _ = a.Journal.Append("pause", map[string]any{"paused": true})
		}
	} else if a.Journal != nil {
		_, _ = a.Journal.Append("pause", map[string]any{"paused": false})
	}
	if a.Hub != nil {
		a.Hub.Publish(trace.Event{
			TS:      time.Now().UTC(),
			Type:    trace.TypePause,
			Source:  "app",
			Payload: map[string]any{"paused": paused},
		})
	}
	return a.SaveConfig()
}

// PagesList 在查询非空时搜索，否则按空间列出。页面库为空返回 nil。
func (a *App) PagesList(q, spaceID string) []pages.Meta {
	if a.Pages == nil {
		return nil
	}
	if strings.TrimSpace(q) != "" {
		return a.Pages.Search(q, spaceID)
	}
	return a.Pages.List(spaceID)
}

// PagesGet 读取一页。页面库为空返回 pages store unavailable。
func (a *App) PagesGet(spaceID, id string) (pages.Page, error) {
	if a.Pages == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	return a.Pages.Get(spaceID, id)
}

// PagesSave 在页面 id 为空时创建，否则按 expected 修订号保存。成功时记 page.save 或 page.edit。
func (a *App) PagesSave(p pages.Page, expected int) (pages.Page, error) {
	if a.Pages == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	if strings.TrimSpace(p.ID) == "" {
		out, err := a.Pages.Create(p)
		if err == nil && a.Journal != nil {
			_, _ = a.Journal.Append("page.save", map[string]any{"id": out.ID, "title": out.Title, "manual": true})
		}
		return out, err
	}
	out, err := a.Pages.Save(p, expected)
	if err == nil && a.Journal != nil {
		_, _ = a.Journal.Append("page.edit", map[string]any{"id": out.ID, "revision": out.Revision, "manual": true})
	}
	return out, err
}

// PagesDelete 删除页面，并清掉仍指向它的会话 PageID。id 空白返回 page not found。
func (a *App) PagesDelete(spaceID, id string) error {
	if a.Pages == nil {
		return fmt.Errorf("pages store unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("page not found")
	}
	if err := a.Pages.Delete(spaceID, id); err != nil {
		return err
	}
	list, _ := a.ListSessions()
	for _, m := range list {
		if m.PageID != id {
			continue
		}
		m.PageID = ""
		_ = a.writeSession(m)
	}
	if a.Journal != nil {
		_, _ = a.Journal.Append("page.delete", map[string]any{"id": id})
	}
	return nil
}

// PagesPropose 提交一页审阅，并向收件箱推一条提案。
func (a *App) PagesPropose(sessionID, title, content, pageID, spaceID, parentID string, expected int) (pages.Review, error) {
	if a.Pages == nil {
		return pages.Review{}, fmt.Errorf("pages store unavailable")
	}
	rev, err := a.Pages.Propose(pages.Review{
		ThreadID: sessionID, SessionID: sessionID, Title: title, Content: content,
		PageID: pageID, SpaceID: spaceID, ParentID: parentID, ExpectedRevision: expected,
	})
	if err != nil {
		return rev, err
	}
	if a.Inbox != nil {
		a.Inbox.Push(inbox.Item{Kind: inbox.KindProposal, Title: "Page: " + rev.Title, Body: rev.Hash, SessionID: sessionID, Key: rev.ID})
	}
	if a.Journal != nil {
		_, _ = a.Journal.Append("page.save", map[string]any{"review": rev.ID, "hash": rev.Hash, "propose": true})
	}
	return rev, nil
}

// PagesReviews 返回待处理的页面审阅。页面库为空返回 nil。
func (a *App) PagesReviews() []pages.Review {
	if a.Pages == nil {
		return nil
	}
	return a.Pages.Pending()
}

// PagesDecide 按 hash 批准或拒绝审阅。成功时记 page.save。
func (a *App) PagesDecide(id, hash string, approve bool) (pages.Page, pages.Review, error) {
	if a.Pages == nil {
		return pages.Page{}, pages.Review{}, fmt.Errorf("pages store unavailable")
	}
	pg, rev, err := a.Pages.Decide(id, hash, approve)
	if err == nil && a.Journal != nil {
		_, _ = a.Journal.Append("page.save", map[string]any{"id": pg.ID, "review": rev.ID, "approve": approve})
	}
	return pg, rev, err
}

// ProfilesList 返回档案列表。档案库为空返回 nil。
func (a *App) ProfilesList() []profile.Profile {
	if a.Profiles == nil {
		return nil
	}
	return a.Profiles.List()
}

// ProfilesGet 读取一份档案。档案库为空返回 profiles unavailable。
func (a *App) ProfilesGet(id string) (profile.Profile, error) {
	if a.Profiles == nil {
		return profile.Profile{}, fmt.Errorf("profiles unavailable")
	}
	return a.Profiles.Get(id)
}

// ProfilesSave 保存档案，记 profile.grant，并停掉正在使用该档案的回合。
func (a *App) ProfilesSave(p profile.Profile) (profile.Profile, error) {
	if a.Profiles == nil {
		return profile.Profile{}, fmt.Errorf("profiles unavailable")
	}
	out, err := a.Profiles.Save(p)
	if err != nil {
		return out, err
	}
	if a.Journal != nil {
		_, _ = a.Journal.Append("profile.grant", map[string]any{"id": out.ID})
	}
	a.abortProfileSessions(out.ID)
	return out, nil
}

// abortProfileSessions 对 ProfileID 匹配且正在跑的会话调用 StopTurn。
func (a *App) abortProfileSessions(profileID string) {
	list, err := a.ListSessions()
	if err != nil {
		return
	}
	for _, m := range list {
		if m.ProfileID == profileID && a.Running(m.ID) {
			a.StopTurn(m.ID)
		}
	}
}

// SetSessionProfile 把空 id 换成默认档案。档案变化且会话正在跑时 StopTurn。返回前 attachAuthMode。
func (a *App) SetSessionProfile(id, profileID string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	if profileID == "" {
		profileID = profile.DefaultID
	}
	if a.Profiles != nil {
		if _, err := a.Profiles.Get(profileID); err != nil {
			return m, err
		}
	}
	prev := m.ProfileID
	m.ProfileID = profileID
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	if prev != profileID && a.Running(id) {
		a.StopTurn(id)
	}
	return a.attachAuthMode(m), nil
}

// SetSessionPage 写入会话的 PageID，返回前 attachAuthMode。
func (a *App) SetSessionPage(id, pageID string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.PageID = strings.TrimSpace(pageID)
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

// sessionProfile 读取会话档案。视频频道、有 ParentID 或 id 含 --task-- 的会话返回 nil。档案 id 为空时用默认档案。
func (a *App) sessionProfile(sessionID string) *profile.Profile {
	if a.Profiles == nil {
		return nil
	}
	m, err := a.GetSession(sessionID)
	if err != nil {
		return nil
	}
	if session.NormalizeChannel(m.Channel) == session.ChannelVideo {
		return nil
	}
	if m.ParentID != "" || strings.Contains(m.ID, "--task--") {
		return nil
	}
	id := m.ProfileID
	if id == "" {
		id = profile.DefaultID
	}
	p, err := a.Profiles.Get(id)
	if err != nil {
		return nil
	}
	return &p
}

// rememberCapture 把 shot 路径写入 ResearchShot。其他种类写入 ResearchBrief，summary 非空时同时写入 Preview。
func (a *App) rememberCapture(sessionID, kind, path, summary string) {
	m, err := a.GetSession(sessionID)
	if err != nil {
		return
	}
	switch kind {
	case "shot":
		m.ResearchShot = path
	default:
		m.ResearchBrief = path
		if summary != "" {
			m.Preview = summary
		}
	}
	_ = a.writeSession(m)
}

// ScheduleRetry 重试一条日程，向收件箱推一条，并在后台 runScheduledJob。
func (a *App) ScheduleRetry(id string) (schedule.Job, error) {
	if a.Schedule == nil {
		return schedule.Job{}, fmt.Errorf("no scheduler")
	}
	j, ok := a.Schedule.Retry(id)
	if !ok {
		return schedule.Job{}, fmt.Errorf("unknown job")
	}
	if a.Inbox != nil {
		a.Inbox.Push(inbox.Item{Kind: inbox.KindSchedule, Title: "Retry after review: " + j.ID, Body: j.Prompt, SessionID: j.SessionID})
	}
	go a.runScheduledJob(j)
	return j, nil
}

// runScheduledJob 在暂停时 Interrupt 并返回。KindFollow 用 StartSendOpts 续上原会话。短片任务交给 runStudioJob，其余交给 runIsolatedJob。
func (a *App) runScheduledJob(j schedule.Job) {
	if a.IsPaused() {
		if a.Schedule != nil {
			a.Schedule.Interrupt(j.ID, "paused")
		}
		return
	}
	if j.Kind == schedule.KindFollow && strings.TrimSpace(j.SessionID) != "" {
		prompt := "[scheduled follow] Continue this conversation. Do not send_as_you without a new approval.\n\n" + j.Prompt
		if a.Inbox != nil {
			a.Inbox.Push(inbox.Item{Kind: inbox.KindHeartbeat, Title: "Follow " + j.ID, Body: j.Prompt, SessionID: j.SessionID})
		}
		err := a.StartSendOpts(j.SessionID, prompt, false, nil)
		if err != nil {
			if a.Schedule != nil {
				a.Schedule.Interrupt(j.ID, err.Error())
			}
			return
		}
		if a.Schedule != nil {
			a.Schedule.Record(j.ID, nil)
		}
		return
	}
	if schedule.IsStudioJob(j) {
		a.runStudioJob(j)
		return
	}
	a.runIsolatedJob(j)
}

// ConnectorMention 把文本发给第一场未归档的代理会话，没有则 NewSession。暂停返回 paused，空文本返回 empty mention。
func (a *App) ConnectorMention(account, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty mention")
	}
	if a.IsPaused() {
		return fmt.Errorf("paused")
	}
	id := ""
	list, _ := a.ListSessions()
	for _, m := range list {
		if session.NormalizeChannel(m.Channel) == session.ChannelAgent && !m.Archived {
			id = m.ID
			break
		}
	}
	if id == "" {
		meta, err := a.NewSession("")
		if err != nil {
			return err
		}
		id = meta.ID
	}
	prefix := "[from connector"
	if strings.TrimSpace(account) != "" {
		prefix += " " + account
	}
	prefix += "]\nHuman HITL stays in Yoyo — approve there if asked.\n\n"
	return a.StartSendOpts(id, prefix+text, false, nil)
}

// VoiceTranscribe 用 speechCreds 把音频 POST 到 /audio/transcriptions。模型空则用 gpt-4o-mini-transcribe。响应体最多读 1 MiB。暂停返回 paused。
func (a *App) VoiceTranscribe(filename string, audio []byte) (string, error) {
	if a.IsPaused() {
		return "", fmt.Errorf("paused")
	}
	endpoint, key, model, err := a.speechCreds()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", firstNonEmpty(model, "gpt-4o-mini-transcribe"))
	fw, err := w.CreateFormFile("file", firstNonEmpty(filename, "audio.webm"))
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	_ = w.Close()
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(endpoint, "/")+"/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("stt %s: %s", resp.Status, bytes.TrimSpace(b))
	}
	var out struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(b, &out) != nil {
		return strings.TrimSpace(string(b)), nil
	}
	return strings.TrimSpace(out.Text), nil
}

// VoiceSpeak 把文本 POST 到 /audio/speech，声音固定 alloy。模型空则用 gpt-4o-mini-tts。响应体最多读 8 MiB。
func (a *App) VoiceSpeak(text string) ([]byte, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", fmt.Errorf("empty speech")
	}
	endpoint, key, model, err := a.speechCreds()
	if err != nil {
		return nil, "", err
	}
	payload, _ := json.Marshal(map[string]any{
		"model": firstNonEmpty(model, "gpt-4o-mini-tts"),
		"input": text,
		"voice": "alloy",
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(endpoint, "/")+"/audio/speech", bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("tts %s: %s", resp.Status, bytes.TrimSpace(b))
	}
	ctype := resp.Header.Get("Content-Type")
	if ctype == "" {
		ctype = "audio/mpeg"
	}
	return b, ctype, nil
}

// VoiceReceipt 往轨迹追加一条语音事件。转写截到 4000 个 rune。轨迹库或会话 id 为空则返回。
func (a *App) VoiceReceipt(sessionID string, durationSec int, transcript string) {
	if a.Traces == nil || sessionID == "" {
		return
	}
	_ = a.Traces.Append(trace.Event{
		TS:        time.Now().UTC(),
		Type:      trace.TypeVoiceCall,
		Source:    "voice",
		SessionID: sessionID,
		Payload: map[string]any{
			"duration_sec": durationSec,
			"transcript":   clipRunes(transcript, 4000),
		},
	})
}

// speechCreds 优先用活动的 speech 连接。端点仍空则用 Config.BaseURL。密钥空则依次租用 vault 的 openai 和 api_key。端点仍空返回 speech connection not configured。
func (a *App) speechCreds() (endpoint, key, model string, err error) {
	if a.Video != nil && a.Video.Conn != nil {
		if c, e := a.Video.Conn.Active(connection.CapSpeech, ""); e == nil {
			endpoint = strings.TrimSpace(c.Endpoint)
			model = connection.ModelFor(c, connection.CapSpeech)
			if k, e2 := a.Video.Conn.Lease(c); e2 == nil {
				key = k
			}
		}
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(a.Config.BaseURL)
	}
	if endpoint == "" {
		return "", "", "", fmt.Errorf("speech connection not configured")
	}
	if key == "" && a.Vault != nil {
		key, _ = a.Vault.Lease("openai")
		if key == "" {
			key, _ = a.Vault.Lease("api_key")
		}
	}
	return endpoint, key, model, nil
}

// firstNonEmpty 返回两个字符串里第一个去掉空白后非空的值。
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// profileMCPAllowed 在允许名单为空时返回真，否则名字或服务器名有一个通过 profile.MCPAllowed 即可。
func profileMCPAllowed(allow []string, name, server string) bool {
	if len(allow) == 0 {
		return true
	}
	return profile.MCPAllowed(allow, name) || profile.MCPAllowed(allow, server)
}

// clipRunes 按 rune 截断。n 小于等于 0 或字符串更短时原样返回。
func clipRunes(s string, n int) string {
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n])
}
