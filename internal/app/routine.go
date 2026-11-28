// routine.go 让定时例行的结果也走开口门。例行在自己的会话里跑完，结果收成一条 routine 感知，由送达循环决定开不开口。
// 例行自己判断没什么可说时回 [quiet]，结果只留在它自己的会话里。每次运行最多说一次；人按住的那一条例行不说；
// 忽略三次会一直按住。失败一定开口，并说清没跑成的原因。
// 还没送到的结果写在 memory/routines.json：在跟随的进程里手动跑的例行（重试、webhook），领头的送达循环也看得到。
// 不拥有：到期判断（schedule 包）和跟进会话的续写（KindFollow 留在原会话里）。

package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/lease"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// routineQuiet 是例行觉得没什么可说时的回答。
const routineQuiet = "[quiet]"

// routineTimeout 是一次例行最多跑多久。等审批也算在里面，没人批就到点停。
const routineTimeout = 30 * time.Minute

// routineKeep 是没送到的结果留多久。过了就不再说，免得隔天说一件旧事。
const routineKeep = 20 * time.Hour

// routineMax 是最多留几条没送到的结果，多了丢最早的。
const routineMax = 40

// routineAsk 接在例行提示后面。它说明没人看着，需要批准的动作先别做。
const routineAsk = "\n\n---\n这是一条定时例行，此刻没有人在看着。需要人批准的动作（发消息、改动外部应用、花钱）先不要做，写成一个问题留给人。做完用两三句写给人看的结论，第一句就是结论。没有值得告诉人的新东西时，只回 " + routineQuiet + "。"

// routineResult 是跑完、还没经过开口门送到的一次例行。
type routineResult struct {
	JobID     string    `json:"jobId"`
	Title     string    `json:"title"`
	SessionID string    `json:"sessionId,omitempty"`
	State     string    `json:"state"`
	Text      string    `json:"text,omitempty"`
	At        time.Time `json:"at"`
}

// runRoutine 在 sessionID 里同步跑一次例行，等它说完，把结果交给开口门。返回这次运行的错误。
func (a *App) runRoutine(j schedule.Job, sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), routineTimeout)
	defer cancel()
	out, err := a.SendOpts(ctx, sessionID, j.Prompt+routineAsk, nil, nil, false)
	a.noteRoutine(j, sessionID, out, err, time.Now())
	return err
}

// routineSaysQuiet 在例行的回答只是 [quiet]（可以带标点和很短的一句）时为真。
func routineSaysQuiet(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	if !strings.Contains(strings.ToLower(text), routineQuiet) {
		return false
	}
	return len([]rune(text)) <= len(routineQuiet)+20
}

func (a *App) routinePath() string {
	if a == nil || a.Home == nil {
		return ""
	}
	return filepath.Join(a.Home.Memory(), "routines.json")
}

// routineEdit 在锁里读出还没送到的结果，交给 fn 改，再写回。没有 home 时只在内存里。
func (a *App) routineEdit(fn func([]routineResult) []routineResult) {
	a.routineMu.Lock()
	defer a.routineMu.Unlock()
	path := a.routinePath()
	if path == "" {
		a.routines = fn(a.routines)
		return
	}
	unlock, _ := lease.Lock(path + ".lock")
	defer unlock()
	var rs []routineResult
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &rs)
	}
	before := len(rs)
	out := fn(rs)
	a.routines = out
	if len(out) == 0 && before == 0 {
		return
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// noteRoutine 记下一次例行的结果。安静的不留。
func (a *App) noteRoutine(j schedule.Job, sessionID, out string, err error, now time.Time) {
	text := strings.TrimSpace(out)
	state := "done"
	if err != nil {
		state = "failed"
		text = scrubSense(err.Error())
		if text == "" {
			text = "出错了"
		}
	} else if routineSaysQuiet(text) {
		return
	}
	res := routineResult{
		JobID:     j.ID,
		Title:     routineTitle(j.Prompt),
		SessionID: sessionID,
		State:     state,
		Text:      clipBrief(text, 600),
		At:        now,
	}
	a.routineEdit(func(rs []routineResult) []routineResult {
		rs = append(rs, res)
		if len(rs) > routineMax {
			rs = rs[len(rs)-routineMax:]
		}
		return rs
	})
}

// routineFacts 把还没送到的例行结果收成感知。过期的先丢掉。
func (a *App) routineFacts(now time.Time) []initiative.Fact {
	var facts []initiative.Fact
	a.routineEdit(func(rs []routineResult) []routineResult {
		kept := rs[:0]
		for _, r := range rs {
			if now.Sub(r.At) > routineKeep {
				continue
			}
			kept = append(kept, r)
			facts = append(facts, routineFact(r))
		}
		return kept
	})
	return facts
}

func routineFact(r routineResult) initiative.Fact {
	rank := 2
	reason := "你设的例行「" + r.Title + "」跑完了，它觉得这件值得说"
	if r.State == "failed" {
		rank = 1
		reason = "你设的例行「" + r.Title + "」没跑成，不说的话你会以为它跑过了"
	}
	var refs []initiative.Ref
	if r.SessionID != "" {
		refs = append(refs, initiative.Ref{Kind: "session", Label: "这次运行", Target: r.SessionID})
	}
	return initiative.Fact{
		ID:       routineFactID(r),
		Code:     "routine",
		Class:    "routine:" + r.JobID,
		Name:     r.Title,
		Detail:   r.Text,
		State:    r.State,
		Sig:      r.State,
		Rank:     rank,
		Eligible: true,
		Reason:   reason,
		Refs:     refs,
	}
}

// routineDelivered 丢掉已经送到的那几次。
func (a *App) routineDelivered(sent []initiative.Signal) {
	done := map[string]bool{}
	for _, s := range sent {
		if s.Code == "routine" {
			done[s.ID] = true
		}
	}
	if len(done) == 0 {
		return
	}
	a.routineEdit(func(rs []routineResult) []routineResult {
		kept := rs[:0]
		for _, r := range rs {
			if done[routineFactID(r)] {
				continue
			}
			kept = append(kept, r)
		}
		return kept
	})
}

func routineFactID(r routineResult) string {
	return "routine:" + r.JobID + "@" + r.At.UTC().Format("20060102T150405")
}

// routineTitle 取提示的第一行当名字，最多 40 个字。
func routineTitle(prompt string) string {
	line := strings.TrimSpace(prompt)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if line == "" || scrubSense(line) == "" {
		return "例行"
	}
	return clipBrief(line, 40)
}
