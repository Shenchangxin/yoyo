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

func (a *App) briefDir() string {
	if a == nil || a.Home == nil || strings.TrimSpace(a.Home.Root) == "" {
		return ""
	}
	return filepath.Join(a.Home.Root, "briefs")
}

func briefFileName(slot, day string) string {
	slot = strings.TrimSpace(slot)
	day = strings.TrimSpace(day)
	if slot == "" || day == "" {
		return ""
	}
	return day + "-" + slot + ".json"
}

func (a *App) briefPath(slot, day string) string {
	dir := a.briefDir()
	name := briefFileName(slot, day)
	if dir == "" || name == "" {
		return ""
	}
	return filepath.Join(dir, name)
}

// briefRecorded 在这一天这一档已经有文件时为真。文件是第一次写下的那一封。
func (a *App) briefRecorded(slot, day string) bool {
	path := a.briefPath(slot, day)
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}

// rememberSessionBrief 用这条会话里已经写下的那一封建档，不用后来又送来的正文盖掉它。
func (a *App) rememberSessionBrief(sessionID, id string) {
	evs, err := a.Traces.Read(sessionID)
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
			a.saveBriefRecord(rec)
		}
		return
	}
}

// rememberBrief 把这一封收进 briefs。文件已经在时不覆盖。
func (a *App) rememberBrief(note DeskNote, at time.Time) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	a.saveBriefRecord(BriefRecord{
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

func (a *App) saveBriefRecord(rec BriefRecord) {
	path := a.briefPath(rec.Slot, rec.Day)
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
func (a *App) BriefList() []BriefRecord {
	dir := a.briefDir()
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
func (a *App) sealBriefs() {
	if a == nil || a.Traces == nil {
		return
	}
	briefSealMu.Lock()
	defer briefSealMu.Unlock()
	ids, err := a.Traces.ListSessions()
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
		evs, err := a.Traces.Read(id)
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
			a.saveBriefRecord(rec)
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
		_ = a.Traces.Replace(session, kept)
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
