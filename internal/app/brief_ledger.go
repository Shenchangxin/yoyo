package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// briefSealMu 挡住同时写下同一封简报。换会话再推一次时，后一次看见文件就停。
var briefSealMu sync.Mutex

// BriefRecord 是已经写死的一封晨间或晚间简报。正文写过之后不再改。
type BriefRecord struct {
	ID        string `json:"id"`
	Slot      string `json:"slot"`
	Day       string `json:"day"`
	Text      string `json:"text"`
	Chip      string `json:"chip"`
	Brief     string `json:"brief"`
	SessionID string `json:"sessionId,omitempty"`
	At        string `json:"at,omitempty"`
}

func (appHost *deskHost) briefDir() string {
	if appHost.app == nil || appHost.app.Home == nil || strings.TrimSpace(appHost.app.Home.Root) == "" {
		return ""
	}
	return filepath.Join(appHost.app.Home.Root, "briefs")
}

func briefFileName(slot, day string) string {
	slot = strings.TrimSpace(slot)
	day = strings.TrimSpace(day)
	if slot == "" || day == "" {
		return ""
	}
	return day + "-" + slot + ".json"
}

func (appHost *deskHost) briefPath(slot, day string) string {
	dir := appHost.app.briefDir()
	name := briefFileName(slot, day)
	if dir == "" || name == "" {
		return ""
	}
	return filepath.Join(dir, name)
}

// briefRecorded 在这一天这一档已经有文件时为真。文件是第一次写下的那一封。
func (appHost *deskHost) briefRecorded(slot, day string) bool {
	path := appHost.app.briefPath(slot, day)
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}

// rememberSessionBrief 用这条会话里已经写下的那一封建档，不用后来又送来的正文盖掉它。
func (appHost *deskHost) rememberSessionBrief(sessionID, id string) {
	evs, err := appHost.app.Traces.Read(sessionID)
	if err != nil {
		return
	}
	for _, ev := range evs {
		got, _ := ev.Payload["id"].(string)
		if got != id {
			continue
		}
		if rec, ok := briefFromEvent(ev); ok {
			if rec.SessionID == "" {
				rec.SessionID = sessionID
			}
			appHost.app.saveBriefRecord(rec)
		}
		return
	}
}

// rememberBrief 把这一封收进 briefs。文件已经在时不覆盖。
func (appHost *deskHost) rememberBrief(note DeskNote, at time.Time) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	appHost.app.saveBriefRecord(BriefRecord{
		ID:        deskNoteID(note),
		Slot:      strings.TrimSpace(note.Slot),
		Day:       strings.TrimSpace(note.Day),
		Text:      strings.TrimSpace(note.Text),
		Chip:      strings.TrimSpace(note.Chip),
		Brief:     note.Brief,
		SessionID: strings.TrimSpace(note.SessionID),
		At:        at.UTC().Format(time.RFC3339Nano),
	})
}

func (appHost *deskHost) saveBriefRecord(rec BriefRecord) {
	path := appHost.app.briefPath(rec.Slot, rec.Day)
	if path == "" || strings.TrimSpace(rec.Brief) == "" && strings.TrimSpace(rec.Text) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(raw, '\n'))
	_ = f.Close()
}

// BriefList 按写下的时间从新到旧列出已经固定的简报。
func (appHost *deskHost) BriefList() []BriefRecord {
	dir := appHost.app.briefDir()
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []BriefRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var rec BriefRecord
		if json.Unmarshal(raw, &rec) != nil || strings.TrimSpace(rec.Day) == "" || strings.TrimSpace(rec.Slot) == "" {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At != out[j].At {
			return out[i].At > out[j].At
		}
		if out[i].Day != out[j].Day {
			return out[i].Day > out[j].Day
		}
		return out[i].Slot > out[j].Slot
	})
	return out
}

// sealBriefs 用各条会话里最早的那一封简报建档，并删掉之后又推进去的副本。
// 启动时跑一次。之后 DeskPush 看见文件就不再往别的会话追加。
func (appHost *deskHost) sealBriefs() {
	if appHost.app == nil || appHost.app.Traces == nil {
		return
	}
	briefSealMu.Lock()
	defer briefSealMu.Unlock()
	ids, err := appHost.app.Traces.ListSessions()
	if err != nil || len(ids) == 0 {
		return
	}
	type loc struct {
		session string
		index   int
		at      time.Time
		ev      trace.Event
	}
	grouped := map[string][]loc{}
	stored := map[string][]trace.Event{}
	for _, id := range ids {
		evs, err := appHost.app.Traces.Read(id)
		if err != nil {
			continue
		}
		stored[id] = evs
		for i, ev := range evs {
			rec, ok := briefFromEvent(ev)
			if !ok {
				continue
			}
			grouped[rec.ID] = append(grouped[rec.ID], loc{session: id, index: i, at: ev.TS, ev: ev})
		}
	}
	drop := map[string]map[int]bool{}
	for _, locs := range grouped {
		sort.Slice(locs, func(i, j int) bool {
			if !locs[i].at.Equal(locs[j].at) {
				return locs[i].at.Before(locs[j].at)
			}
			if locs[i].session != locs[j].session {
				return locs[i].session < locs[j].session
			}
			return locs[i].index < locs[j].index
		})
		if rec, ok := briefFromEvent(locs[0].ev); ok {
			if rec.SessionID == "" {
				rec.SessionID = locs[0].session
			}
			appHost.app.saveBriefRecord(rec)
		}
		for _, later := range locs[1:] {
			if drop[later.session] == nil {
				drop[later.session] = map[int]bool{}
			}
			drop[later.session][later.index] = true
		}
	}
	for session, indexes := range drop {
		evs := stored[session]
		kept := make([]trace.Event, 0, len(evs))
		for i, ev := range evs {
			if indexes[i] {
				continue
			}
			kept = append(kept, ev)
		}
		if len(kept) == len(evs) {
			continue
		}
		_ = appHost.app.Traces.Replace(session, kept)
	}
}

func briefFromEvent(ev trace.Event) (BriefRecord, bool) {
	if ev.Payload == nil {
		return BriefRecord{}, false
	}
	kind, _ := ev.Payload["kind"].(string)
	if kind != "brief" {
		return BriefRecord{}, false
	}
	slot, _ := ev.Payload["slot"].(string)
	day, _ := ev.Payload["day"].(string)
	slot = strings.TrimSpace(slot)
	day = strings.TrimSpace(day)
	if slot == "" || day == "" {
		return BriefRecord{}, false
	}
	id, _ := ev.Payload["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		id = "brief:" + slot + ":" + day
	}
	text, _ := ev.Payload["text"].(string)
	chip, _ := ev.Payload["chip"].(string)
	brief, _ := ev.Payload["brief"].(string)
	return BriefRecord{
		ID:        id,
		Slot:      slot,
		Day:       day,
		Text:      strings.TrimSpace(text),
		Chip:      strings.TrimSpace(chip),
		Brief:     brief,
		SessionID: ev.SessionID,
		At:        ev.TS.UTC().Format(time.RFC3339Nano),
	}, true
}
