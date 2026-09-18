package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/cite"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/office"
	"github.com/Shenchangxin/yoyo/internal/osutil"
	"github.com/Shenchangxin/yoyo/internal/preview"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func (t *WorkspaceTools) officeCreate(args map[string]any) ToolResult {
	rel := str(args["path"])
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "office_create", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	req := office.CreateReq{
		Kind:  office.Kind(str(args["kind"])),
		Title: str(args["title"]),
		Body:  str(args["body"]),
	}
	if h, ok := args["headings"].([]any); ok {
		for _, v := range h {
			req.Headings = append(req.Headings, fmt.Sprint(v))
		}
	}
	if rows, ok := args["rows"].([]any); ok {
		for _, r := range rows {
			var row []string
			switch rr := r.(type) {
			case []any:
				for _, c := range rr {
					row = append(row, fmt.Sprint(c))
				}
			case []string:
				row = rr
			}
			req.Rows = append(req.Rows, row)
		}
	}
	if s, ok := args["slides"].([]any); ok {
		for _, v := range s {
			req.Slides = append(req.Slides, fmt.Sprint(v))
		}
	}
	if err := office.Create(p, req); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "wrote " + rel, FileChange: &FileChange{Paths: []string{rel}}}
}

func (t *WorkspaceTools) officeEdit(rel, old, neu string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "office_edit", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	if err := office.EditReplace(p, old, neu); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "edited " + rel, FileChange: &FileChange{Paths: []string{rel}}}
}

func (t *WorkspaceTools) officeQuery(rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "office_query", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	out, err := office.Query(p)
	return ToolResult{Content: out, Err: err}
}

func (t *WorkspaceTools) officeRender(rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "office_render", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	out, err := preview.Render(p)
	return ToolResult{Content: out, Err: err}
}

func (t *WorkspaceTools) citeSources(rel string, raw string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "cite_sources", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	var payload struct {
		Sources []cite.Source `json:"sources"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ToolResult{Err: err}
	}
	for i, s := range payload.Sources {
		if s.Hash == "" {
			payload.Sources[i] = cite.HashExcerpt(s.URL, s.Excerpt)
		}
	}
	if err := cite.Write(p, payload.Sources); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: fmt.Sprintf("wrote %d citations to %s", len(payload.Sources), rel), FileChange: &FileChange{Paths: []string{rel}}}
}

func (t *WorkspaceTools) memorySearch(q, kind string) ToolResult {
	if t.Memory == nil {
		return ToolResult{Content: "memory store unavailable"}
	}
	items := t.Memory.Search(q, memory.Kind(kind), 20)
	b, _ := json.MarshalIndent(items, "", "  ")
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) memoryWrite(kind, text, project string) ToolResult {
	if err := t.check(capability.MemoryWrite, "memory_write", "", text); err != nil {
		return ToolResult{Err: err}
	}
	if t.Memory == nil {
		return ToolResult{Err: fmt.Errorf("no memory store")}
	}
	it := t.Memory.Write(memory.Item{Kind: memory.Kind(kind), Text: text, Project: project})
	return ToolResult{Content: fmt.Sprintf("staged memory %s (kind=%s). checkout still owns promotion.", it.ID, it.Kind)}
}

func (t *WorkspaceTools) memoryForget(id string) ToolResult {
	if err := t.check(capability.MemoryWrite, "memory_forget", "", id); err != nil {
		return ToolResult{Err: err}
	}
	if t.Memory == nil {
		return ToolResult{Err: fmt.Errorf("no memory store")}
	}
	if !t.Memory.Forget(id) {
		return ToolResult{Err: fmt.Errorf("unknown memory id")}
	}
	return ToolResult{Content: "forgot " + id}
}

func (t *WorkspaceTools) scheduleCreate(kind, spec, prompt string) ToolResult {
	if err := t.check(capability.Schedule, "schedule_create", "", prompt); err != nil {
		return ToolResult{Err: err}
	}
	if t.Schedule == nil {
		return ToolResult{Err: fmt.Errorf("no scheduler")}
	}
	j := t.Schedule.Create(schedule.Job{Kind: schedule.Kind(kind), Spec: spec, Prompt: prompt, Workspace: t.Workspace, Isolate: true})
	b, _ := json.Marshal(j)
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) scheduleList() ToolResult {
	if t.Schedule == nil {
		return ToolResult{Content: "[]"}
	}
	b, _ := json.MarshalIndent(t.Schedule.List(), "", "  ")
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) scheduleCancel(id string) ToolResult {
	if err := t.check(capability.Schedule, "schedule_cancel", "", id); err != nil {
		return ToolResult{Err: err}
	}
	if t.Schedule == nil || !t.Schedule.Cancel(id) {
		return ToolResult{Err: fmt.Errorf("unknown job")}
	}
	return ToolResult{Content: "cancelled " + id}
}

func (t *WorkspaceTools) browserOpen(raw string) ToolResult {
	if err := t.check(capability.Browser, "browser_open", raw, raw); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	snap, err := t.Browser.OpenURL(raw)
	if err != nil {
		return ToolResult{Err: err}
	}
	t.mu.Lock()
	t.LastBrowser = snap.Text
	t.mu.Unlock()
	return ToolResult{Content: fmt.Sprintf("%s\nprofile=%s\n\n%s", snap.URL, snap.Profile, snap.Text)}
}

func (t *WorkspaceTools) browserSnapshot() ToolResult {
	t.mu.Lock()
	s := t.LastBrowser
	t.mu.Unlock()
	if s == "" {
		return ToolResult{Content: "no snapshot"}
	}
	return ToolResult{Content: s}
}

func (t *WorkspaceTools) browserClick(sel string) ToolResult {
	if err := t.check(capability.Browser, "browser_click", sel, sel); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	err := t.Browser.Click(sel)
	return ToolResult{Content: "recorded click " + sel, Err: err}
}

func (t *WorkspaceTools) browserType(sel, text string) ToolResult {
	if err := t.check(capability.Browser, "browser_type", sel, text); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	err := t.Browser.Type(sel, text)
	return ToolResult{Err: err, Content: "recorded type " + sel}
}

func (t *WorkspaceTools) browserDownload(raw, rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.Browser, "browser_download", p, raw); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.Download(raw, p); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "downloaded " + rel, FileChange: &FileChange{Paths: []string{rel}}}
}

func (t *WorkspaceTools) clipboardRead() ToolResult {
	s, err := osutil.ClipboardRead()
	if err != nil {
		return ToolResult{Err: err}
	}
	capped, _ := capText(s, 4000)
	return ToolResult{Content: "clipboard (untrusted):\n" + capped}
}

func (t *WorkspaceTools) clipboardWrite(text string) ToolResult {
	if err := osutil.ClipboardWrite(text); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "clipboard updated"}
}

func (t *WorkspaceTools) screenshot(rel string) ToolResult {
	if rel == "" {
		rel = filepath.ToSlash(filepath.Join(".yoyo", "captures", "shot.png"))
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "screenshot_region", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	if err := osutil.Screenshot(p); err != nil {
		return ToolResult{Err: err}
	}
	return t.viewImage(rel)
}

func (t *WorkspaceTools) fsBatch(from, to string) ToolResult {
	src, err := t.resolve(from)
	if err != nil {
		return ToolResult{Err: err}
	}
	dst, err := t.resolve(to)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "fs_batch", dst, from+" -> "+to); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.Rename(src, dst); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "moved " + from + " -> " + to, FileChange: &FileChange{Paths: []string{from, to}}}
}

func (t *WorkspaceTools) connectorRead(account, query string) ToolResult {
	if err := t.check(capability.ReadConnector, "connector_read", account, query); err != nil {
		return ToolResult{Err: err}
	}
	if t.Connectors == nil {
		return ToolResult{Err: fmt.Errorf("no connectors")}
	}
	items, err := t.Connectors.Read(account, query)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.MarshalIndent(items, "", "  ")
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) connectorDraft(account, to, subject, body string) ToolResult {
	if err := t.check(capability.WriteConnector, "connector_draft", account, to); err != nil {
		return ToolResult{Err: err}
	}
	if t.Connectors == nil {
		return ToolResult{Err: fmt.Errorf("no connectors")}
	}
	d := t.Connectors.Draft(account, to, subject, body)
	b, _ := json.Marshal(d)
	return ToolResult{Content: "draft " + string(b) + " — send with connector_send after send_as_you approval"}
}

func (t *WorkspaceTools) connectorSend(id string) ToolResult {
	req := capability.Request{
		Level:     capability.SendAsYou,
		Action:    "connector_send",
		Command:   id,
		SessionID: t.SessionID,
		Workspace: t.Workspace,
		ForceAsk:  true,
	}
	if t.Caps != nil {
		var err error
		if t.Ctx != nil {
			err = t.Caps.CheckCtx(t.Ctx, req)
		} else {
			err = t.Caps.Check(req)
		}
		if err != nil {
			return ToolResult{Err: err}
		}
	}
	if t.Connectors == nil {
		return ToolResult{Err: fmt.Errorf("no connectors")}
	}
	d, err := t.Connectors.Send(id)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(d)
	return ToolResult{Content: "sent " + string(b)}
}

func (t *WorkspaceTools) computerAct(app, op, detail string) ToolResult {
	req := capability.Request{
		Level:     capability.ComputerUse,
		Action:    "computer_act",
		Command:   op + " " + app,
		SessionID: t.SessionID,
		Workspace: t.Workspace,
		ForceAsk:  true,
	}
	if t.Caps != nil {
		var err error
		if t.Ctx != nil {
			err = t.Caps.CheckCtx(t.Ctx, req)
		} else {
			err = t.Caps.Check(req)
		}
		if err != nil {
			return ToolResult{Err: err}
		}
	}
	if t.Computer == nil {
		return ToolResult{Err: fmt.Errorf("no computer-use host")}
	}
	ev, err := t.Computer.Act(app, op, detail)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(ev)
	return ToolResult{Content: "virtual-display " + string(b)}
}

func (t *WorkspaceTools) projectList() ToolResult {
	if t.Projects == nil {
		return ToolResult{Content: "[]"}
	}
	b, _ := json.MarshalIndent(t.Projects.List(), "", "  ")
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) notifyActionable(title, body string) ToolResult {
	if t.Inbox == nil {
		return ToolResult{Content: "inbox unavailable; noted " + title}
	}
	it := t.Inbox.Push(inbox.Item{Kind: inbox.KindAsk, Title: title, Body: body, SessionID: t.SessionID})
	return ToolResult{Content: "inbox " + it.ID}
}

func encodeDataURL(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) > 2<<20 {
		b = b[:2<<20]
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}
