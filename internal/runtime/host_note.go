// host_note.go 把主机直接写进会话的内容收成「主机记录」：工人的回报、通知、简报和提醒。
// 这些话已经显示给人看过，但不是这条会话的模型说的。回放历史时它们并成一条用户角色的记录，放在下一句人话前面；
// 常驻的桌面代理靠 HostLines 在下一轮开始前补上还没见过的那几条。两条路用同一个 HostNote，写出的字一样。

package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// hostNotePrefix 是主机记录的开头。isControlUser 认它，记录不会被当成人的要求。
const hostNotePrefix = "（主机记录，不是人说的话。"

// hostNoteLead 是主机记录的第一行，告诉模型这些话的来历和该怎么用。
const hostNoteLead = hostNotePrefix + "下面这些已经直接显示给人看过，不用回应；人问起时据此回答。）"

const (
	// hostLineRunes 是一条记录最多留多少字。更长的在工单里，路由要细节时用 task_read。
	hostLineRunes = 600
	// hostNoteLines 是一条主机记录最多带几行。人离开几天后攒下的旧行只报个数。
	hostNoteLines = 16
)

// HostLine 是主机代写的一行。ID 是那条事件的编号，桌面代理用它记住哪些已经给模型看过。
type HostLine struct {
	ID   string
	Text string
}

// HostWritten 判断 ev 是不是主机代写的助手事件：来源是工人、通知或桌面推送。模型自己的回复来源是 model。
func HostWritten(ev trace.Event) bool {
	if ev.Type != trace.TypeAssistant {
		return false
	}
	switch ev.Source {
	case "worker", "notice", "desk":
		return true
	}
	return false
}

// HostLineOf 把一条主机代写的事件写成一行。正文为空的不算。
func HostLineOf(ev trace.Event) (HostLine, bool) {
	if !HostWritten(ev) {
		return HostLine{}, false
	}
	p := ev.Payload
	text := strings.Join(strings.Fields(payloadStr(p, "text")), " ")
	kind := payloadStr(p, "kind")
	line := ""
	switch {
	case ev.Source == "worker" && kind == "group":
		// 任务组卡片的正文是给界面的，这里按编号、标题和方式重写一句。
		mode := "分工"
		if payloadStr(p, "mode") == "compare" {
			mode = "对比"
		}
		line = fmt.Sprintf("任务组 %s「%s」已开出（%s）", payloadStr(p, "group"), payloadStr(p, "title"), mode)
	case text == "":
		return HostLine{}, false
	case ev.Source == "notice":
		line = "通知：" + text
	case ev.Source == "desk" && kind == "brief":
		slot := "晨间"
		if payloadStr(p, "slot") == "evening" {
			slot = "晚间"
		}
		line = fmt.Sprintf("%s简报（%s）：%s", slot, payloadStr(p, "day"), text)
	case ev.Source == "desk":
		line = "提醒：" + text
	default:
		who := "任务 " + payloadStr(p, "ticket")
		if name := payloadStr(p, "agentName"); name != "" {
			who += "（" + name + "）"
		}
		line = fmt.Sprintf("%s的回报 · %s：%s", who, hostStateWord(kind, payloadStr(p, "state")), text)
	}
	id := payloadStr(p, "id")
	if id == "" {
		id = ev.Source + ":" + ev.TS.UTC().Format("20060102T150405.000000000")
	}
	return HostLine{ID: id, Text: capRunes(line, hostLineRunes)}, true
}

// hostStateWord 用人话说这份回报是什么状态。kind 为 handoff 时是在等人。
func hostStateWord(kind, state string) string {
	switch state {
	case "ready":
		return "做完了"
	case "waiting":
		return "停下等下一句"
	case "stopped":
		return "已经停下"
	case "failed":
		return "没做成"
	case "blocked":
		return "卡住了，在等人"
	case "dropped":
		return "不用做，放下了"
	case "delivery":
		return "结果没送出去"
	case "question":
		return "在等人回答"
	}
	if kind == "handoff" {
		return "在等人处理"
	}
	return "有进展"
}

// HostLines 按先后列出 evs 里主机代写的行。调用方传最后一次检查点之后的事件。
func HostLines(evs []trace.Event) []HostLine {
	var out []HostLine
	for _, ev := range evs {
		if delta, _ := ev.Payload["delta"].(bool); delta {
			continue
		}
		if line, ok := HostLineOf(ev); ok {
			out = append(out, line)
		}
	}
	return out
}

// HostNote 把几行并成一条主机记录。行数超过 hostNoteLines 时只留最近的，前面写明略去几条。没有行时返回空串。
func HostNote(lines []HostLine) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(hostNoteLead)
	if skipped := len(lines) - hostNoteLines; skipped > 0 {
		fmt.Fprintf(&b, "\n（更早的 %d 条略去，要看用 task_status 或 task_read。）", skipped)
		lines = lines[skipped:]
	}
	for _, line := range lines {
		b.WriteString("\n- ")
		b.WriteString(line.Text)
	}
	return b.String()
}

// IsHostNote 判断一条用户角色的正文是不是主机记录。
func IsHostNote(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), hostNotePrefix)
}
