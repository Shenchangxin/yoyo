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

// DeskPush 把一条通知写进会话。同一 id 已经在轨迹里时不再写，但仍然返回真，表示这条已经送达。它不开始模型回合。
func (a *App) DeskPush(note DeskNote) bool {
	if a == nil || a.Traces == nil {
		return false
	}
	sessionID := strings.TrimSpace(note.SessionID)
	text := strings.TrimSpace(note.Text)
	if sessionID == "" || text == "" {
		return false
	}
	id := deskNoteID(note)
	if a.traceHasID(sessionID, id) {
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
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
	return true
}

func deskNoteID(note DeskNote) string {
	kind := strings.TrimSpace(note.Kind)
	day := strings.TrimSpace(note.Day)
	if kind == "brief" {
		return "brief:" + strings.TrimSpace(note.Slot) + ":" + day
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
	return a.Poi.Refresh(obs, blobs, time.Now())
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
			if it.Staging || strings.TrimSpace(it.Text) == "" {
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

func memoryKind(kind memory.Kind) string {
	switch kind {
	case memory.KindProject, memory.KindEpisodic, memory.KindSemantic, memory.KindProfile:
		return string(kind)
	default:
		return string(memory.KindProfile)
	}
}
