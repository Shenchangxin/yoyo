// 个人引擎上的委派、目标和页面监视，以及 present_choices、personal_status、remember_fact。浏览器和连接器在 tools_personal.go。

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/personal"
)

// needPersonal 在工具或 Personal 为空时返回 personal engine unavailable。
func (t *WorkspaceTools) needPersonal() (*personal.Engine, error) {
	if t == nil || t.Personal == nil {
		return nil, fmt.Errorf("personal engine unavailable")
	}
	return t.Personal, nil
}

// delegateWork 用 title、prompt、kind 和若干输入字段调用 CreateTask，并返回任务 JSON。
// args 读取 kind、title、prompt、goal_id、interval_minutes，以及 path、csv、account、message_id、to、reply、url、condition、value、fields 和 input。
func (t *WorkspaceTools) delegateWork(args map[string]any) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	kind := personal.Kind(strings.ToLower(str(args["kind"])))
	input := map[string]any{}
	for _, k := range []string{"path", "csv", "account", "message_id", "to", "reply", "goal_id", "url", "condition", "value"} {
		if v := str(args[k]); v != "" {
			input[k] = v
		}
	}
	if v, ok := args["fields"].(map[string]any); ok {
		input["fields"] = v
	}
	if v, ok := args["input"].(map[string]any); ok {
		for k, x := range v {
			input[k] = x
		}
	}
	if n := intArg(args["interval_minutes"]); n > 0 {
		input["interval_minutes"] = n
	}
	task, err := p.CreateTask(str(args["title"]), str(args["prompt"]), kind, input, str(args["goal_id"]), t.SessionID)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(task)
	return ToolResult{Content: string(b)}
}

// completeWork 调用 Personal.Complete，返回任务 JSON。
// id 作为第一参交给 Personal.Complete。
// result 作为第二参交给 Personal.Complete。
func (t *WorkspaceTools) completeWork(id, result string) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	task, err := p.Complete(id, result)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(task)
	return ToolResult{Content: string(b)}
}

// answerWork 调用 Personal.Answer，返回任务 JSON。
// id 作为第一参交给 Personal.Answer。
// text 作为第二参交给 Personal.Answer。
func (t *WorkspaceTools) answerWork(id, text string) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	task, err := p.Answer(id, text)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(task)
	return ToolResult{Content: string(b)}
}

// createGoal 调用 CreateGoal。里程碑来自参数，第三个位置参数传空串。
// title 作为第一参交给 CreateGoal。
// description 作为第二参交给 CreateGoal。
// milestones 作为第四参交给 CreateGoal。
func (t *WorkspaceTools) createGoal(title, description string, milestones []string) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	g, err := p.CreateGoal(title, description, "", milestones)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(g)
	return ToolResult{Content: string(b)}
}

// watchPage 用 title、url、condition、value 和 interval_minutes 调用 Watch，返回 monitor 和 task 的 JSON。
// args 读取 title、url、condition、value 和 interval_minutes，交给 Watch。
func (t *WorkspaceTools) watchPage(args map[string]any) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	interval := intArg(args["interval_minutes"])
	m, task, err := p.Watch(str(args["title"]), str(args["url"]), str(args["condition"]), str(args["value"]), interval)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(map[string]any{"monitor": m, "task": task})
	return ToolResult{Content: string(b)}
}

// presentChoices 把 options 收成 ChoiceOption。没有 AskUser 时只返回选项 JSON。有 AskUser 时弹出第一项为推荐项的表单，能对上选项 ID 就 SelectChoice。
// title 连同收出来的选项交给 PresentChoices。
// raw 只在动态类型是 []any 时逐项收成 ChoiceOption；map 读 id、label、source 或 sources，其他值去掉空白后非空才当成标签。
func (t *WorkspaceTools) presentChoices(title string, raw any) ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	var opts []personal.ChoiceOption
	switch v := raw.(type) {
	case []any:
		for i, x := range v {
			switch o := x.(type) {
			case map[string]any:
				id := str(o["id"])
				if id == "" {
					id = fmt.Sprintf("%d", i+1)
				}
				label := str(o["label"])
				if label == "" {
					label = id
				}
				src := str(o["source"])
				if src == "" {
					src = str(o["sources"])
				}
				opts = append(opts, personal.ChoiceOption{ID: id, Label: label, Source: src})
			default:
				s := strings.TrimSpace(fmt.Sprint(x))
				if s != "" {
					opts = append(opts, personal.ChoiceOption{ID: fmt.Sprintf("%d", i+1), Label: s})
				}
			}
		}
	}
	c := p.PresentChoices(t.SessionID, title, opts)
	if t.AskUser == nil {
		b, _ := json.Marshal(c)
		return ToolResult{Content: string(b)}
	}
	q := AskQuestion{ID: "choice", Prompt: c.Title}
	ids := make([]string, 0, len(c.Options))
	for _, o := range c.Options {
		q.Options = append(q.Options, AskOption{ID: o.ID, Label: o.Label})
		ids = append(ids, o.ID)
	}
	if len(q.Options) > 0 {
		q.Recommended = q.Options[0].ID
	}
	res := t.askForm(AskForm{Title: c.Title, Questions: []AskQuestion{q}})
	if res.Err != nil {
		return res
	}
	if id := chosenOption(res.Content, ids); id != "" {
		if got, e := p.SelectChoice(c.ID, id); e == nil {
			return ToolResult{Content: res.Content + "\nselected " + got.Selected}
		}
	}
	return ToolResult{Content: res.Content}
}

// personalStatus 返回任务、想法、目标、监视、提案和 worker 的计数，并列出等待输入或等待审批的任务。
func (t *WorkspaceTools) personalStatus() ToolResult {
	p, err := t.needPersonal()
	if err != nil {
		return ToolResult{Err: err}
	}
	s := p.Snapshot()
	summary := map[string]any{
		"tasks":     len(s.Tasks),
		"ideas":     len(s.Ideas),
		"goals":     len(s.Goals),
		"monitors":  len(s.Monitors),
		"proposals": len(s.Proposals),
		"worker":    s.Worker,
	}
	var waiting []map[string]any
	for _, task := range s.Tasks {
		if task.Status == personal.StatusWaitingInput || task.Status == personal.StatusWaitingApproval {
			waiting = append(waiting, map[string]any{"id": task.ID, "title": task.Title, "status": task.Status, "question": task.Question, "action_id": task.ActionID})
		}
	}
	summary["waiting"] = waiting
	b, _ := json.MarshalIndent(summary, "", "  ")
	return ToolResult{Content: string(b)}
}

// rememberFact 在 Memory 为空或文本为空白时返回错误。写入 KindProfile，来源固定为 User confirmed in chat。
// text 去掉空白，空串返回错误；否则作为 KindProfile 的正文写入 Memory。
func (t *WorkspaceTools) rememberFact(text string) ToolResult {
	if t.Memory == nil {
		return ToolResult{Err: fmt.Errorf("no memory")}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ToolResult{Err: fmt.Errorf("empty fact")}
	}
	it := t.Memory.Write(memory.Item{Kind: memory.KindProfile, Text: text, Source: "User confirmed in chat"})
	b, _ := json.Marshal(it)
	return ToolResult{Content: "remembered " + string(b)}
}
