package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/poi"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// DeskNote 是推进主会话的一条简报或兴趣变化。Brief 是简报 JSON，兴趣变化时留空。
type DeskNote struct {
	SessionID string `json:"sessionId"`
	Kind      string `json:"kind"`
	Slot      string `json:"slot"`
	Day       string `json:"day"`
	Text      string `json:"text"`
	Chip      string `json:"chip"`
	Brief     string `json:"brief"`
}

// DeskPush 把一条通知写进会话。同一 id 已经在轨迹里时不再写，但仍然返回真，表示这条已经送达。
// 简报还要看磁盘上的那一封：同一天同一档已经写下之后，换一条会话也不再追加。它不开始模型回合。
func (a *App) DeskPush(note DeskNote) bool {
	if a == nil || a.Traces == nil {
		return false
	}
	sessionID := strings.TrimSpace(note.SessionID)
	text := strings.TrimSpace(note.Text)
	if sessionID == "" || text == "" {
		return false
	}
	kind := strings.TrimSpace(note.Kind)
	if kind == "brief" {
		briefSealMu.Lock()
		defer briefSealMu.Unlock()
		if a.briefRecorded(note.Slot, note.Day) {
			return true
		}
	}
	id := deskNoteID(note)
	if a.traceHasID(sessionID, id) {
		if kind == "brief" {
			a.rememberSessionBrief(sessionID, id)
		}
		return true
	}
	brief := note.Brief
	if len(brief) > 32_000 {
		brief = brief[:32_000]
	}
	ev := trace.Event{
		TS:        time.Now().UTC(),
		Type:      trace.TypeAssistant,
		Source:    "desk",
		SessionID: sessionID,
		Payload: map[string]any{
			"text":  text,
			"id":    id,
			"kind":  strings.TrimSpace(note.Kind),
			"slot":  strings.TrimSpace(note.Slot),
			"day":   strings.TrimSpace(note.Day),
			"chip":  strings.TrimSpace(note.Chip),
			"brief": brief,
		},
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return false
	}
	if kind == "brief" {
		a.rememberBrief(note, ev.TS)
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
	return true
}

func deskNoteID(note DeskNote) string {
	kind := strings.TrimSpace(note.Kind)
	day := strings.TrimSpace(note.Day)
	if kind == "brief" || kind == "speak" {
		return kind + ":" + strings.TrimSpace(note.Slot) + ":" + day
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(note.Text)))
	return kind + ":" + day + ":" + hex.EncodeToString(sum[:6])
}

func (a *App) traceHasID(sessionID, id string) bool {
	evs, err := a.Traces.Read(sessionID)
	if err != nil {
		return false
	}
	for _, ev := range evs {
		got, _ := ev.Payload["id"].(string)
		if got == id {
			return true
		}
	}
	return false
}

// PoiRefresh 用记忆和桌面表层名字刷新兴趣库。
func (a *App) PoiRefresh() poi.Snapshot {
	if a == nil || a.Poi == nil {
		return poi.Snapshot{}
	}
	obs, blobs := a.poiObservations()
	now := time.Now()
	snap := a.Poi.Refresh(obs, blobs, now)
	snap.Census = poi.ScanCensus(poi.DefaultRoots(), now)
	for i := range snap.Census.Lines {
		snap.Census.Lines[i].ID = memory.MachineID("project", snap.Census.Lines[i].Name)
	}
	a.fileCensus(snap.Census)
	if briefing, ok := a.assessDesk(snap); ok {
		snap.Initiative = &briefing
	}
	return snap
}

// PoiAck 记下这一版兴趣库已经推进过。
func (a *App) PoiAck(sig string) bool {
	if a == nil || a.Poi == nil {
		return false
	}
	return a.Poi.Ack(sig)
}

func (a *App) touchPoi() {
	if a == nil || a.Poi == nil {
		return
	}
	obs, blobs := a.poiObservations()
	a.Poi.Refresh(obs, blobs, time.Now())
}

func (a *App) poiObservations() ([]poi.Observation, []string) {
	var obs []poi.Observation
	var blobs []string
	if a.Memory != nil {
		for _, it := range a.Memory.All() {
			if it.Staging || it.Source == "machine" || strings.TrimSpace(it.Text) == "" {
				continue
			}
			obs = append(obs, poi.Observation{Name: it.Text, Kind: memoryKind(it.Kind), Evidence: "memory"})
			blobs = append(blobs, it.Text)
		}
	}
	now := time.Now()
	obs = append(obs, poi.ScanRoots(poi.DefaultRoots(), now)...)
	obs = append(obs, poi.ScanAppRoots(poi.DefaultAppRoots(), now)...)
	return obs, blobs
}

// fileCensus 把这一轮清点收成记忆。同一条线用稳定 id，没有变化就不改更新时间。
func (a *App) fileCensus(c poi.Census) {
	if a == nil || a.Memory == nil {
		return
	}
	var facts []memory.MachineFact
	if user := strings.TrimSpace(c.User); user != "" {
		facts = append(facts, memory.MachineFact{ID: "machine:device", Kind: string(memory.KindProfile), Project: "device", Text: user})
	}
	for _, fact := range []memory.MachineFact{
		{ID: "machine:host", Text: c.Host},
		{ID: "machine:os", Text: c.OS},
		{ID: "machine:zone", Text: c.Zone},
		{ID: "machine:git", Text: c.Git},
	} {
		if strings.TrimSpace(fact.Text) == "" {
			continue
		}
		fact.Kind = string(memory.KindProfile)
		fact.Project = "device"
		facts = append(facts, fact)
	}
	if strings.TrimSpace(c.Hours) != "" {
		facts = append(facts, memory.MachineFact{ID: "machine:hours", Kind: string(memory.KindProfile), Project: "work", Text: c.Hours})
	}
	for _, line := range c.Lines {
		name := strings.TrimSpace(line.Name)
		if name == "" {
			continue
		}
		facts = append(facts, memory.MachineFact{
			ID:       memory.MachineID("project", name),
			Kind:     string(memory.KindProject),
			Project:  name,
			Text:     name,
			State:    line.State,
			Evidence: line.Evidence,
			When:     line.When,
		})
	}
	if a.Memory.FileMachine(facts) {
		a.touchPoi()
	}
}

func memoryKind(kind memory.Kind) string {
	switch kind {
	case memory.KindProject, memory.KindEpisodic, memory.KindSemantic, memory.KindProfile:
		return string(kind)
	default:
		return string(memory.KindProfile)
	}
}
