package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/poi"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// DeskNote 是推进主会话的一条简报、开口短讯或兴趣变化。Brief 是简报 JSON，其余时候留空。
// Class、Why、Name、Detail、State 和 MemoryID 只在开口短讯上有：界面按它们写这一行，反馈记到 Class 这一类。
type DeskNote struct {
	SessionID string `json:"sessionId"`
	Kind      string `json:"kind"`
	Slot      string `json:"slot"`
	Day       string `json:"day"`
	Text      string `json:"text"`
	Chip      string `json:"chip"`
	Brief     string `json:"brief"`
	Class     string `json:"class,omitempty"`
	Why       string `json:"why,omitempty"`
	Name      string `json:"name,omitempty"`
	Detail    string `json:"detail,omitempty"`
	State     string `json:"state,omitempty"`
	MemoryID  string `json:"memoryId,omitempty"`
	// Reason 是「为什么是这条」，Refs 是这条依据的材料。只在开口短讯上有。
	Reason string           `json:"reason,omitempty"`
	Refs   []initiative.Ref `json:"refs,omitempty"`
	// Quiet 只用在简报上：这封信写下来留档，拉开今天能读到，但不往会话里放芯片。
	Quiet bool `json:"quiet,omitempty"`
}

// DeskPush 把一条通知写进会话。同一 id 已经在轨迹里时不再写，但仍然返回真，表示这条已经送达。
// 简报还要看磁盘上的那一封：同一天同一档已经写下之后，换一条会话也不再追加。它不开始模型回合。
func (appHost *deskHost) DeskPush(note DeskNote) bool {
	if appHost.app == nil || appHost.app.Traces == nil {
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
		if appHost.app.briefRecorded(note.Slot, note.Day) {
			return true
		}
		if note.Quiet {
			appHost.app.rememberBrief(note, time.Now().UTC())
			return appHost.app.briefRecorded(note.Slot, note.Day)
		}
	}
	id := deskNoteID(note)
	if appHost.app.traceHasID(sessionID, id) {
		if kind == "brief" {
			appHost.app.rememberSessionBrief(sessionID, id)
		}
		return true
	}
	brief := note.Brief
	if len(brief) > 1<<20 {
		return false
	}
	payload := map[string]any{
		"text":  text,
		"id":    id,
		"kind":  strings.TrimSpace(note.Kind),
		"slot":  strings.TrimSpace(note.Slot),
		"day":   strings.TrimSpace(note.Day),
		"chip":  strings.TrimSpace(note.Chip),
		"brief": brief,
	}
	for key, value := range map[string]string{
		"class": note.Class, "why": note.Why, "name": note.Name, "detail": note.Detail, "state": note.State, "memoryId": note.MemoryID,
		"reason": note.Reason,
	} {
		if value = strings.TrimSpace(value); value != "" {
			payload[key] = value
		}
	}
	if len(note.Refs) > 0 {
		refs := make([]map[string]any, 0, len(note.Refs))
		for _, ref := range note.Refs {
			row := map[string]any{"kind": ref.Kind, "label": ref.Label}
			if ref.Target != "" {
				row["target"] = ref.Target
			}
			refs = append(refs, row)
		}
		payload["refs"] = refs
	}
	ev := trace.Event{
		TS:        time.Now().UTC(),
		Type:      trace.TypeAssistant,
		Source:    "desk",
		SessionID: sessionID,
		Payload:   payload,
	}
	if err := appHost.app.Traces.AppendDurable(ev); err != nil {
		return false
	}
	if kind == "brief" {
		appHost.app.rememberBrief(note, ev.TS)
	}
	if appHost.app.Hub != nil {
		appHost.app.Hub.Publish(ev)
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

func (appHost *deskHost) traceHasID(sessionID, id string) bool {
	evs, err := appHost.app.Traces.Read(sessionID)
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
func (appHost *deskHost) PoiRefresh() poi.Snapshot {
	if appHost.app == nil || appHost.app.Poi == nil {
		return poi.Snapshot{}
	}
	obs, blobs := appHost.app.poiObservations()
	now := time.Now()
	snap := appHost.app.Poi.Refresh(obs, blobs, now)
	snap.Census = poi.ScanCensus(poi.DefaultRoots(), now)
	for i := range snap.Census.Lines {
		snap.Census.Lines[i].ID = memory.MachineID("project", snap.Census.Lines[i].Name)
	}
	appHost.app.fileCensus(snap.Census)
	if briefing, ok := appHost.app.assessDesk(snap); ok {
		snap.Initiative = &briefing
	}
	return snap
}

// PoiAck 记下这一版兴趣库已经推进过。
func (appHost *deskHost) PoiAck(sig string) bool {
	if appHost.app == nil || appHost.app.Poi == nil {
		return false
	}
	return appHost.app.Poi.Ack(sig)
}

func (appHost *deskHost) touchPoi() {
	if appHost.app == nil || appHost.app.Poi == nil {
		return
	}
	obs, blobs := appHost.app.poiObservations()
	appHost.app.Poi.Refresh(obs, blobs, time.Now())
}

func (appHost *deskHost) poiObservations() ([]poi.Observation, []string) {
	var obs []poi.Observation
	var blobs []string
	if appHost.app.Memory != nil {
		for _, it := range appHost.app.Memory.All() {
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
func (appHost *deskHost) fileCensus(c poi.Census) {
	if appHost.app == nil || appHost.app.Memory == nil {
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
	if appHost.app.Memory.FileMachine(facts) {
		appHost.app.touchPoi()
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
