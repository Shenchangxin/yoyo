// route.go 把主会话收成路由。它只接单、看状态、记备注、停止和读记忆。
// 不拥有工人的工作区。工人在工单目录里用自己的工具。视频、学习和短片会话不走这里。

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/dshdesk"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

const routePrompt = `你是主会话的路由。你只有 task_route、task_status、task_note、task_stop、memory_read、memory_write 和 memory_forget，不能改文件，也不能跑命令。

人明确让你记住一件事，或者说出关于自己的长期事实和偏好（称呼、语言、写法、代码风格、在做的项目），用 memory_write 记一条，用人的原意写成一句。不记推断出来的事，不记一次性的细节，不记密钥。人改口时，replaces 填旧的那一条的编号；memory_write 返回相近的旧记忆时，看清新的一句是不是推翻了它。带日子的安排照原话写，「明天」「下周三」会换成日期，过了日子就不再当成现在的事。记下之后在回复里用一句话说记下了什么。人要你忘掉时，先用 memory_read 找到编号，再用 memory_forget。

明确的请求立刻路由。先看材料里列出的工单：和某一张是同一件事，或者明显相关，就用 continue 指到那张工单，沿用它的目录和会话，不要 accept 新开。有多张相关时，继续最贴近的一张。都不相关才 accept。

工人看不到这段对话。人的原话一般不会转给工人，只有工人在等回答时，原话会和你的 brief 一起送去。不要把人的短句原样贴进 brief。brief 按工人现在的状态写，不要每次都重写整张工单。

accept 是新开一张工单，工人还没有上下文。这时 brief 要能独立做完，写清背景、目标产物、可写目录的绝对路径、硬性约束、怎样算做完。只用已经出现过的目录、文件和决定，不要编路径，也不要留 A/B/C 让工人挑。title 只在 accept 时写一个短名。

continue 沿用已有工单。先看材料里这一张的阶段，再决定 brief 写多长：
- 正在做，人只是改一个决定或补一句：只写变化。不要重述背景、路径和完成标准。
- 在等回答：点明它回答的是哪个问题，再按人的意思写答案。人的原话会一起送去，不用逐字抄。
- 停着，或在等验收而人要改，目录和上一次说明还在：写从哪停下、要改哪里、下一步做什么。不重写整张工单。
- 没有目录、说明已经对不上、或人把目标换成了另一件事：才按 accept 那样写完整说明。
continue 的 title 留空。brief 为空会被拒绝。

对人只说一两句：已经交给哪张工单、这一轮要做成什么。不要把 brief 贴进对话，也不要等工人做完。

推断的工作只有在 auto 下、并且 rules 里写了至少一条还启用的规则编号时才能接受。只是聊天就忽略。人明确要求停时才用 task_stop。人说先停一下、等等，mode 填 pause，只停正在跑的这一轮，工单留着；人说不用做了、取消，才结束这张工单。工单在等验收：人说可以了、通过，mode 填 done；人说不行、要改，用 task_route continue，把要改的地方写进 brief。工人的阶段性进展和结果会写回这段对话，你不要重做。`

// errNeedsBrief 在新开一张工单却没写任务说明时返回。人的原话不会代替这份说明。
var errNeedsBrief = fmt.Errorf("需要 brief。新开的工单要写清背景、目标产物、已经知道的路径、硬性约束和怎样算做完。人的原话不会另送")

// errNeedsHandoff 在继续一张已有工单却什么都没交代时返回。交代可以很短，但不能空，也不能用原话顶上。
var errNeedsHandoff = fmt.Errorf("继续这张工单需要给工人一句交代。正在做就只写变化，在等回答就只写答案，上下文断了再写完整说明。不要把人的原话原样送去")

// beginRoute 在智能体主会话上换成路由工具。视频、学习、短片和即时消息会话保持原来的工具。
func (a *App) beginRoute(tools *runtime.WorkspaceTools, meta SessionMeta, sessionID string) bool {
	if a == nil || tools == nil || !a.routeSession(meta, sessionID) {
		return false
	}
	tools.RouteOnly = true
	tools.Extra = a.routerExtrasFor(sessionID)
	return true
}

// routeSession 在这条会话该由路由接手时为真。
func (a *App) routeSession(meta SessionMeta, sessionID string) bool {
	if session.NormalizeChannel(meta.Channel) != session.ChannelAgent {
		return false
	}
	if meta.ProfileID == "im" {
		return false
	}
	if a.studioLane(sessionID) {
		return false
	}
	return true
}

// routeFragment 是写进主会话的路由说明。
func routeFragment() artifact.PromptFragment {
	return artifact.PromptFragment{Slot: "role", Text: routePrompt}
}

// intakeDesk 把这一轮桌面消息记成明确触发。已经落到同一条具体线程上的，告诉路由只能停止。
// 主会话上的新请求先列出最近工单，由路由判断是继续原目录，还是新开。
func (a *App) intakeDesk(sessionID, turnID, text string) string {
	text = strings.TrimSpace(text)
	if a == nil || text == "" || strings.TrimSpace(turnID) == "" {
		return ""
	}
	a.heard(sessionID, text)
	rt := a.orchestrator()
	if rt == nil {
		return ""
	}
	id := "desk:" + sessionID + ":" + turnID
	_, reached := rt.Receive(superagent.Trigger{
		ID: id, Source: "desk", Threads: []string{"desk:" + sessionID}, Primary: "desk:" + sessionID,
		Human: true, Addressed: true, Sender: "operator", Text: text, Channel: "desk",
	})
	if len(reached) > 0 {
		return "这条消息已经交给工单 " + strings.Join(reached, ", ") + "。不要再接受。只有人明确要求停止时才用 task_stop。"
	}
	line := "这条桌面消息的触发编号是 " + id + "。先看最近的工单和它现在的阶段。同一件事或明显相关就用 task_route，action 用 continue，ticket 填那张工单，不要新开目录。都不相关才 accept。accept 要写完整 brief。continue 的 brief 看阶段：正在做就只写变化，在等回答就只写答案，停着就写下一步，上下文断了才写完整说明。人的原话不要原样送去，brief 留空会被拒绝。title 只在 accept 时写短名，continue 留空。只是聊天就忽略。不要改文件，也不要等工人做完。"
	if roster := taskRoster(rt); roster != "" {
		line += "\n最近工单：\n" + roster
	}
	if rules := enabledRuleLine(rt); rules != "" {
		line += " 推断接受时 rules 只能从这些编号里选：" + rules
	}
	return line
}

// enabledRuleLine 把还启用的规则编号拼成一行。
func enabledRuleLine(rt *superagent.Runtime) string {
	if rt == nil {
		return ""
	}
	var ids []string
	for _, rule := range rt.View().Rules {
		if rule.Enabled && strings.TrimSpace(rule.ID) != "" {
			ids = append(ids, rule.ID)
		}
	}
	return strings.Join(ids, ", ")
}

// ruleParam 从一项路由参数里取出规则编号。字符串和数组都可以。
func ruleParam(raw any) []string {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	case []any:
		var out []string
		for _, item := range v {
			text, ok := item.(string)
			if ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	default:
		return nil
	}
}

// routerExtras 是不属于哪条会话时的七件路由工具。记下的记忆都算路由的推断。
func (a *App) routerExtras() map[string]runtime.ExtraTool {
	return a.routerExtrasFor("")
}

// routerExtrasFor 是主会话仅有的七件工具。写记忆时拿 sessionID 这条会话里当事人最近写的话核对出处。
func (a *App) routerExtrasFor(sessionID string) map[string]runtime.ExtraTool {
	return map[string]runtime.ExtraTool{
		"task_route":    a.routeTool(),
		"task_status":   a.statusTool(),
		"task_note":     a.noteTool(),
		"task_stop":     a.stopTool(),
		"memory_read":   a.memoryTool(),
		"memory_write":  a.memoryWriteTool(sessionID),
		"memory_forget": a.memoryForgetTool(),
	}
}

// heardMax 是每条会话留几句当事人的话，heardRunes 是每句留多长。
const (
	heardMax   = 3
	heardRunes = 2000
)

// heard 记下当事人在 sessionID 里亲手写的一句话，只留最近 heardMax 句。
func (a *App) heard(sessionID, text string) {
	if a == nil || sessionID == "" || strings.TrimSpace(text) == "" {
		return
	}
	if r := []rune(text); len(r) > heardRunes {
		text = string(r[:heardRunes])
	}
	a.wordsMu.Lock()
	defer a.wordsMu.Unlock()
	if a.words == nil {
		a.words = map[string][]string{}
	}
	lines := append(a.words[sessionID], text)
	if len(lines) > heardMax {
		lines = lines[len(lines)-heardMax:]
	}
	a.words[sessionID] = lines
}

// heardIn 是当事人在 sessionID 里最近写的几句话。
func (a *App) heardIn(sessionID string) []string {
	if a == nil || sessionID == "" {
		return nil
	}
	a.wordsMu.Lock()
	defer a.wordsMu.Unlock()
	return append([]string(nil), a.words[sessionID]...)
}

// routeTool 接受、忽略、停止或挂接触发。一次可以有多项。
func (a *App) routeTool() runtime.ExtraTool {
	return routeExtra("task_route", "路由一条或多项触发。items 里每项有 trigger、action、ticket、title、brief、reason、rules。action 取 accept、continue、ignore、stop、attach。同一件事用 continue 并填 ticket。brief 不能空，人的原话不会另送。accept 要写完整说明。continue 按工单阶段写：正在做只写变化，在等回答只写答案，停着写下一步，上下文断了才写完整说明。title 只在 accept 时作为短名。推断接受时 rules 是启用的规则编号。", map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"trigger": map[string]any{"type": "string"},
					"action":  map[string]any{"type": "string", "enum": []string{"accept", "continue", "ignore", "stop", "attach"}},
					"ticket":  map[string]any{"type": "string"},
					"title":   map[string]any{"type": "string", "description": "只在 accept 时填写的短名。不要写成任务说明。"},
					"brief":   map[string]any{"type": "string", "description": "交给工人的交代。accept 写完整说明。continue 按阶段伸缩：正在做只写变化，在等回答只写答案，停着写下一步，上下文断了才写完整说明。不要把人的原话原样放进来。留空会被拒绝。"},
					"reason":  map[string]any{"type": "string"},
					"rules":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"trigger", "action"},
			},
		},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		raw, _ := json.Marshal(args["items"])
		var items []map[string]any
		if json.Unmarshal(raw, &items) != nil || len(items) == 0 {
			items = []map[string]any{args}
		}
		var b strings.Builder
		for _, item := range items {
			line, err := a.routeOne(rt, item)
			if err != nil {
				fmt.Fprintf(&b, "%s\n", err.Error())
				continue
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		return strings.TrimSpace(b.String()), nil
	})
}

// routeOne 执行一项路由。rt 是编排运行时。
func (a *App) routeOne(rt *superagent.Runtime, item map[string]any) (string, error) {
	trigger := strParam(item, "trigger")
	action := strParam(item, "action")
	title := strParam(item, "title")
	reason := strParam(item, "reason")
	ticket := strParam(item, "ticket")
	if reason == "" {
		reason = title
	}
	brief := strings.TrimSpace(strParam(item, "brief"))
	switch action {
	case "accept":
		if brief == "" {
			return "", errNeedsBrief
		}
		if ticket != "" {
			if err := rt.Act("assign", ticket, trigger, brief, ""); err != nil {
				return "", err
			}
			return "attached " + trigger + " to " + ticket, nil
		}
		rules := ruleParam(item["rules"])
		if err := rt.DecideBrief(trigger, "accept", title, brief, rules); err != nil {
			return "", err
		}
		return "accepted " + trigger, nil
	case "continue", "attach":
		if ticket == "" {
			return "", fmt.Errorf("continue needs ticket")
		}
		if brief == "" {
			return "", errNeedsHandoff
		}
		// 工人在等一个答案时，人的原话就是答案，路径和数值转述会走样。原话和交代一起送过去。
		kind := "assign"
		if waitsForAnswer(rt, ticket) {
			kind = "attach"
		}
		if err := rt.Act(kind, ticket, trigger, brief, ""); err != nil {
			return "", err
		}
		return "continued " + ticket, nil
	case "ignore":
		if err := rt.Decide(trigger, "ignore", reason); err != nil {
			return "", err
		}
		return "ignored " + trigger, nil
	case "stop":
		if ticket != "" {
			if err := rt.Act("stop", ticket, "", "", reason); err != nil {
				return "", err
			}
			return "stopped " + ticket, nil
		}
		if err := rt.Decide(trigger, "stop", reason); err != nil {
			return "", err
		}
		return "stopped " + trigger, nil
	default:
		return "", fmt.Errorf("bad action %s", action)
	}
}

// waitsForAnswer 在 ticketID 停在等人回答一个问题时为真。
func waitsForAnswer(rt *superagent.Runtime, ticketID string) bool {
	if rt == nil {
		return false
	}
	for _, row := range rt.View().Tickets {
		if row.ID == ticketID {
			return row.Stage != "ended" && row.Action == "answer"
		}
	}
	return false
}

// statusTool 列出最近的工单，包含已经停下或做完的。
func (a *App) statusTool() runtime.ExtraTool {
	return routeExtra("task_status", "列出最近的工单，包含已经停下或做完的。没有参数。", map[string]any{}, true, func(map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		text := taskRoster(rt)
		if text == "" {
			return "没有工单", nil
		}
		return text, nil
	})
}

// taskRoster 把最近八张工单收成路由能对照的几行。新的在前。
func taskRoster(rt *superagent.Runtime) string {
	if rt == nil {
		return ""
	}
	rows := append([]superagent.TicketRow(nil), rt.View().Tickets...)
	sort.Slice(rows, func(i, j int) bool {
		return ticketNum(rows[i].ID) > ticketNum(rows[j].ID)
	})
	if len(rows) > 8 {
		rows = rows[:8]
	}
	var b strings.Builder
	for _, ticket := range rows {
		title := strings.TrimSpace(ticket.Title)
		if title == "" {
			title = "未命名"
		}
		fmt.Fprintf(&b, "%s %s %s\n", ticket.ID, ticketSituation(ticket), title)
		if dir := strings.TrimSpace(ticket.Directory); dir != "" {
			fmt.Fprintf(&b, "  目录 %s\n", dir)
		}
		if ask := strings.TrimSpace(ticket.Attention); ask != "" {
			fmt.Fprintf(&b, "  在等回答 %s\n", clipBrief(ask, 160))
		}
		if summary := lastTicketSummary(ticket); summary != "" {
			fmt.Fprintf(&b, "  最近 %s\n", clipBrief(summary, 180))
		}
	}
	return strings.TrimSpace(b.String())
}

// ticketSituation 用阶段告诉路由该写多长的交代。
func ticketSituation(ticket superagent.TicketRow) string {
	if ticket.Stage == "ended" {
		return ticketState(ticket)
	}
	switch ticket.Action {
	case "answer":
		return "在等回答"
	case "review":
		return "等你验收"
	case "blocked", "failed":
		return "卡住了"
	case "delivery":
		return "结果没送出"
	case "queued":
		return "还没开始"
	}
	switch ticket.Stage {
	case "working":
		return "正在做"
	case "idle":
		return "停着"
	case "person":
		return "在等人"
	default:
		return ticketState(ticket)
	}
}

// lastTicketSummary 取最近一次派工留给人看的那句。
func lastTicketSummary(ticket superagent.TicketRow) string {
	for i := len(ticket.Assignments) - 1; i >= 0; i-- {
		if text := strings.TrimSpace(ticket.Assignments[i].Summary); text != "" {
			return text
		}
	}
	return ""
}

func ticketNum(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "tkt-"))
	if err != nil {
		return 0
	}
	return n
}

func ticketState(ticket superagent.TicketRow) string {
	if ticket.Stage != "ended" {
		return "进行中"
	}
	switch ticket.Outcome {
	case "stopped":
		return "已打断"
	case "failed", "dropped":
		return "没做成"
	default:
		return "做完了"
	}
}

// noteTool 给工单记一条内部备注。
func (a *App) noteTool() runtime.ExtraTool {
	return routeExtra("task_note", "给 ticket 记一条内部备注 text。这不会发给人。", map[string]any{
		"ticket": map[string]any{"type": "string"},
		"text":   map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		if err := rt.Act("note", strParam(args, "ticket"), "", strParam(args, "text"), ""); err != nil {
			return "", err
		}
		return "noted", nil
	})
}

// stopTool 停下或收掉一张工单。mode 为 pause 时只停正在跑的这一轮；done 时人验收通过；其余结束工单。
func (a *App) stopTool() runtime.ExtraTool {
	return routeExtra("task_stop", "停下或收掉 ticket。mode 取 pause：只停正在跑的这一轮，工单留着，人再说一句就接着做；done：工单在等验收，人说通过，记成做完了；stop 或不填：结束这张工单，记成已打断。reason 写给人看的原因。", map[string]any{
		"ticket": map[string]any{"type": "string"},
		"mode":   map[string]any{"type": "string", "enum": []string{"stop", "pause", "done"}},
		"reason": map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		ticket := strParam(args, "ticket")
		switch strParam(args, "mode") {
		case "pause":
			if err := rt.Act("pause", ticket, "", "", ""); err != nil {
				if err.Error() == "notRunning" {
					return ticket + " 现在没有在跑，没有要停的这一轮。", nil
				}
				return "", err
			}
			return "paused " + ticket, nil
		case "done":
			if err := rt.Act("accept", ticket, "", "", strParam(args, "reason")); err != nil {
				if err.Error() == "notReview" {
					return ticket + " 现在不在等验收。还在做就等它报做完；要它改就用 task_route continue。", nil
				}
				return "", err
			}
			return "accepted " + ticket, nil
		}
		reason := strParam(args, "reason")
		if reason == "" {
			reason = "stopped"
		}
		if err := rt.Act("stop", ticket, "", "", reason); err != nil {
			return "", err
		}
		return "stopped", nil
	})
}

// memoryTool 按词查找记忆。每行是编号、种类、最近记下的日子和正文，忘掉或改口时用这个编号。过去的安排标出来。
func (a *App) memoryTool() runtime.ExtraTool {
	return routeExtra("memory_read", "按 query 查找已经留下的记忆。每行是编号、种类、最近记下的日子和正文；已经过去的安排会标出来。", map[string]any{
		"query": map[string]any{"type": "string"},
	}, true, func(args map[string]any) (string, error) {
		if a.Memory == nil {
			return "没有记忆", nil
		}
		items := a.Memory.Search(strParam(args, "query"), memory.Kind(""), 8)
		if len(items) == 0 {
			return "没有匹配的记忆", nil
		}
		now := time.Now()
		var b strings.Builder
		for _, item := range items {
			past := ""
			if memory.Expired(item, now) {
				past = "（已经过去）"
			}
			fmt.Fprintf(&b, "%s %s %s %s%s\n", item.ID, item.Kind, memoryDay(item), item.Text, past)
		}
		return strings.TrimSpace(b.String()), nil
	})
}

// inferredSource 是路由记下、却不出自当事人原话的记忆的来源。界面上是「对话里记下」，工人看不到。
const inferredSource = "Inferred in chat"

// memoryWriteTool 把人说出的事实或偏好留下。改口时换掉旧的那一条。
// 正文大半出自当事人在这条会话里最近写的话时，来源记成本人确认，界面上就是「你说的」；
// 否则是路由的转述或推断，可能来自工人报告这类别处的内容，记成 inferredSource。
func (a *App) memoryWriteTool(sessionID string) runtime.ExtraTool {
	return routeExtra("memory_write", "记下人说出的一条长期事实或偏好。kind 取 profile（关于这个人）、semantic（偏好和写法）或 project（在做的项目）。text 用人的原意写成一句；带日子的照原话写，「明天」「下周三」会换成日期。人改口时 replaces 填旧的那一条的编号。不记推断、一次性细节和密钥。", map[string]any{
		"kind":     map[string]any{"type": "string", "enum": []string{"profile", "semantic", "project"}},
		"text":     map[string]any{"type": "string"},
		"replaces": map[string]any{"type": "string", "description": "人改口时，被换掉的那条旧记忆的编号。"},
	}, false, func(args map[string]any) (string, error) {
		if a.Memory == nil {
			return "", fmt.Errorf("没有记忆库")
		}
		kind := memory.Kind(strParam(args, "kind"))
		switch kind {
		case memory.KindProfile, memory.KindSemantic, memory.KindProject:
		default:
			kind = memory.KindProfile
		}
		text := strParam(args, "text")
		source := "User confirmed in chat"
		if !memory.Grounded(text, a.heardIn(sessionID)) {
			source = inferredSource
		}
		origin := ""
		if sessionID != "" {
			origin = "session:" + sessionID
		}
		kept := a.Memory.Keep(memory.Item{Kind: kind, Text: text, Source: source, Origin: origin}, strParam(args, "replaces"))
		if kept.Action != "refused" {
			a.touchPoi()
		}
		return memoryReceipt(kept), nil
	})
}

// memoryReceipt 告诉路由这一次写成了什么：新记下、已经记着、改了哪一句，以及可能被这一句推翻的旧记忆。
func memoryReceipt(kept memory.Kept) string {
	it := kept.Item
	var b strings.Builder
	switch {
	case kept.Action == "refused":
		return "没有记：正文是空的、像密钥，或者 replaces 指的编号不存在"
	case kept.Action == "same" || (kept.Action == "merged" && kept.Before == it.Text):
		b.WriteString("已经记着 " + it.ID + "：" + it.Text)
	case kept.Action == "merged" || kept.Action == "replaced":
		b.WriteString("改好了 " + it.ID + "：" + kept.Before + " → " + it.Text)
	default:
		b.WriteString("记下了 " + it.ID + "：" + it.Text)
	}
	if it.ValidUntil != "" {
		b.WriteString("（到 " + it.ValidUntil + " 为止，过了就当成过去的事）")
	}
	if it.Source == inferredSource {
		b.WriteString("\n这句不是人的原话，记成了你的推断，工人看不到。要记成人说的，就照人的原话写。")
	}
	if vague := memory.Vague(it.Text); len(vague) > 0 {
		b.WriteString("\n「" + strings.Join(vague, "」「") + "」换算不成日子。人说了具体哪天，就写具体哪天。")
	}
	if len(kept.Related) > 0 {
		b.WriteString("\n相近的旧记忆。新的一句要是推翻了其中一条，再调一次 memory_write，把 replaces 填成那一条的编号，或者用 memory_forget 忘掉它；说的是两件事就不用管：")
		for _, old := range kept.Related {
			b.WriteString("\n" + old.ID + " " + old.Text)
		}
	}
	return b.String()
}

// memoryForgetTool 按编号忘掉一条。本机清点每一轮都会重写，不从这里忘。
func (a *App) memoryForgetTool() runtime.ExtraTool {
	return routeExtra("memory_forget", "按 id 忘掉一条记忆。id 来自 memory_read。本机清点出来的条目不能从这里忘。", map[string]any{
		"id": map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		id := strParam(args, "id")
		if strings.HasPrefix(id, "machine:") {
			return "这是本机清点出来的，每一轮都会重写，不从这里忘", nil
		}
		if !a.MemoryForget(id) {
			return "", fmt.Errorf("没有编号为 %s 的记忆", id)
		}
		return "忘掉了 " + id, nil
	})
}

// routeExtra 组装一件路由工具。readOnly 为真时不要求独占。
func routeExtra(name, desc string, props map[string]any, readOnly bool, fn func(map[string]any) (string, error)) runtime.ExtraTool {
	return runtime.ExtraTool{
		ReadOnly:  readOnly,
		Exclusive: !readOnly,
		JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
			"name":        name,
			"description": desc,
			"parameters": map[string]any{
				"type":       "object",
				"properties": props,
			},
		}},
		Call: func(argsJSON string) runtime.ToolResult {
			var args map[string]any
			_ = json.Unmarshal([]byte(argsJSON), &args)
			text, err := fn(args)
			if err != nil {
				return runtime.ToolResult{Content: err.Error(), Err: err}
			}
			return runtime.ToolResult{Content: text}
		},
	}
}

// mountWorker 把工作区工具登记到这张工单的代理上。会话授权是 full，避免每条命令都问人。
func (a *App) mountWorker(reg *dsh.Registry, agent *dsh.Agent, dir, ticketID string) (func() int, func(time.Duration)) {
	id := "ticket-" + ticketID
	if a != nil {
		a.applySessionAuth(id, capability.AuthFull)
	}
	home := ""
	var caps *capability.Broker
	if a != nil {
		caps = a.Caps
		if a.Home != nil {
			home = a.Home.Root
		}
	}
	var roots []string
	if a != nil {
		roots = a.namedWorkRoots(ticketID, home)
	}
	tools := &runtime.WorkspaceTools{Workspace: dir, Home: home, SessionID: id, Caps: caps, Roots: roots}
	return dshdesk.MountWorkspace(reg, agent, tools)
}

// namedWorkRoots 从这张工单已经交给工人的说明里取出可写目录。家目录和过浅的路径不放进来。
func (a *App) namedWorkRoots(ticketID, home string) []string {
	if a == nil {
		return nil
	}
	rt := a.orchestrator()
	if rt == nil {
		return nil
	}
	var b strings.Builder
	for _, text := range rt.TicketTexts(ticketID) {
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return runtime.NamedRoots(b.String(), home)
}

// relayNotice 把一条通知写进指定的桌面会话。它不开始新的路由回合。
func (a *App) relayNotice(sessionID, text string) {
	if a == nil || a.Traces == nil || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(text) == "" {
		return
	}
	// 界面按 id 合并同一条助手消息。每条通知要有自己的 id，否则后一条会把前一条盖掉。
	now := time.Now().UTC()
	sum := sha256.Sum256([]byte(sessionID + "\n" + text + "\n" + now.Format(time.RFC3339Nano)))
	ev := trace.Event{
		TS:        now,
		Type:      trace.TypeAssistant,
		Source:    "notice",
		SessionID: sessionID,
		Payload:   map[string]any{"text": strings.TrimSpace(text), "id": "notice:" + sessionID + ":" + hex.EncodeToString(sum[:6])},
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
}

// relayReport 把需要人看的报告写进正在看这张工单的桌面会话。
// 做完、失败、放下和打断会带上已经报过的阶段性进展，并写进记忆。会话写不进去时，记忆仍然留下。
// 同一段正文只写一次。后来跟上的会话优先，没有时才回到开工那一条。
func (a *App) relayReport(ticketID, state, summary, question string) {
	if a == nil {
		return
	}
	// 进行中只留在任务页。主会话只收结束、终止，以及要人接手的报告。
	if strings.TrimSpace(question) == "" && state == "working" {
		return
	}
	text := strings.TrimSpace(summary)
	if q := strings.TrimSpace(question); q != "" && !strings.Contains(text, q) {
		if text != "" {
			text += "\n\n"
		}
		text += q
	}
	if taskEndedState(state) {
		if rt := a.orchestrator(); rt != nil {
			extra := strings.TrimSpace(rt.StagedProgress(ticketID))
			if extra != "" && !strings.Contains(text, extra) {
				if text != "" {
					text = extra + "\n" + text
				} else {
					text = extra
				}
			}
		}
		a.rememberTask(ticketID, state, text)
	}
	if a.Traces == nil || text == "" {
		return
	}
	rt := a.orchestrator()
	if rt == nil {
		return
	}
	sids := rt.DeskAudiences(ticketID)
	if len(sids) == 0 {
		// GitHub、飞书、邮件来的工单没有跟着的桌面会话。要人看的报告仍然回到主会话，不然只能去任务页翻。
		if meta, ok := a.savedMain(); ok && meta.ID != "" {
			sids = []string{meta.ID}
		}
	}
	if len(sids) == 0 {
		return
	}
	sum := sha256.Sum256([]byte(ticketID + "\n" + state + "\n" + text))
	id := "handoff:" + ticketID + ":" + hex.EncodeToString(sum[:8])
	kind := "handoff"
	if strings.TrimSpace(question) == "" && (state == "ready" || state == "waiting" || state == "stopped") {
		kind = "status"
	}
	for _, sid := range sids {
		if sid == "" || a.traceHasID(sid, id) {
			continue
		}
		ev := trace.Event{
			Type:      trace.TypeAssistant,
			Source:    "worker",
			SessionID: sid,
			Payload: map[string]any{
				"text": text, "id": id, "kind": kind, "ticket": ticketID, "state": state,
			},
		}
		if err := a.Traces.AppendDurable(ev); err != nil {
			continue
		}
		if a.Hub != nil {
			a.Hub.Publish(ev)
		}
	}
}

// handoffAsk 从还等人的工单上取出最后一段该给人看的报告。没有正文时不算。
func handoffAsk(row superagent.TicketRow) (state, summary, question string, ok bool) {
	if !row.Person || row.Stage == "ended" {
		return "", "", "", false
	}
	var line superagent.ReportLine
	found := false
	for _, asg := range row.Assignments {
		for _, ln := range asg.Lines {
			line = ln
			found = true
		}
	}
	state = row.Action
	if found {
		summary = line.Summary
		question = line.Question
		if line.State != "" {
			state = line.State
		}
	}
	if strings.TrimSpace(question) == "" {
		question = row.Attention
	}
	if strings.TrimSpace(summary) == "" && strings.TrimSpace(question) == "" {
		return "", "", "", false
	}
	if state == "" {
		state = "blocked"
	}
	return state, summary, question, true
}

// taskEndedState 在这份结果该留下记忆时为真。卡住和普通进度都不算结束。
func taskEndedState(state string) bool {
	switch state {
	case "ready", "failed", "dropped", "stopped":
		return true
	default:
		return false
	}
}

// rememberTask 把工单的结果写成一条情景。同一张工单再写一次会换掉原来的。
func (a *App) rememberTask(ticketID, state, text string) {
	if a == nil || a.Memory == nil || strings.TrimSpace(ticketID) == "" {
		return
	}
	label := ticketID
	if rt := a.orchestrator(); rt != nil {
		if name := rt.TicketLabel(ticketID); name != "" {
			label = name
		}
	}
	line := label
	body := strings.TrimSpace(text)
	if body != "" && body != label {
		line = label + "：" + body
	}
	switch state {
	case "failed":
		line = "没做成。" + line
	case "dropped":
		line = "放下了。" + line
	case "stopped":
		line = "停下了。" + line
	}
	a.Memory.Write(memory.Item{
		ID:     "task:" + ticketID,
		Kind:   memory.KindEpisodic,
		Text:   clipTaskMemory(line),
		Source: "worker",
		Origin: "ticket:" + ticketID,
		State:  state,
	})
}

// clipTaskMemory 把记忆收成最多四百个字。
func clipTaskMemory(text string) string {
	r := []rune(strings.TrimSpace(text))
	if len(r) > 400 {
		return string(r[:400])
	}
	return string(r)
}

// connectorTriggers 把邮件和日历收成推断触发。没有真实正文的占位行不收。
func (a *App) connectorTriggers() []superagent.Trigger {
	if a == nil || a.Connectors == nil {
		return nil
	}
	var out []superagent.Trigger
	for _, acct := range a.Connectors.List() {
		if acct.Kind != connector.KindMail && acct.Kind != connector.KindCalendar {
			continue
		}
		rows, err := a.Connectors.Read(acct.ID, "")
		if err != nil {
			continue
		}
		n := 0
		for _, row := range rows {
			if n >= 8 || strings.Contains(row["note"], "no live token") {
				continue
			}
			itemID := strings.TrimSpace(row["id"])
			subject := strings.TrimSpace(row["subject"])
			if subject == "" {
				subject = strings.TrimSpace(row["title"])
			}
			body := strings.TrimSpace(row["body"])
			if itemID == "" || (subject == "" && body == "") {
				continue
			}
			text := subject
			if body != "" {
				if text != "" {
					text += "\n"
				}
				text += body
			}
			source := string(acct.Kind)
			thread := source + ":" + acct.ID
			due := strings.TrimSpace(row["start"])
			if due == "" {
				due = strings.TrimSpace(row["due"])
			}
			out = append(out, superagent.Trigger{
				ID: source + ":" + acct.ID + ":" + itemID, Source: source,
				Threads: []string{thread}, Primary: thread, Human: true,
				Addressed: strings.Contains(text, "@"), Sender: acct.Label, Text: text, Channel: source, Due: due,
			})
			n++
		}
	}
	return out
}

// orchestratorPrepare 在执行根里跑准备命令。env 原样追加，调用方不要放进钥匙。
func orchestratorPrepare(root string) func(ctx context.Context, work, command string, env map[string]string) (string, error) {
	return func(ctx context.Context, work, command string, env map[string]string) (string, error) {
		command = strings.TrimSpace(command)
		if command == "" {
			return "", nil
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		var cmd *exec.Cmd
		if goruntime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		if strings.TrimSpace(work) == "" {
			work = root
		}
		if work != "" {
			cmd.Dir = work
		}
		cmd.Env = os.Environ()
		for key, value := range env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
		out, err := cmd.CombinedOutput()
		text := string(out)
		if len(text) > 8000 {
			text = text[:8000]
		}
		return text, err
	}
}
