// route.go 把主会话收成路由。它只接单、看状态、记备注、停止和读记忆。
// 不拥有工人的工作区。工人在工单目录里用自己的工具。视频、学习和短片会话不走这里。

package app

import (
	"context"
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

const routePrompt = "你是主会话的路由。你只有 task_route、task_status、task_note、task_stop 和 memory_read，不能改文件，也不能跑命令。明确的请求立刻路由。先看材料里列出的工单：和某一张是同一件事，或者明显相关，就用 continue 指到那张工单，沿用它的目录和会话，不要 accept 新开。有多张相关时，继续最贴近的一张。都不相关才 accept，并给出 title，不要把范围展开成工作计划。推断的工作只有在 auto 下、并且 rules 里写了至少一条还启用的规则编号时才能接受。接受或继续之后用一句话告诉人已经开始，然后结束，不要等工人。只是聊天就忽略。人明确要求打断时才 stop。打断只停这一轮，原会话还能继续。工人的阶段性进展和结果会写回这段对话，你不要重做。"

// beginRoute 在智能体主会话上换成路由工具。视频、学习、短片和即时消息会话保持原来的工具。
func (a *App) beginRoute(tools *runtime.WorkspaceTools, meta SessionMeta, sessionID string) bool {
	if a == nil || tools == nil || !a.routeSession(meta, sessionID) {
		return false
	}
	tools.RouteOnly = true
	tools.Extra = a.routerExtras()
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
	line := "这条桌面消息的触发编号是 " + id + "。先看最近的工单。同一件事或明显相关就用 task_route，action 用 continue，ticket 填那张工单，不要新开目录。都不相关才 accept，并给出 title。只是聊天就忽略。不要改文件，也不要等工人做完。"
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

// routerExtras 是主会话仅有的五件工具。
func (a *App) routerExtras() map[string]runtime.ExtraTool {
	return map[string]runtime.ExtraTool{
		"task_route":  a.routeTool(),
		"task_status": a.statusTool(),
		"task_note":   a.noteTool(),
		"task_stop":   a.stopTool(),
		"memory_read": a.memoryTool(),
	}
}

// routeTool 接受、忽略、停止或挂接触发。一次可以有多项。
func (a *App) routeTool() runtime.ExtraTool {
	return routeExtra("task_route", "路由一条或多项触发。items 里每项有 trigger、action、title、reason、ticket、rules。action 取 accept、continue、ignore、stop、attach。同一件事用 continue 并填 ticket。推断接受时 rules 是启用的规则编号。", map[string]any{
		"items": map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
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
	switch action {
	case "accept":
		if ticket != "" {
			if err := rt.Act("attach", ticket, trigger, title, ""); err != nil {
				return "", err
			}
			return "attached " + trigger + " to " + ticket, nil
		}
		rules := ruleParam(item["rules"])
		var err error
		if len(rules) > 0 {
			err = rt.DecideMatched(trigger, "accept", title, rules)
		} else {
			err = rt.Decide(trigger, "accept", title)
		}
		if err != nil {
			return "", err
		}
		return "accepted " + trigger, nil
	case "continue", "attach":
		if ticket == "" {
			return "", fmt.Errorf("continue needs ticket")
		}
		if err := rt.Act("attach", ticket, trigger, title, ""); err != nil {
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
		fmt.Fprintf(&b, "%s %s %s\n", ticket.ID, ticketState(ticket), title)
	}
	return strings.TrimSpace(b.String())
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

// stopTool 停止一张工单。
func (a *App) stopTool() runtime.ExtraTool {
	return routeExtra("task_stop", "停止 ticket。reason 写给人看的原因。", map[string]any{
		"ticket": map[string]any{"type": "string"},
		"reason": map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		reason := strParam(args, "reason")
		if reason == "" {
			reason = "stopped"
		}
		if err := rt.Act("stop", strParam(args, "ticket"), "", "", reason); err != nil {
			return "", err
		}
		return "stopped", nil
	})
}

// memoryTool 只读记忆。
func (a *App) memoryTool() runtime.ExtraTool {
	return routeExtra("memory_read", "按 query 查找已经留下的记忆。不能写入。", map[string]any{
		"query": map[string]any{"type": "string"},
	}, true, func(args map[string]any) (string, error) {
		if a.Memory == nil {
			return "没有记忆", nil
		}
		items := a.Memory.Search(strParam(args, "query"), memory.Kind(""), 8)
		if len(items) == 0 {
			return "没有匹配的记忆", nil
		}
		var b strings.Builder
		for _, item := range items {
			fmt.Fprintf(&b, "%s %s\n", item.Kind, item.Text)
		}
		return strings.TrimSpace(b.String()), nil
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
	tools := &runtime.WorkspaceTools{Workspace: dir, Home: home, SessionID: id, Caps: caps}
	return dshdesk.MountWorkspace(reg, agent, tools)
}

// relayNotice 把一条通知写进指定的桌面会话。它不开始新的路由回合。
func (a *App) relayNotice(sessionID, text string) {
	if a == nil || a.Traces == nil || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(text) == "" {
		return
	}
	ev := trace.Event{
		Type:      trace.TypeAssistant,
		Source:    "notice",
		SessionID: sessionID,
		Payload:   map[string]any{"text": strings.TrimSpace(text), "id": "notice:" + sessionID},
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
}

// relayReport 把需要人看的报告写进开工时的那条桌面会话。
// 做完、失败、放下和打断会带上已经报过的阶段性进展，并写进记忆。会话写不进去时，记忆仍然留下。
func (a *App) relayReport(ticketID, state, summary, question string) {
	if a == nil {
		return
	}
	text := strings.TrimSpace(summary)
	if strings.TrimSpace(question) != "" {
		if text != "" {
			text += "\n"
		}
		text += strings.TrimSpace(question)
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
	sid := rt.DeskSession(ticketID)
	if sid == "" {
		return
	}
	ev := trace.Event{
		Type:      trace.TypeAssistant,
		Source:    "worker",
		SessionID: sid,
		Payload:   map[string]any{"text": text, "id": ticketID + ":" + state + ":" + fmt.Sprintf("%d", time.Now().UnixNano())},
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
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
