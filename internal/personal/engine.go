package personal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
)

type Bind struct {
	Connectors *connector.Broker
	Inbox      *inbox.Store
	Workspace  string
	Fetch      func(ctx context.Context, url string) (string, error)
	Now        func() time.Time
	Poll       time.Duration
	Lease      time.Duration
	OnChange   func()
}

type Engine struct {
	dir        string
	store      *Store
	Connectors *connector.Broker
	Inbox      *inbox.Store
	Workspace  string
	Fetch      func(ctx context.Context, url string) (string, error)
	Now        func() time.Time
	Poll       time.Duration
	Lease      time.Duration
	OnChange   func()

	mu       sync.Mutex
	running  bool
	inFlight map[string]struct{}
	seq      atomic.Int64
	abort    map[string]context.CancelFunc
	lastTick time.Time
}

func Open(dir string) (*Engine, error) {
	st, err := openStore(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		return nil, err
	}
	return &Engine{
		dir:      dir,
		store:    st,
		Poll:     time.Second,
		Lease:    time.Minute,
		inFlight: map[string]struct{}{},
		abort:    map[string]context.CancelFunc{},
		Fetch:    SafeGET,
		Now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

func (e *Engine) Bind(b Bind) {
	if b.Connectors != nil {
		e.Connectors = b.Connectors
	}
	if b.Inbox != nil {
		e.Inbox = b.Inbox
	}
	if b.Workspace != "" {
		e.Workspace = b.Workspace
	}
	if b.Fetch != nil {
		e.Fetch = b.Fetch
	}
	if b.Now != nil {
		e.Now = b.Now
	}
	if b.Poll > 0 {
		e.Poll = b.Poll
	}
	if b.Lease > 0 {
		e.Lease = b.Lease
	}
	if b.OnChange != nil {
		e.OnChange = b.OnChange
	}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e *Engine) newID(prefix string) string {
	n := e.seq.Add(1)
	return fmt.Sprintf("%s-%s-%d", prefix, e.now().Format("150405.000"), n)
}

func hashID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (e *Engine) Snapshot() Snapshot {
	s := e.store.snapshot()
	e.mu.Lock()
	s.Worker.Running = e.running
	s.Worker.LastTickAt = e.lastTick
	e.mu.Unlock()
	return s
}

func (e *Engine) changed() {
	if e.OnChange != nil {
		e.OnChange()
	}
}

func (e *Engine) notify(kind inbox.Kind, title, body, taskID, key string) {
	if e.Inbox == nil {
		return
	}
	e.Inbox.Push(inbox.Item{
		Kind:   kind,
		Title:  title,
		Body:   body,
		TaskID: taskID,
		Key:    key,
	})
	e.changed()
}

func (e *Engine) CreateTask(title, prompt string, kind Kind, input map[string]any, goalID, sessionID string) (Task, error) {
	kind = Kind(strings.ToLower(strings.TrimSpace(string(kind))))
	switch kind {
	case KindAgent, KindDocument, KindMonitor, KindFinance, KindPlan:
	default:
		kind = KindAgent
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" && title == "" {
		return Task{}, errf("prompt required")
	}
	if title == "" {
		title = clip(prompt, 80)
	}
	if input == nil {
		input = map[string]any{}
	}
	now := e.now()
	t := Task{
		ID:        e.newID("task"),
		Title:     clip(title, 160),
		Prompt:    prompt,
		Kind:      kind,
		Status:    StatusQueued,
		GoalID:    goalID,
		Input:     input,
		State:     map[string]any{},
		CreatedAt: now,
		UpdatedAt: now,
		SessionID: sessionID,
		Plan:      defaultPlan(kind),
	}
	if kind == KindAgent {
		t.Status = StatusWaitingInput
		t.Question = "Continue this work in chat, then call complete_work with the result. The worker does not start a second model loop."
	}
	e.store.putTask(t)
	e.changed()
	return t, nil
}

func defaultPlan(kind Kind) []Step {
	switch kind {
	case KindDocument:
		return []Step{
			{ID: "1", Title: "Inspect the document", Status: "pending"},
			{ID: "2", Title: "Write a filled copy", Status: "pending"},
			{ID: "3", Title: "Prepare a reply", Status: "pending"},
			{ID: "4", Title: "Wait for send approval", Status: "pending"},
		}
	case KindMonitor:
		return []Step{{ID: "1", Title: "Check the page", Status: "pending"}}
	case KindFinance:
		return []Step{{ID: "1", Title: "Analyze the CSV", Status: "pending"}}
	case KindPlan:
		return []Step{{ID: "1", Title: "Write milestones", Status: "pending"}}
	default:
		return []Step{{ID: "1", Title: "Do the work in chat", Status: "pending"}}
	}
}

func (e *Engine) Answer(id, text string) (Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Task{}, errf("empty answer")
	}
	t, ok := e.store.task(id)
	if !ok {
		return Task{}, errf("unknown task")
	}
	if t.Status != StatusWaitingInput {
		return Task{}, errf("task is not waiting for input")
	}
	input := cloneMap(t.Input)
	state := cloneMap(t.State)
	state["answer"] = text
	switch t.Kind {
	case KindDocument:
		fields := parseFields(text)
		if len(fields) == 0 {
			return Task{}, errf("reply with JSON object of field values")
		}
		input["fields"] = fields
	case KindPlan:
		var ms []string
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(strings.TrimLeft(line, "-* "))
			if line != "" {
				ms = append(ms, line)
			}
		}
		if len(ms) == 0 {
			return Task{}, errf("reply with milestone titles, one per line")
		}
		input["milestones"] = ms
	}
	out, ok := e.store.casTask(id, string(StatusWaitingInput), "", func(cur *Task) bool {
		cur.Input = input
		cur.State = state
		cur.Status = StatusQueued
		cur.Question = ""
		cur.Error = ""
		cur.UpdatedAt = e.now()
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		return true
	})
	if !ok {
		return t, errf("task changed")
	}
	e.changed()
	return out, nil
}

func parseFields(text string) map[string]string {
	text = strings.TrimSpace(text)
	out := map[string]string{}
	if strings.HasPrefix(text, "{") {
		var raw map[string]any
		if json.Unmarshal([]byte(text), &raw) == nil {
			for k, v := range raw {
				out[k] = strings.TrimSpace(fmt.Sprint(v))
			}
			return out
		}
	}
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			k, v, ok = strings.Cut(line, "=")
		}
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func (e *Engine) Complete(id, result string) (Task, error) {
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		cur.Status = StatusSucceeded
		cur.Result = result
		cur.Error = ""
		cur.Question = ""
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot complete task")
	}
	e.notify(inbox.KindWork, out.Title, result, out.ID, "task-done:"+out.ID)
	return out, nil
}

func (e *Engine) Fail(id, errMsg string) (Task, error) {
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		cur.Status = StatusFailed
		cur.Error = errMsg
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot fail task")
	}
	e.changed()
	return out, nil
}

func (e *Engine) Cancel(id string) (Task, error) {
	var actionID string
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		actionID = cur.ActionID
		cur.Status = StatusCancelled
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot cancel task")
	}
	if actionID != "" {
		if p, ok := e.store.proposal(actionID); ok && p.Status == ProposalAwaiting {
			p.Status = ProposalCancelled
			p.Activity = append(p.Activity, Activity{At: e.now(), Status: string(ProposalCancelled), Detail: "task cancelled"})
			e.store.putProposal(p)
		}
	}
	e.abortTask(id)
	for _, m := range e.store.snapshot().Monitors {
		if m.TaskID == id {
			m.Status = "stopped"
			e.store.putMonitor(m)
		}
	}
	e.changed()
	return out, nil
}

func (e *Engine) CreateGoal(title, description, category string, milestones []string) (Goal, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Goal{}, errf("goal title required")
	}
	if category == "" {
		category = "Personal"
	}
	now := e.now()
	g := Goal{
		ID:          e.newID("goal"),
		Title:       clip(title, 160),
		Description: clip(description, 4000),
		Category:    clip(category, 80),
		Status:      "active",
		CreatedAt:   now,
	}
	for i, m := range milestones {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		g.Milestones = append(g.Milestones, Milestone{ID: fmt.Sprintf("m%d", i+1), Title: clip(m, 200)})
	}
	e.store.putGoal(g)
	e.changed()
	return g, nil
}

func (e *Engine) Watch(title, rawURL, condition, value string, intervalMinutes int) (Monitor, Task, error) {
	title = strings.TrimSpace(title)
	rawURL = strings.TrimSpace(rawURL)
	if title == "" || rawURL == "" {
		return Monitor{}, Task{}, errf("title and url required")
	}
	if !strings.HasPrefix(rawURL, "sample://") {
		uok := strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
		if !uok {
			return Monitor{}, Task{}, errf("url must be http(s) or sample://")
		}
		if u, err := parsePublicURL(rawURL); err != nil {
			return Monitor{}, Task{}, err
		} else if blockedHost(u) {
			return Monitor{}, Task{}, errf("host is not allowed")
		}
	}
	condition = strings.ToLower(strings.TrimSpace(condition))
	if condition == "" {
		condition = "change"
	}
	switch condition {
	case "change", "contains", "price_below":
	default:
		return Monitor{}, Task{}, errf("condition must be change, contains, or price_below")
	}
	if condition != "change" && strings.TrimSpace(value) == "" {
		return Monitor{}, Task{}, errf("enter a condition value")
	}
	if condition == "price_below" {
		n := 0.0
		fmt.Sscanf(value, "%f", &n)
		if n <= 0 {
			return Monitor{}, Task{}, errf("enter a positive price")
		}
	}
	if intervalMinutes < 1 {
		intervalMinutes = 15
	}
	if intervalMinutes > 10080 {
		intervalMinutes = 10080
	}
	now := e.now()
	t, err := e.CreateTask(title, "Watch "+rawURL, KindMonitor, map[string]any{
		"title": title, "url": rawURL, "condition": condition, "value": value, "interval_minutes": intervalMinutes,
	}, "", "")
	if err != nil {
		return Monitor{}, Task{}, err
	}
	m := Monitor{
		ID:              e.newID("watch"),
		TaskID:          t.ID,
		Title:           clip(title, 160),
		URL:             rawURL,
		Condition:       condition,
		Value:           value,
		IntervalMinutes: intervalMinutes,
		Status:          "active",
		NextCheckAt:     now,
	}
	e.store.putMonitor(m)
	e.store.casTask(t.ID, "", "", func(cur *Task) bool {
		cur.Input["monitorId"] = m.ID
		cur.Status = StatusScheduled
		cur.NextRunAt = &now
		cur.UpdatedAt = now
		return true
	})
	t, _ = e.store.task(t.ID)
	e.changed()
	return m, t, nil
}

func parsePublicURL(raw string) (string, error) {
	// hostname only for blockedHost
	s := raw
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	host, _, _ := strings.Cut(s, "/")
	host, _, _ = strings.Cut(host, ":")
	if host == "" {
		return "", errf("need public http(s) URL")
	}
	return host, nil
}

func (e *Engine) PresentChoices(sessionID, title string, options []ChoiceOption) Choice {
	if title == "" {
		title = "Choose"
	}
	if len(options) == 0 {
		options = []ChoiceOption{{ID: "ok", Label: "OK"}}
	}
	comparison := false
	for _, o := range options {
		if strings.TrimSpace(o.Source) != "" {
			comparison = true
			break
		}
	}
	if comparison && len(options) > 3 {
		options = options[:3]
	}
	c := Choice{ID: e.newID("choice"), SessionID: sessionID, Title: title, Options: options, CreatedAt: e.now()}
	e.store.putChoice(c)
	var body strings.Builder
	for _, o := range options {
		fmt.Fprintf(&body, "- %s: %s\n", o.ID, o.Label)
	}
	e.notify(inbox.KindAsk, title, body.String(), "", "choice:"+c.ID)
	return c
}

func (e *Engine) SelectChoice(id, option string) (Choice, error) {
	c, ok := e.store.choice(id)
	if !ok {
		return Choice{}, errf("unknown choice")
	}
	option = strings.TrimSpace(option)
	okOpt := false
	for _, o := range c.Options {
		if o.ID == option || strings.EqualFold(o.Label, option) {
			option = o.ID
			okOpt = true
			break
		}
	}
	if !okOpt {
		return Choice{}, errf("unknown option")
	}
	c.Selected = option
	e.store.putChoice(c)
	e.changed()
	return c, nil
}

func (e *Engine) filesDir(taskID string) string {
	p := filepath.Join(e.dir, "files", taskID)
	_ = os.MkdirAll(p, 0o755)
	return p
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n > 0 && len(s) > n {
		return s[:n]
	}
	return s
}

func strMap(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func (e *Engine) SetSample(key, text string) {
	e.store.sample(key, text)
}
