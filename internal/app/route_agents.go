// route_agents.go 拥有路由这一侧的 agent 选择：人 @ 过的 agent 怎样核对、记进触发、写进这一轮的说明，
// task_route 的 agent 怎样补缺省、校验和改派，以及给路由看的可派名单。
// 不拥有工人的打开和审批（buddy_worker.go），也不拥有 agent 的检测、安装和登录（internal/buddy）。

package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// rosterMax 是给路由看的可派名单最多几行。
const rosterMax = 8

// agentIntake 取出 text 里 @ 过的 agent，用 Manager 核对。返回认得的编号（按出现顺序）和写给路由的说明，没有可说的时为空串。
// 认不出的丢掉并说一句；点名的 agent 现在派不了时，让路由去问人，不擅自换人。
func (a *App) agentIntake(text string) ([]string, string) {
	m := a.Buddies()
	var known, lines []string
	for _, id := range runtime.MentionAgents(text) {
		if id == buddy.YoyoID {
			known = append(known, id)
			continue
		}
		if m == nil {
			lines = append(lines, "@agent:"+id+" 不是已知的 agent，已忽略。")
			continue
		}
		v, err := m.Get(id)
		if err != nil {
			lines = append(lines, "@agent:"+id+" 不是已知的 agent，已忽略。")
			continue
		}
		known = append(known, id)
		if reason := blockedReason(v); reason != "" {
			lines = append(lines, fmt.Sprintf("%s 现在离线（%s）。告诉人，问是等它上线还是改用 yoyo。不要擅自换人。", id, reason))
		}
	}
	s := a.buddySettings()
	switch {
	case len(known) == 1:
		lines = append([]string{"人指定 " + known[0] + " 来做。accept 时 agent 填 " + known[0] + "。"}, lines...)
	case len(known) > 1:
		lines = append([]string{"人点了 " + strings.Join(known, "、") + "。一张工单只交给其中一个，accept 时 agent 从这几个里选。"}, lines...)
	case s.AutoPick:
		lines = append(lines, "人没有点名 agent。打开了自动挑选：可以按名单里的「擅长」选一个能派的 agent，也可以交给 yoyo。")
	case strings.TrimSpace(s.DefaultAgent) != "":
		lines = append(lines, "人没有点名 agent。agent 留空时交给默认的 "+strings.TrimSpace(s.DefaultAgent)+"。")
	}
	if roster := a.agentRoster(len(known) > 0, s); roster != "" {
		lines = append(lines, "可派的 agent：\n"+roster)
	}
	return known, strings.Join(lines, "\n")
}

// agentRoster 是给路由看的可派名单，最多 rosterMax 行，每行「编号 名字，在线或在做 N，擅长 …」。
// 人 @ 过 agent、打开了自动挑选或设了默认 agent 时才给；能派的只有 yoyo 时不给。
func (a *App) agentRoster(mentioned bool, s buddy.Settings) string {
	m := a.Buddies()
	if m == nil || (!mentioned && !s.AutoPick && strings.TrimSpace(s.DefaultAgent) == "") {
		return ""
	}
	var lines []string
	external := false
	for _, item := range m.Roster() {
		if !routableState(item.ID, item.State) {
			continue
		}
		status := "待确认"
		switch {
		case item.Working > 0:
			status = fmt.Sprintf("在做 %d", item.Working)
		case item.ID == buddy.YoyoID || item.State == buddy.StateOnline:
			status = "在线"
		}
		line := item.ID + " " + item.Name + "，" + status
		if strengths := strings.TrimSpace(item.Strengths); strengths != "" {
			line += "，擅长 " + clipRunes(strengths, 60)
		}
		lines = append(lines, line)
		external = external || item.ID != buddy.YoyoID
		if len(lines) == rosterMax {
			break
		}
	}
	if !external {
		return ""
	}
	return strings.Join(lines, "\n")
}

// routeAgent 定下 triggerID 这一项交给谁。agent 是路由填的，可以为空；返回空串表示 yoyo 自带工人。
// 只能交给人 @ 过的 agent 或 yoyo；人没 @ 时用默认 agent，打开自动挑选后也可以从名单里选。pinned 表示人点了名。
// 违反时返回路由能照着改的说明。
func (a *App) routeAgent(rt *superagent.Runtime, triggerID, agent string) (string, bool, error) {
	mentioned := rt.TriggerAgents(triggerID)
	s := a.buddySettings()
	def := strings.TrimSpace(s.DefaultAgent)
	agent = strings.ToLower(strings.TrimSpace(agent))
	if agent == "" {
		switch {
		case len(mentioned) == 1:
			agent = mentioned[0]
		case len(mentioned) > 1:
			return "", false, fmt.Errorf("人点了 %s，agent 要填其中一个", strings.Join(mentioned, "、"))
		case def != "" && def != buddy.YoyoID && a.agentRoutable(def) == "":
			agent = def
		default:
			return "", false, nil
		}
	}
	pinned := slices.Contains(mentioned, agent)
	if agent == buddy.YoyoID {
		return "", pinned, nil
	}
	if !pinned && (len(mentioned) > 0 || (agent != def && !s.AutoPick)) {
		return "", false, fmt.Errorf("人没有点名 %s。%s", agent, a.agentChoices(mentioned, def, s.AutoPick))
	}
	if reason := a.agentRoutable(agent); reason != "" {
		return "", false, fmt.Errorf("%s 现在派不了（%s）。现在能派的有：%s。%s", agent, reason, strings.Join(a.routableIDs(), "、"), a.agentChoices(mentioned, def, s.AutoPick))
	}
	return agent, pinned, nil
}

// switchAgent 在路由给的 agent 和 ticket 现在的不同时改派，返回给路由的回执；agent 为空或相同时什么都不做，返回空串。
func (a *App) switchAgent(rt *superagent.Runtime, ticket, trigger, agent, brief string) (string, error) {
	agent = strings.ToLower(strings.TrimSpace(agent))
	cur, ok := rt.TicketAgent(ticket)
	if agent == "" || !ok || agent == cur {
		return "", nil
	}
	pick, pinned, err := a.routeAgent(rt, trigger, agent)
	if err != nil {
		return "", err
	}
	if err := rt.Reassign(ticket, trigger, pick, brief, pinned); err != nil {
		return "", err
	}
	return "reassigned " + ticket + " to " + firstNonEmpty(pick, buddy.YoyoID), nil
}

// agentChoices 是这一项的 agent 能填什么，写进拒绝说明里。
func (a *App) agentChoices(mentioned []string, def string, auto bool) string {
	switch {
	case len(mentioned) > 0:
		return "agent 只能填人 @ 过的 " + strings.Join(mentioned, "、") + "，或 yoyo。"
	case auto:
		return "agent 可以填名单里能派的：" + strings.Join(a.routableIDs(), "、") + "。"
	case def != "" && def != buddy.YoyoID:
		return "人没有点名 agent，agent 只能填默认的 " + def + "，或 yoyo。"
	default:
		return "人没有点名 agent，agent 只能填 yoyo 或留空。"
	}
}

// agentRoutable 说明 id 现在为什么派不了。能派时为空串。
func (a *App) agentRoutable(id string) string {
	if id == buddy.YoyoID {
		return ""
	}
	m := a.Buddies()
	if m == nil {
		return "Buddy 没有启动"
	}
	v, err := m.Get(id)
	if err != nil {
		return "不是已知的 agent"
	}
	return blockedReason(v)
}

// routableIDs 是现在能派的 agent 编号，yoyo 在前。
func (a *App) routableIDs() []string {
	out := []string{buddy.YoyoID}
	m := a.Buddies()
	if m == nil {
		return out
	}
	for _, item := range m.Roster() {
		if item.ID != buddy.YoyoID && routableState(item.ID, item.State) {
			out = append(out, item.ID)
		}
	}
	return out
}

// blockedReason 说明 v 为什么现在不能接单。还没探测完和刚断开的不算：开工前工人会再探一次，真连不上时再问人。
func blockedReason(v buddy.View) string {
	switch {
	case v.ID == buddy.YoyoID || v.Builtin:
		return ""
	case v.Install.Kind == "none" || v.State == buddy.StateNotInstalled:
		return "没有安装"
	case !v.Config.Enabled:
		return "已停用"
	case routableState(v.ID, v.State):
		return ""
	}
	switch v.State {
	case buddy.StateInstalling:
		return "正在安装"
	case buddy.StateNeedsAuth:
		return "需要登录"
	case buddy.StateNeedsConfig:
		return "要先配置"
	case buddy.StateUnsupported:
		return "本机不支持"
	}
	if detail := strings.TrimSpace(v.Detail); detail != "" {
		return clipRunes(detail, 80)
	}
	return "连不上"
}

// routableState 在 state 下可以把工单交给 id 时为真：在线、还没探测完或刚断开。
func routableState(id string, state buddy.State) bool {
	if id == buddy.YoyoID {
		return true
	}
	switch state {
	case buddy.StateOnline, buddy.StateProbing, buddy.StateRecentlyLost:
		return true
	}
	return false
}
