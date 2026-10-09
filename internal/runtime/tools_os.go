package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/personal"
)

func (t *WorkspaceTools) needPersonal() (*personal.Engine, error) {
	if t == nil || t.Personal == nil {
		return nil, fmt.Errorf("personal engine unavailable")
	}
	return t.Personal, nil
}

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
	prompt := c.Title
	for _, o := range c.Options {
		prompt += "\n" + o.ID + ". " + o.Label
	}
	if t.AskUser != nil {
		ans, err := t.AskUser(prompt)
		if err != nil {
			return ToolResult{Err: err}
		}
		if got, e := p.SelectChoice(c.ID, strings.TrimSpace(ans)); e == nil {
			return ToolResult{Content: "selected " + got.Selected}
		}
		return ToolResult{Content: "answered " + ans}
	}
	b, _ := json.Marshal(c)
	return ToolResult{Content: string(b)}
}

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
