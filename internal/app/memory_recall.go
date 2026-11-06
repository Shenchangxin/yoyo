package app

import (
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

// recallBytes 是每句话附带的旧记忆的字节上限。
const recallBytes = 600

// memoryOn 在这条会话用得上记忆时为真：有记忆库，会话的身份没有关掉记忆。
func (a *App) memoryOn(sessionID string) bool {
	if a == nil || a.Memory == nil {
		return false
	}
	p := a.sessionProfile(sessionID)
	return p == nil || p.Memory
}

// memoryRecallNote 是这句话可能用得上、又不在常驻画像里的旧记忆：做过的事、项目和过去的安排。没有足够相关的就是空串。
// 它跟着这一句进对话，不进系统提示，所以不动前缀缓存。
func memoryRecallNote(store *memory.Store, message string, now time.Time) string {
	if store == nil || strings.TrimSpace(message) == "" {
		return ""
	}
	var b strings.Builder
	for _, it := range store.Recall(message, 3) {
		line := "- " + recallLabel(it, now) + it.Text + "\n"
		if b.Len() > 0 && b.Len()+len(line) > recallBytes {
			break
		}
		b.WriteString(line)
	}
	if b.Len() == 0 {
		return ""
	}
	return "Earlier memory that may relate to this message (dates are when it was noted; check before relying on it):\n" + strings.TrimRight(b.String(), "\n")
}

// recallLabel 是一条旧记忆前面的日子和来历。
func recallLabel(it memory.Item, now time.Time) string {
	day := memoryDay(it)
	switch {
	case memory.Expired(it, now):
		return day + " · past plan: "
	case it.Kind == memory.KindEpisodic && (it.Source == "worker" || strings.HasPrefix(it.ID, "task:")):
		return day + " · work done: "
	case it.Kind == memory.KindEpisodic:
		return day + " · earlier: "
	case it.Kind == memory.KindProject:
		return day + " · project: "
	default:
		return day + ": "
	}
}

// memoryDay 是这一条最近一次记下或改过的那一天。
func memoryDay(it memory.Item) string {
	return it.Touched().Local().Format("2006-01-02")
}
