// memory_seen.go 让人看见记忆是怎么用的：每一轮带进对话的旧记忆和常驻画像，以及路由这一轮记下或改口的那几条。
// 两样都写成 source 为 memory 的注入事件，排在这一轮的回复后面。回放历史时它们不进模型（runtime/history.go 只回放 mention、wake 这几类），
// 也不是主机代写的助手事件，不会变成主机记录。界面把它们画成一行回执：带进去的每一条可以说「不对」或「别再提」，记下的可以撤销。
// 只有召回了旧记忆、或者常驻画像和这条会话上一次不一样时才写「带进了哪些」，不每一轮都重复同一张清单。

package app

import (
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// memWritesMax 是一轮里最多留几条记下的回执。
const memWritesMax = 8

// memRef 是界面上的一条记忆：编号、种类、正文和日子。
type memRef struct {
	ID   string
	Kind string
	Text string
	Day  string
}

// memWrote 是路由这一轮记下或改口的一条。Action 取 added 或 replaced。
type memWrote struct {
	ID     string
	Action string
	Text   string
	Before string
}

func memRefOf(it memory.Item) memRef {
	return memRef{ID: it.ID, Kind: string(it.Kind), Text: it.Text, Day: memoryDay(it)}
}

func memRefs(items []memory.Item) []memRef {
	out := make([]memRef, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.ID) == "" {
			continue
		}
		out = append(out, memRefOf(it))
	}
	return out
}

// memoryRecallNote 是 memoryRecall 写给模型的那一段。
func memoryRecallNote(store *memory.Store, message string, now time.Time) string {
	note, _ := memoryRecall(store, message, now)
	return note
}

// profileUse 是常驻画像，以及真正写进去的那几条。
func (a *App) profileUse() (string, []memRef) {
	if a == nil || a.Memory == nil {
		return "", nil
	}
	text, items := a.Memory.ProfileUse(profilePinBytes)
	// 写进画像的记下「用到了」，很久没用的才退到后排（memory/backrow.go）。一天最多记一次，不会每轮写盘。
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	a.Memory.MarkUsed(ids)
	return text, memRefs(items)
}

// noteMemoryWrite 记下路由这一次写成了什么，这一轮说完之后写成回执。已经记着的和没记的不写。
func (a *App) noteMemoryWrite(sessionID string, kept memory.Kept) {
	if a == nil || strings.TrimSpace(sessionID) == "" || kept.Item.ID == "" {
		return
	}
	w := memWrote{ID: kept.Item.ID, Text: kept.Item.Text, Before: kept.Before}
	switch {
	case kept.Action == "added":
		w.Action = "added"
		w.Before = ""
	case (kept.Action == "replaced" || kept.Action == "merged") && kept.Before != "" && kept.Before != kept.Item.Text:
		w.Action = "replaced"
	default:
		return
	}
	a.memSeenMu.Lock()
	defer a.memSeenMu.Unlock()
	if a.memWrites == nil {
		a.memWrites = map[string][]memWrote{}
	}
	rows := append(a.memWrites[sessionID], w)
	if len(rows) > memWritesMax {
		rows = rows[len(rows)-memWritesMax:]
	}
	a.memWrites[sessionID] = rows
}

// emitMemorySeen 在一轮说完之后写回执：先是这一轮记下的，再是带进对话的。没有可写的就什么都不写。
func (a *App) emitMemorySeen(sessionID, turnID string, pinned, recalled []memRef) {
	if a == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	a.memSeenMu.Lock()
	writes := a.memWrites[sessionID]
	delete(a.memWrites, sessionID)
	sig := pinSig(pinned)
	pinChanged := false
	if len(pinned) > 0 {
		if a.memPinSig == nil {
			a.memPinSig = map[string]string{}
		}
		pinChanged = a.memPinSig[sessionID] != sig
		a.memPinSig[sessionID] = sig
	}
	a.memSeenMu.Unlock()
	if len(writes) > 0 {
		rows := make([]map[string]any, 0, len(writes))
		var lines []string
		for _, w := range writes {
			row := map[string]any{"id": w.ID, "action": w.Action, "text": w.Text}
			if w.Before != "" {
				row["before"] = w.Before
			}
			rows = append(rows, row)
			if w.Action == "replaced" {
				lines = append(lines, "改好了："+w.Before+" → "+w.Text)
			} else {
				lines = append(lines, "记下了："+w.Text)
			}
		}
		a.memoryEvent(sessionID, turnID, map[string]any{
			"kind": "wrote", "name": "memory", "items": rows, "text": strings.Join(lines, "\n"),
		})
	}
	if len(recalled) == 0 && !pinChanged {
		return
	}
	a.memoryEvent(sessionID, turnID, map[string]any{
		"kind": "used", "name": "memory",
		"recalled": memRows(recalled), "pinned": memRows(pinned),
		"text": usedText(len(recalled), len(pinned)),
	})
}

func usedText(recalled, pinned int) string {
	var parts []string
	if recalled > 0 {
		parts = append(parts, "这句话召回了 "+itoa(recalled)+" 条旧记忆")
	}
	if pinned > 0 {
		parts = append(parts, "常驻画像 "+itoa(pinned)+" 条")
	}
	return "这一轮带进了记忆：" + strings.Join(parts, "，")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func memRows(refs []memRef) []map[string]any {
	rows := make([]map[string]any, 0, len(refs))
	for _, r := range refs {
		rows = append(rows, map[string]any{"id": r.ID, "kind": r.Kind, "text": r.Text, "day": r.Day})
	}
	return rows
}

// pinSig 是常驻画像这一版的样子：哪几条、写成什么。
func pinSig(refs []memRef) string {
	keys := make([]string, 0, len(refs))
	for _, r := range refs {
		keys = append(keys, r.ID+"="+r.Text)
	}
	sort.Strings(keys)
	return strings.Join(keys, "\n")
}

func (a *App) memoryEvent(sessionID, turnID string, payload map[string]any) {
	ev := trace.Event{
		TS:        time.Now().UTC(),
		Type:      trace.TypeInject,
		Source:    "memory",
		SessionID: sessionID,
		TurnID:    turnID,
		Payload:   payload,
	}
	if a.Traces != nil {
		if err := a.Traces.Append(ev); err != nil {
			return
		}
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
}

// MemoryMute 让 id 这一条不再带进对话，或者重新带进去。条目留着，记忆页还看得到。
func (a *App) MemoryMute(id string, muted bool) bool {
	if a == nil || a.Memory == nil {
		return false
	}
	ok := a.Memory.SetMuted(id, muted)
	if ok {
		a.forgetPinSig()
		a.touchPoi()
	}
	return ok
}

// MemoryRevert 把 id 这一条退回上一次的说法。回执上「撤销改口」用它。退不了时返回零值。
func (a *App) MemoryRevert(id string) memory.Item {
	if a == nil || a.Memory == nil {
		return memory.Item{}
	}
	it, ok := a.Memory.Revert(id)
	if !ok {
		return memory.Item{}
	}
	a.forgetPinSig()
	a.touchPoi()
	return it
}

// forgetPinSig 让下一轮重新写一次带进了哪些：人刚改过记忆，该让他看到改完的样子。
func (a *App) forgetPinSig() {
	a.memSeenMu.Lock()
	a.memPinSig = nil
	a.memSeenMu.Unlock()
}
