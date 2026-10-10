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

func (a *App) IsPaused() bool {
	return a != nil && a.Config.Paused
}

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
			TS:     time.Now().UTC(),
			Type:   trace.TypePause,
			Source: "app",
			Payload: map[string]any{"paused": paused},
		})
	}
	return a.SaveConfig()
}

func (a *App) PagesList(q, spaceID string) []pages.Meta {
	if a.Pages == nil {
		return nil
	}
	if strings.TrimSpace(q) != "" {
		return a.Pages.Search(q, spaceID)
	}
	return a.Pages.List(spaceID)
}

func (a *App) PagesGet(spaceID, id string) (pages.Page, error) {
	if a.Pages == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	return a.Pages.Get(spaceID, id)
}

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

func (a *App) PagesReviews() []pages.Review {
	if a.Pages == nil {
		return nil
	}
	return a.Pages.Pending()
}

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

func (a *App) ProfilesList() []profile.Profile {
	if a.Profiles == nil {
		return nil
	}
	return a.Profiles.List()
}

func (a *App) ProfilesGet(id string) (profile.Profile, error) {
	if a.Profiles == nil {
		return profile.Profile{}, fmt.Errorf("profiles unavailable")
	}
	return a.Profiles.Get(id)
}

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
	a.runIsolatedJob(j)
}

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

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func profileMCPAllowed(allow []string, name, server string) bool {
	if len(allow) == 0 {
		return true
	}
	return profile.MCPAllowed(allow, name) || profile.MCPAllowed(allow, server)
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n])
}
