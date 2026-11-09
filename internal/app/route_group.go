// route_group.go 拥有路由这一侧的任务组工具：task_split 把一条触发开成任务组，task_recommend 记下对比推荐哪一份，
// task_pick 记下对比采用哪一份，task_read 读一张工单的详情；以及 task_split 之后写进主会话的任务组卡片。
// 不拥有任务组的状态、放行和唤醒（superagent 的 group.go 和 group_runtime.go），也不拥有唤醒那一轮（route_wake.go）。

package app

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// splitTool 把一条触发开成任务组。sessionID 是路由所在的会话，任务组有了结果时唤醒它。
func (a *App) splitTool(sessionID string) runtime.ExtraTool {
	return routeExtra("task_split", "把一条触发开成任务组，交给几个 agent 一起做。mode 取 split 或 compare。split：parts 填 2 到 6 份，每份有 title、agent、role、brief、depends_on；brief 要能独立读懂，写清交付物和放在哪；几份会改同一个目录时划开互不重叠的区域，做不到就用 depends_on 让后面的等前面的。compare：人要几个 agent 分别做同一件事再比较时用，agents 填 2 到 4 个不同的 agent，brief 写同一份完整说明，不能有 depends_on。agent 能填什么和 task_route 一样。title 是任务组的短名。推断接受时 rules 是启用的规则编号。", map[string]any{
		"trigger": map[string]any{"type": "string"},
		"mode":    map[string]any{"type": "string", "enum": []string{superagent.GroupSplit, superagent.GroupCompare}},
		"title":   map[string]any{"type": "string", "description": "任务组的短名。"},
		"reason":  map[string]any{"type": "string"},
		"rules":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"brief":   map[string]any{"type": "string", "description": "compare 时交给每个 agent 的同一份完整说明。split 不用填，写在每一份里。"},
		"agents":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "compare 时做同一件事的 2 到 4 个不同 agent。"},
		"parts": map[string]any{
			"type":        "array",
			"description": "split 时的 2 到 6 份，有先后的按先后排。",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":      map[string]any{"type": "string", "description": "这一份的短名。"},
					"agent":      map[string]any{"type": "string"},
					"role":       map[string]any{"type": "string", "description": "这一份的分工，例如后端、前端、测试。"},
					"brief":      map[string]any{"type": "string", "description": "交给这一份的完整说明。"},
					"depends_on": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "要等哪几份做完才开工，填它们在 parts 里的序号，从 0 开始。"},
				},
				"required": []string{"agent", "brief"},
			},
		},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		return a.splitOne(rt, sessionID, args)
	})
}

// splitOne 校验 args 里的每一份交给谁，开出任务组，写一次任务组卡片，再把回执交给路由。违反时返回路由能照着改的说明。
func (a *App) splitOne(rt *superagent.Runtime, sessionID string, args map[string]any) (string, error) {
	trigger := strParam(args, "trigger")
	mode := strings.ToLower(strParam(args, "mode"))
	req := superagent.SplitRequest{
		Trigger: trigger, Mode: mode, Title: strParam(args, "title"), Reason: strParam(args, "reason"),
		Rules: ruleParam(args["rules"]), Brief: strings.TrimSpace(strParam(args, "brief")), Session: sessionID,
	}
	switch mode {
	case superagent.GroupSplit:
		req.Parts = splitPartsParam(args["parts"])
	case superagent.GroupCompare:
		for _, agent := range stringsParam(args["agents"]) {
			req.Parts = append(req.Parts, superagent.SplitPart{Agent: agent})
		}
		if len(req.Parts) == 0 {
			req.Parts = splitPartsParam(args["parts"])
		}
	default:
		return "", fmt.Errorf("mode 只能是 split（按分工拆成几份）或 compare（几个 agent 分别做同一件事再比较）")
	}
	field := "parts"
	if mode == superagent.GroupCompare && len(stringsParam(args["agents"])) > 0 {
		field = "agents"
	}
	for i := range req.Parts {
		pick, _, err := a.routeAgent(rt, trigger, req.Parts[i].Agent)
		if err != nil {
			return "", fmt.Errorf("%s[%d]：%s", field, i, err.Error())
		}
		req.Parts[i].Agent = firstNonEmpty(pick, buddy.YoyoID)
	}
	g, err := rt.Split(req)
	if err != nil {
		return "", splitError(err, mode, trigger)
	}
	members := rt.GroupMembers(g.ID)
	if !a.groupCard(g, members) {
		return "这条触发已经开过任务组 " + g.ID + "「" + g.Title + "」，没有再开。要改哪一份就对那张工单用 task_route continue。", nil
	}
	return a.splitReceipt(g, members, rt.TriggerAgents(trigger)), nil
}

// splitReceipt 是开出任务组后给路由的回执：每个成员一行，人 @ 过却没分到的 agent 提醒一句，再说接下来怎么做。
func (a *App) splitReceipt(g superagent.Group, members []superagent.Ticket, mentioned []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "开了任务组 %s「%s」（%s，%d 份）：\n", g.ID, g.Title, groupModeWord(g.Mode), len(members))
	assigned := map[string]bool{}
	for _, t := range members {
		agent := firstNonEmpty(t.Agent, buddy.YoyoID)
		assigned[agent] = true
		state := "开工了"
		if len(t.DependsOn) > 0 {
			state = "等 " + strings.Join(t.DependsOn, "、") + " 做完再开工"
		}
		fmt.Fprintf(&b, "%s %s · %s：%s\n", t.ID, a.agentDisplay(agent), firstNonEmpty(t.Role, t.Title), state)
	}
	var missing []string
	for _, id := range mentioned {
		if !assigned[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		b.WriteString("人 @ 过的 " + strings.Join(missing, "、") + " 没有分到。人要它参与的话，用 task_route continue 把其中一份改派给它；不用它就告诉人为什么。\n")
	}
	b.WriteString("任务组卡片已经写进对话，不用再列一遍成员。对人只说一两句：")
	if g.Mode == superagent.GroupCompare {
		b.WriteString("交给了谁、各自独立做。都做完后会再叫你来比较。")
	} else {
		b.WriteString("怎么分的、谁先做。有了结果或要人处理时会再叫你。")
	}
	return b.String()
}

// splitError 把任务组的错误码换成路由能照着改的说明。mode 和 trigger 用来写得具体。
func splitError(err error, mode, trigger string) error {
	switch err.Error() {
	case "unknownTrigger":
		return fmt.Errorf("没有这条触发 %s。trigger 要填这一轮材料里给的编号", trigger)
	case "settled":
		return fmt.Errorf("这条触发已经路由过了。要改就对那张工单用 task_route continue")
	case "workerDecides":
		return fmt.Errorf("这条消息落在一张工单跟着的线程上，由那张工单的工人处理，不要再拆")
	case "observed":
		return fmt.Errorf("现在不是 auto 模式，推断出的工作只记下、不开工。人明确要做时会直接交代")
	case "needs_rule":
		return fmt.Errorf("推断的工作要在 rules 里写至少一条还启用的规则编号")
	case "not_human", "mode_off", "not_admitted":
		return fmt.Errorf("这条触发不能接（%s）。只是聊天就忽略", err.Error())
	case "empty_title":
		return fmt.Errorf("title 不能空，写任务组的短名")
	case "bad_parts":
		if mode == superagent.GroupCompare {
			return fmt.Errorf("compare 要 2 到 4 个 agent")
		}
		return fmt.Errorf("split 要拆成 2 到 6 份。只有一份就用 task_route accept")
	case "empty_agent":
		return fmt.Errorf("每一份都要填 agent")
	case "empty_brief":
		if mode == superagent.GroupCompare {
			return fmt.Errorf("compare 要写 brief：交给每个 agent 的同一份完整说明")
		}
		return fmt.Errorf("每一份都要写 brief：能独立读懂，写清交付物和放在哪")
	case "bad_deps":
		return fmt.Errorf("depends_on 只能填排在前面的份在 parts 里的序号，从 0 开始，不能指向自己或后面的份")
	case "compare_deps":
		return fmt.Errorf("compare 不能写 depends_on，各自独立做")
	case "dup_agents":
		return fmt.Errorf("compare 的 agent 不能重复，每个 agent 只做一份")
	}
	return err
}

// splitPartsParam 从 task_split 的 parts 里取出每一份。认不出的项照样占一个位置，交给引擎去拒。
func splitPartsParam(raw any) []superagent.SplitPart {
	items, _ := raw.([]any)
	out := make([]superagent.SplitPart, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		out = append(out, superagent.SplitPart{
			Title: strParam(m, "title"), Agent: strParam(m, "agent"), Role: strParam(m, "role"),
			Brief: strings.TrimSpace(strParam(m, "brief")), DependsOn: intsParam(m["depends_on"]),
		})
	}
	return out
}

// intsParam 取出一列序号。不是整数的项记成 -1，让引擎按越界拒掉，不悄悄丢掉。
func intsParam(raw any) []int {
	items, _ := raw.([]any)
	out := make([]int, 0, len(items))
	for _, item := range items {
		n := -1
		switch v := item.(type) {
		case float64:
			if v == float64(int(v)) {
				n = int(v)
			}
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				n = i
			}
		}
		out = append(out, n)
	}
	return out
}

// stringsParam 取出一列字符串。数组和逗号隔开的一行都可以。
func stringsParam(raw any) []string {
	var out []string
	switch v := raw.(type) {
	case string:
		for _, s := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '，' || r == '、' }) {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	}
	return out
}

// groupCard 把任务组写成发起会话里的一张卡片：编号、标题、模式和成员。同一个任务组只写一次，已经写过时返回假。
// 正文是界面认不出卡片时的退路，也是路由回看历史时读到的那一段。
func (a *App) groupCard(g superagent.Group, members []superagent.Ticket) bool {
	if a == nil || a.Traces == nil || strings.TrimSpace(g.Session) == "" {
		return true
	}
	id := "group:" + g.ID
	if a.traceHasID(g.Session, id) {
		return false
	}
	var lines []string
	rows := make([]map[string]any, 0, len(members))
	for _, t := range members {
		agent := firstNonEmpty(t.Agent, buddy.YoyoID)
		name := a.agentDisplay(agent)
		row := map[string]any{"ticket": t.ID, "agent": agent, "agentName": name, "role": t.Role, "title": t.Title}
		line := "- " + t.ID + " " + name
		if label := firstNonEmpty(t.Role, t.Title); label != "" {
			line += " · " + label
		}
		if len(t.DependsOn) > 0 {
			row["dependsOn"] = slices.Clone(t.DependsOn)
			line += "（等 " + strings.Join(t.DependsOn, "、") + "）"
		}
		rows = append(rows, row)
		lines = append(lines, line)
	}
	text := fmt.Sprintf("任务组「%s」· %s %d 份\n%s", g.Title, groupModeWord(g.Mode), len(members), strings.Join(lines, "\n"))
	ev := trace.Event{
		Type:      trace.TypeAssistant,
		Source:    "worker",
		SessionID: g.Session,
		Payload: map[string]any{
			"id": id, "kind": "group", "group": g.ID, "title": g.Title, "mode": g.Mode, "members": rows, "text": text,
		},
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return true
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
	return true
}

// agentDisplay 是 agent 给人和路由看的名字。yoyo 自带工人就是 yoyo，其余没有名字时用编号。
func (a *App) agentDisplay(agent string) string {
	if agent == "" || agent == buddy.YoyoID {
		return buddy.YoyoID
	}
	return firstNonEmpty(a.buddyName(agent), agent)
}

// groupModeWord 是任务组模式的说法。
func groupModeWord(mode string) string {
	if mode == superagent.GroupCompare {
		return "对比"
	}
	return "分工"
}

// pickTool 在对比的任务组里记下人采用哪一份，并停掉还在做的其他成员。
func (a *App) pickTool() runtime.ExtraTool {
	return routeExtra("task_pick", "在对比的任务组里记下采用哪一份，并停掉还在做的其他成员。人同意采用之后才调用。group 是任务组编号，ticket 是采用的那张工单，reason 写给人看的原因。", map[string]any{
		"group":  map[string]any{"type": "string"},
		"ticket": map[string]any{"type": "string"},
		"reason": map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		group, ticket := strParam(args, "group"), strParam(args, "ticket")
		if err := rt.Act("pick", ticket, "", group, strParam(args, "reason")); err != nil {
			return "", groupActError(err, group, ticket)
		}
		return "采用了 " + ticket + "，任务组 " + group + " 里其他还在做的成员停下了。要把结果合回项目时，对 " + ticket + " 用 task_route continue。", nil
	})
}

// recommendTool 在对比的任务组里记下路由推荐哪一份。界面在任务组卡片和对比页上标出来，不停任何成员。
func (a *App) recommendTool() runtime.ExtraTool {
	return routeExtra("task_recommend", "对比做完、比较之后，记下你推荐采用哪一份；界面会在任务组卡片和对比页上标出来。这不是采用：问过人、人同意后再用 task_pick。group 是任务组编号，ticket 是推荐的那张工单。", map[string]any{
		"group":  map[string]any{"type": "string"},
		"ticket": map[string]any{"type": "string"},
	}, false, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		group, ticket := strParam(args, "group"), strParam(args, "ticket")
		if err := rt.Act("recommend", ticket, "", group, ""); err != nil {
			return "", groupActError(err, group, ticket)
		}
		return "记下了：任务组 " + group + " 推荐 " + ticket + "。接着问人要不要采用。", nil
	})
}

// groupActError 把任务组动作的错误码换成路由能照着改的说明。
func groupActError(err error, group, ticket string) error {
	switch err.Error() {
	case "unknownGroup":
		return fmt.Errorf("没有任务组 %s。编号见任务组卡片或 task_status", group)
	case "groupEnded":
		return fmt.Errorf("任务组 %s 已经结束了", group)
	case "notCompare":
		return fmt.Errorf("任务组 %s 是分工，不用采用哪一份；每一份做完就算完成", group)
	case "notMember":
		return fmt.Errorf("%s 不在任务组 %s 里", ticket, group)
	}
	return err
}

// readTool 读一张工单的详情，比较对比结果或要细节时用。
func (a *App) readTool() runtime.ExtraTool {
	return routeExtra("task_read", "读一张工单的详情：状态、工人最后一段话、最后一份报告、写过的文件、目录、agent 和用量。比较对比的结果、或任务组的回报不够细时用。", map[string]any{
		"ticket": map[string]any{"type": "string"},
	}, true, func(args map[string]any) (string, error) {
		rt := a.orchestrator()
		if rt == nil {
			return "", fmt.Errorf("orchestrator unavailable")
		}
		id := strParam(args, "ticket")
		read, ok := rt.ReadTicket(id)
		if !ok {
			return "", fmt.Errorf("没有工单 %s", id)
		}
		return a.ticketReadText(read), nil
	})
}

// ticketReadText 把 task_read 读到的详情写成几行，工人最后一段话放在最后。
func (a *App) ticketReadText(read superagent.TicketRead) string {
	var b strings.Builder
	head := read.ID
	if read.Title != "" {
		head += "「" + read.Title + "」"
	}
	fmt.Fprintf(&b, "%s · %s · %s\n", head, a.agentDisplay(read.Agent), read.State)
	if read.Group != "" {
		line := "任务组 " + read.Group
		if read.Role != "" {
			line += " · " + read.Role
		}
		b.WriteString(line + "\n")
	}
	if read.Directory != "" {
		b.WriteString("目录 " + read.Directory + "\n")
	}
	if read.Report != nil {
		fmt.Fprintf(&b, "最后一份报告（%s）：%s\n", read.Report.State, strings.TrimSpace(read.Report.Summary))
		if q := strings.TrimSpace(read.Report.Question); q != "" {
			b.WriteString("问题：" + q + "\n")
		}
	}
	if len(read.Files) > 0 {
		b.WriteString("写过：" + strings.Join(read.Files, "、") + "\n")
	}
	if read.Usage != "" {
		b.WriteString("用量：" + read.Usage + "\n")
	}
	if read.Text != "" {
		b.WriteString("工人最后说：\n" + read.Text)
	}
	return strings.TrimSpace(b.String())
}

// compareWords 是人要几个 agent 各做一遍再比较时常说的词。
var compareWords = regexp.MustCompile(`(?i)分别|各自|各做|都做一遍|都试试|对比|比较|比一比|\bpk\b|\bcompare\b|\beach\b|\bvs\.?\b`)
