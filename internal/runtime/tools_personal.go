// 办公文件、记忆、日程、浏览器、剪贴板、连接器和电脑操作的工具实现。名字登记在 tools_host.go。路径仍经 resolve，不在这里调用 EvalSymlinks。

package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/browser"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/cite"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/office"
	"github.com/Shenchangxin/yoyo/internal/osutil"
	"github.com/Shenchangxin/yoyo/internal/preview"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// officeCreate 经 resolve 后以 WriteWorkspace、动作 office_create 审批，再按 kind、title、body、headings、rows、slides 调用 office.Create。
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

// officeEdit 以 WriteWorkspace、动作 office_edit 审批。写入前 snapshotBeforeWrite，再 office.EditReplace。
func (t *WorkspaceTools) officeEdit(rel, old, neu string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "office_edit", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	t.snapshotBeforeWrite(rel, p)
	if err := office.EditReplace(p, old, neu); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "edited " + rel, FileChange: &FileChange{Paths: []string{rel}}}
}

// officeQuery 以 ReadWorkspace、动作 office_query 审批，再 office.Query。查询错误放在 Err 里，同时带上 Content。
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

// officeRender 以 ReadWorkspace、动作 office_render 审批，再 preview.Render。
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

// citeSources 以 WriteWorkspace、动作 cite_sources 审批。Hash 为空的来源会先 HashExcerpt。写入成功后若有 OnCapture 就记一条 brief。
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
	if t.OnCapture != nil {
		t.OnCapture("brief", rel, fmt.Sprintf("%d citations", len(payload.Sources)))
	}
	return ToolResult{Content: fmt.Sprintf("wrote %d citations to %s", len(payload.Sources), rel), FileChange: &FileChange{Paths: []string{rel}}}
}

// memorySearch 在 Memory 为空时返回 memory store unavailable，这不是 Err。否则按 kind 搜索，最多 20 条。
func (t *WorkspaceTools) memorySearch(q, kind string) ToolResult {
	if t.Memory == nil {
		return ToolResult{Content: "memory store unavailable"}
	}
	items := t.Memory.Search(q, memory.Kind(kind), 20)
	b, _ := json.MarshalIndent(items, "", "  ")
	return ToolResult{Content: string(b)}
}

// memoryWrite 以 MemoryWrite、动作 memory_write 审批。写入后说明仍由 checkout 决定提升，本函数不提升。
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

// memoryForget 以 MemoryWrite、动作 memory_forget 审批。未知 ID 返回错误。
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

// scheduleCreate 以 Schedule、动作 schedule_create 审批。没有调度器则返回错误。任务带上当前工作区和会话。
func (t *WorkspaceTools) scheduleCreate(kind, spec, prompt string) ToolResult {
	if err := t.check(capability.Schedule, "schedule_create", "", prompt); err != nil {
		return ToolResult{Err: err}
	}
	if t.Schedule == nil {
		return ToolResult{Err: fmt.Errorf("no scheduler")}
	}
	j := t.Schedule.Create(schedule.Job{
		Kind: schedule.Kind(kind), Spec: spec, Prompt: prompt,
		Workspace: t.Workspace, SessionID: t.SessionID,
	})
	b, _ := json.Marshal(j)
	return ToolResult{Content: string(b)}
}

// scheduleList 在 Schedule 为空时返回 []。否则返回任务列表的 JSON。
func (t *WorkspaceTools) scheduleList() ToolResult {
	if t.Schedule == nil {
		return ToolResult{Content: "[]"}
	}
	b, _ := json.MarshalIndent(t.Schedule.List(), "", "  ")
	return ToolResult{Content: string(b)}
}

// scheduleCancel 以 Schedule、动作 schedule_cancel 审批。调度器为空或 ID 未知则返回错误。
func (t *WorkspaceTools) scheduleCancel(id string) ToolResult {
	if err := t.check(capability.Schedule, "schedule_cancel", "", id); err != nil {
		return ToolResult{Err: err}
	}
	if t.Schedule == nil || !t.Schedule.Cancel(id) {
		return ToolResult{Err: fmt.Errorf("unknown job")}
	}
	return ToolResult{Content: "cancelled " + id}
}

// browserOpen 拒绝 data: URL。工作区相对路径走预览：能 resolve 时以 ReadWorkspace 审批，并 SetPreview。OpenLocal 失败时仍返回 Content 而不是 Err，因为工作台窗格已经打开，隔离 Chrome 没起来。远程 URL 以 Browser、动作 browser_open 审批，再 OpenURLLane。
func (t *WorkspaceTools) browserOpen(raw string) ToolResult {
	target, lane, profile := parseBrowserOpen(raw)
	if profile != "" && t.Browser != nil {
		if err := t.Browser.UseProfile(profile); err != nil {
			return ToolResult{Err: err}
		}
	}
	if isDataURL(target) {
		return ToolResult{Err: fmt.Errorf("browser_open: data: URLs are not supported. write_file the HTML, then browser_open the workspace path")}
	}
	if rel, err := t.workspacePreviewRel(target); err == nil {
		if abs, rerr := t.resolve(rel); rerr == nil {
			if err := t.check(capability.ReadWorkspace, "browser_open", abs, ""); err != nil {
				return ToolResult{Err: err}
			}
		}
		if t.Browser != nil {
			if abs, rerr := t.resolve(rel); rerr == nil {
				t.Browser.SetPreview(abs)
			}
		}
		t.mu.Lock()
		t.LastBrowser = "workspace:" + rel
		t.mu.Unlock()
		abs := ""
		if p, rerr := t.resolve(rel); rerr == nil {
			abs = p
		}
		if t.Browser != nil && abs != "" {
			snap, err := t.Browser.OpenLocal(abs)
			if err != nil {
				// 预览窗格已经打开，所以这是部分成功，放在 Content 里而不是 Err。没有隔离 Chrome 时工作台窗格仍可用。
				return ToolResult{Content: fmt.Sprintf("opened %s in the workstation Browser pane. Isolated Chrome did not start (%v), so browser_click and browser_type are unavailable until it does; browser_screenshot retries Chrome once. Do not call browser_open again for this path — reopening reloads and wipes clicks. Do not spawn Chrome, python -m http.server, or connect to CDP 9333.", rel, err)}
			}
			snap.Errors = t.Browser.PageErrors()
			t.mu.Lock()
			t.LastBrowser = snap.Text
			t.mu.Unlock()
			return ToolResult{Content: fmt.Sprintf("opened %s in the workstation Browser pane and isolated Chrome\nurl=%s\nchrome=ready\nThis is navigation (reloads the page). Do not browser_open the same file again unless HTML/JS/CSS changed — reopening wipes clicks. Next: browser_snapshot then browser_click a ref= from the list (neither reloads).\n\n%s", rel, snap.URL, withPageErrors(browserBody(snap.Text, snap.Actions), snap.Errors))}
		}
		return ToolResult{Content: fmt.Sprintf("opened %s in the workstation Browser pane (right inspector). Isolated Chrome is unavailable. Do not spawn Chrome, python -m http.server, or connect to CDP 9333.", rel)}
	} else if !isRemoteBrowserURL(target) && strings.TrimSpace(target) != "" {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.Browser, "browser_open", "", target); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	snap, err := t.Browser.OpenURLLane(target, lane)
	if err != nil {
		return ToolResult{Err: err}
	}
	t.mu.Lock()
	t.LastBrowser = snap.Text
	t.mu.Unlock()
	level := browser.NormalizeSnapshotLevel("", snap.URL)
	body := browser.FormatSnapshot(browser.Snapshot{
		URL: snap.URL, Title: snap.Title, Text: snap.Text, Actions: snap.Actions,
		Errors: snap.Errors, Elements: snap.Elements, Profile: t.Browser.ProfileName(),
	}, level, true)
	return ToolResult{Content: fmt.Sprintf("%s\nprofile=%s\nlane=%s\n\n%s", snap.URL, t.Browser.ProfileName(), lane, body)}
}

// browserBody 拼出模型读的页面报告：可见文本，以及下一步可点的 ref= 句柄。没有动作列表时只返回文本。
func browserBody(text, actions string) string {
	if strings.TrimSpace(actions) == "" {
		return text
	}
	return text + "\n\nvisible actions (prefer ref= on browser_click):\n" + actions
}

// withPageErrors 把上次浏览器调用以来的 JavaScript 错误附在报告后面。错误为空则原样返回正文。
func withPageErrors(body, errs string) string {
	if strings.TrimSpace(errs) == "" {
		return body
	}
	return body + "\n\n" + errs
}

// browserSnapshot 在 Browser 能 Refresh 且 URL 非空时格式化活页，并把 live 标志传给 FormatSnapshot。否则返回上次记下的 LastBrowser；空则 no snapshot。
func (t *WorkspaceTools) browserSnapshot(level string) ToolResult {
	if t.Browser != nil {
		snap, live := t.Browser.Refresh()
		if snap.URL != "" {
			t.mu.Lock()
			t.LastBrowser = snap.Text
			t.mu.Unlock()
			return ToolResult{Content: browser.FormatSnapshot(snap, level, live)}
		}
	}
	t.mu.Lock()
	s := t.LastBrowser
	t.mu.Unlock()
	if s == "" {
		return ToolResult{Content: "no snapshot"}
	}
	return ToolResult{Content: s}
}

// browserClick 以 Browser、动作 browser_click 审批。失败只放在 Err 里，不和成功文案拼在一起。没有浏览器或 EnsurePreview 失败也是错误。
func (t *WorkspaceTools) browserClick(sel string) ToolResult {
	if err := t.check(capability.Browser, "browser_click", sel, sel); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Click(sel)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserType 以 Browser、动作 browser_type 审批，再对选择器输入文本。错误放在 Err 里。
func (t *WorkspaceTools) browserType(sel, text string) ToolResult {
	if err := t.check(capability.Browser, "browser_type", sel, text); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Type(sel, text)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserFill 以 Browser、动作 browser_fill 审批，再填充选择器。错误放在 Err 里。
func (t *WorkspaceTools) browserFill(sel, text string) ToolResult {
	if err := t.check(capability.Browser, "browser_fill", sel, text); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Fill(sel, text)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserScroll 以 Browser、动作 browser_scroll 审批，再按方向、幅度和可选选择器滚动。
func (t *WorkspaceTools) browserScroll(direction string, amount int, sel string) ToolResult {
	if err := t.check(capability.Browser, "browser_scroll", sel, direction); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Scroll(direction, amount, sel)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserHover 以 Browser、动作 browser_hover 审批，再悬停到选择器。
func (t *WorkspaceTools) browserHover(sel string) ToolResult {
	if err := t.check(capability.Browser, "browser_hover", sel, sel); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Hover(sel)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserSelect 以 Browser、动作 browser_select 审批，再选中值。
func (t *WorkspaceTools) browserSelect(sel, value string) ToolResult {
	if err := t.check(capability.Browser, "browser_select", sel, value); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Select(sel, value)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserPress 以 Browser、动作 browser_press 审批，再按键。
func (t *WorkspaceTools) browserPress(key string) ToolResult {
	if err := t.check(capability.Browser, "browser_press", key, key); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	obs, err := t.Browser.Press(key)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: obs}
}

// browserTabs 以 Browser、动作 browser_tabs 审批。op 为空或 list 列出标签，switch、new、close 分别切换、新建、关闭。其他 op 返回错误。
func (t *WorkspaceTools) browserTabs(op, id, raw string) ToolResult {
	if err := t.check(capability.Browser, "browser_tabs", id, op); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "", "list":
		tabs, err := t.Browser.ListTabs()
		if err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: formatBrowserTabs(tabs)}
	case "switch":
		if err := t.Browser.SwitchTab(id); err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: "switched tab " + id}
	case "new":
		if err := t.Browser.NewTab(raw); err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: "opened tab " + raw}
	case "close":
		if err := t.Browser.CloseTab(id); err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: "closed tab " + id}
	default:
		return ToolResult{Err: fmt.Errorf("browser_tabs op %q must be list, switch, new, or close", op)}
	}
}

// formatBrowserTabs 列出标签数量。当前标签行首是星号，其余是空格，随后是 ID、URL 和标题。
func formatBrowserTabs(tabs []browser.TabInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d tab(s):\n", len(tabs))
	for _, tab := range tabs {
		mark := " "
		if tab.Active {
			mark = "*"
		}
		fmt.Fprintf(&b, "  %s %s  %s  %s\n", mark, tab.ID, tab.URL, tab.Title)
	}
	return strings.TrimSpace(b.String())
}

// browserProfile 以 Browser、动作 browser_profile 审批。op 为空或 list 列出配置并标出当前项。use 切换配置。其他 op 返回错误。
func (t *WorkspaceTools) browserProfile(op, name string) ToolResult {
	if err := t.check(capability.Browser, "browser_profile", name, op); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "", "list":
		return ToolResult{Content: "profiles: " + strings.Join(t.Browser.ListProfiles(), ", ") + "\ncurrent=" + t.Browser.ProfileName()}
	case "use":
		if err := t.Browser.UseProfile(name); err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: "using isolated profile " + t.Browser.ProfileName()}
	default:
		return ToolResult{Err: fmt.Errorf("browser_profile op %q must be list or use", op)}
	}
}

// browserRecord 以 Browser、动作 browser_record 审批。start 开始录制。status 或空 op 返回状态。stop 或 skill 停止并把会话交给 persistRecordedSkill。
func (t *WorkspaceTools) browserRecord(op, name string) ToolResult {
	if err := t.check(capability.Browser, "browser_record", name, op); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "start":
		id, err := t.Browser.StartRecord()
		if err != nil {
			return ToolResult{Err: err}
		}
		return ToolResult{Content: "recording " + id}
	case "status", "":
		return ToolResult{Content: t.Browser.RecordStatus()}
	case "stop", "skill":
		sess, err := t.Browser.StopRecord()
		if err != nil {
			return ToolResult{Err: err}
		}
		return t.persistRecordedSkill(sess, name)
	default:
		return ToolResult{Err: fmt.Errorf("browser_record op %q must be start, stop, status, or skill", op)}
	}
}

// persistRecordedSkill 在会话为空时返回 recording empty。有正文且能 resolve 到 .yoyo/skills/<名>/SKILL.md 时，以 0o700 建目录、0o600 写入，并放进 Skills。写失败则只把正文放在 Content 里。
func (t *WorkspaceTools) persistRecordedSkill(sess *browser.RecordSession, name string) ToolResult {
	if sess == nil {
		return ToolResult{Content: "recording empty"}
	}
	body := sess.SkillBody
	skillName := strings.TrimSpace(name)
	if skillName == "" {
		skillName = sess.SkillName
	}
	note := fmt.Sprintf("recorded %s: %d events → %d steps", sess.ID, len(sess.Events), len(sess.Steps))
	if sess.Dir != "" {
		note += "\nartifacts=" + sess.Dir
	}
	if body == "" {
		return ToolResult{Content: note + "\nno skill generated"}
	}
	if t.Workspace != "" && skillName != "" {
		rel := filepath.ToSlash(filepath.Join(".yoyo", "skills", skillName, "SKILL.md"))
		abs, err := t.resolve(rel)
		if err == nil {
			if mkErr := os.MkdirAll(filepath.Dir(abs), 0o700); mkErr == nil {
				if werr := os.WriteFile(abs, []byte(body), 0o600); werr == nil {
					if t.Skills == nil {
						t.Skills = map[string]string{}
					}
					t.Skills[skillName] = body
					note += "\nskill=" + rel + " — load_skill " + skillName
					return ToolResult{Content: note + "\n\n" + body, FileChange: &FileChange{Paths: []string{rel}, Created: true}}
				}
			}
		}
	}
	return ToolResult{Content: note + "\n\n" + body}
}

// browserDownload 先 resolve 目标路径，再以 Browser、动作 browser_download 审批。没有浏览器则返回错误。
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

// clipboardRead 以 Clipboard、动作 clipboard_read 审批。正文用 capText 截到 4000 个 rune，并标成 untrusted。
func (t *WorkspaceTools) clipboardRead() ToolResult {
	if err := t.check(capability.Clipboard, "clipboard_read", "", ""); err != nil {
		return ToolResult{Err: err}
	}
	s, err := osutil.ClipboardRead()
	if err != nil {
		return ToolResult{Err: err}
	}
	capped, _ := capText(s, 4000)
	return ToolResult{Content: "clipboard (untrusted):\n" + capped}
}

// clipboardWrite 以 Clipboard、动作 clipboard_write 审批，再写入系统剪贴板。
func (t *WorkspaceTools) clipboardWrite(text string) ToolResult {
	if err := t.check(capability.Clipboard, "clipboard_write", "", text); err != nil {
		return ToolResult{Err: err}
	}
	if err := osutil.ClipboardWrite(text); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "clipboard updated"}
}

// screenshot 在 path 为空时用 .yoyo/captures/shot.png。能力级别是 ComputerUse，动作 screenshot_region，ForceAsk 为真。Computer 为空时说明只捕获虚拟显示器，不捕获主屏。成功后交给 viewImage。
func (t *WorkspaceTools) screenshot(rel string) ToolResult {
	if rel == "" {
		rel = filepath.ToSlash(filepath.Join(".yoyo", "captures", "shot.png"))
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	req := capability.Request{
		Level:     capability.ComputerUse,
		Action:    "screenshot_region",
		Path:      p,
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
		return ToolResult{Err: fmt.Errorf("screenshot_region captures the virtual display only; primary screen is never captured")}
	}
	if err := t.Computer.Capture(p); err != nil {
		return ToolResult{Err: err}
	}
	return t.viewImage(rel)
}

// fsBatch 把 from 改名到 to。两端都要 resolve 进工作区。能力级别是 WriteWorkspace，动作 fs_batch。用 os.Rename，不是复制。
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

// connectorRead 以 ReadConnector、动作 connector_read 审批。账户不在本会话授权里则拒绝。读到的条目再经 BoundView。
func (t *WorkspaceTools) connectorRead(account, query string) ToolResult {
	if err := t.check(capability.ReadConnector, "connector_read", account, query); err != nil {
		return ToolResult{Err: err}
	}
	if t.Connectors == nil {
		return ToolResult{Err: fmt.Errorf("no connectors")}
	}
	if !t.connectorAllowed(account) {
		return ToolResult{Err: fmt.Errorf("connector %s is not granted to this session", account)}
	}
	items, err := t.Connectors.Read(account, query)
	if err != nil {
		return ToolResult{Err: err}
	}
	kind := connector.Kind("")
	for _, a := range t.Connectors.List() {
		if a.ID == account {
			kind = a.Kind
			break
		}
	}
	items, err = connector.BoundView(kind, query, items)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.MarshalIndent(items, "", "  ")
	return ToolResult{Content: string(b)}
}

// connectorDraft 以 WriteConnector、动作 connector_draft 审批。账户必须已授权。草稿写入后返回，并说明发送还要 connector_send 和 send_as_you。本函数不发送。
func (t *WorkspaceTools) connectorDraft(args map[string]any) ToolResult {
	account, to, subject, body := str(args["account"]), str(args["to"]), str(args["subject"]), str(args["body"])
	if err := t.check(capability.WriteConnector, "connector_draft", account, to); err != nil {
		return ToolResult{Err: err}
	}
	if t.Connectors == nil {
		return ToolResult{Err: fmt.Errorf("no connectors")}
	}
	if !t.connectorAllowed(account) {
		return ToolResult{Err: fmt.Errorf("connector %s is not granted to this session", account)}
	}
	d := t.Connectors.Draft(account, to, subject, body)
	d.Cc = str(args["cc"])
	d.Bcc = str(args["bcc"])
	d.ThreadID = str(args["thread_id"])
	d.ReplyTo = str(args["reply_to"])
	d.Location = str(args["location"])
	d.TimeZone = str(args["time_zone"])
	d.Start = str(args["start"])
	d.End = str(args["end"])
	d.CalendarID = str(args["calendar_id"])
	d.AllDay = str(args["all_day"]) == "true" || args["all_day"] == true
	d.Attendees = anyStrings(args["attendees"])
	d.Attachments = anyStrings(args["attachments"])
	d.EventID = str(args["event_id"])
	d.TargetVersion = str(args["target_version"])
	d.Recurrence = str(args["recurrence"])
	d.Op = str(args["op"])
	if title := str(args["title"]); title != "" && d.Subject == "" {
		d.Subject = title
	}
	d = t.Connectors.PutDraft(d)
	b, _ := json.Marshal(d)
	return ToolResult{Content: "draft " + string(b) + " — send with connector_send after send_as_you approval"}
}

// connectorSend 的能力级别是 SendAsYou，动作 connector_send，ForceAsk 为真。有 Personal 时先 RecordChatSend，发送结束后 FinishChatSend。
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
	var propID string
	if t.Personal != nil {
		if d, ok := t.Connectors.GetDraft(id); ok {
			propID = t.Personal.RecordChatSend(d).ID
		}
	}
	d, err := t.Connectors.Send(id)
	if t.Personal != nil && propID != "" {
		t.Personal.FinishChatSend(propID, err)
	}
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.Marshal(d)
	return ToolResult{Content: "sent " + string(b)}
}

// computerAct 的能力级别是 ComputerUse，动作 computer_act，ForceAsk 为真。没有电脑宿主则返回错误。未注入且操作不是 record 时，正文前缀标明 inject pending。
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
	note := "virtual-display "
	if !ev.Injected && op != "record" {
		note = "virtual-display (inject pending) "
	}
	return ToolResult{Content: note + string(b)}
}

// browserScreenshot 以 Browser、动作 browser_screenshot 审批。path 为空时用 .yoyo/captures/browser.png。截图后 viewImage。像素未变时在正文前写 PIXELS_UNCHANGED，并说明不要为此再 browser_open。变了则写 PIXELS_CHANGED。
func (t *WorkspaceTools) browserScreenshot(rel string) ToolResult {
	if err := t.check(capability.Browser, "browser_screenshot", rel, ""); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.EnsurePreview(); err != nil {
		return ToolResult{Err: err}
	}
	if rel == "" {
		rel = filepath.ToSlash(filepath.Join(".yoyo", "captures", "browser.png"))
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.Browser.Screenshot(p); err != nil {
		return ToolResult{Err: err}
	}
	if t.OnCapture != nil {
		t.OnCapture("shot", rel, "browser screenshot")
	}
	res := t.viewImage(rel)
	unchanged, sum, n := t.Browser.ShotMeta()
	if unchanged {
		res.Content = fmt.Sprintf("PIXELS_UNCHANGED sha256=%s bytes=%d. Isolated Chrome was not reloaded. If you clicked, paint did not change — do not browser_open; inspect page JS or click another selector.\n%s", sum, n, res.Content)
	} else if sum != "" {
		res.Content = fmt.Sprintf("PIXELS_CHANGED sha256=%s bytes=%d\n%s", sum, n, res.Content)
	}
	return res
}

// browserTakeover 以 Browser、动作 browser_takeover 审批，然后 StartTakeover 并开始录制。提问为空时用登录、MFA 或验证码的默认句。优先用 Takeover，否则用 AskUser。都没有则只说明窗口在等待。录制出步骤时交给 persistRecordedSkill。
func (t *WorkspaceTools) browserTakeover(question string) ToolResult {
	if err := t.check(capability.Browser, "browser_takeover", "", question); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	if err := t.Browser.StartTakeover(); err != nil {
		return ToolResult{Err: err}
	}
	recID, _ := t.Browser.StartRecord()
	q := question
	if q == "" {
		q = "Complete login, MFA, or CAPTCHA in the Yoyo browser window, then confirm to hand control back."
	}
	ask := t.AskUser
	if t.Takeover != nil {
		ask = t.Takeover
	}
	if ask == nil {
		return ToolResult{Content: "headed browser is waiting for takeover; no ask_user hook"}
	}
	ans, err := ask(q)
	if err != nil {
		return ToolResult{Err: err}
	}
	out := "takeover complete: " + ans
	if recID != "" {
		if sess, stopErr := t.Browser.StopRecord(); stopErr == nil && sess != nil && len(sess.Steps) > 0 {
			got := t.persistRecordedSkill(sess, "")
			out += "\n" + got.Content
			return ToolResult{Content: out, FileChange: got.FileChange}
		}
	}
	return ToolResult{Content: out}
}

// browserCookies 以 Browser、动作 browser_cookies 审批，再把 resolve 后的文件导入隔离配置，并返回导入条数。
func (t *WorkspaceTools) browserCookies(rel string) ToolResult {
	if err := t.check(capability.Browser, "browser_cookies", rel, ""); err != nil {
		return ToolResult{Err: err}
	}
	if t.Browser == nil {
		return ToolResult{Err: fmt.Errorf("no isolated browser")}
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	n, err := t.Browser.ImportCookies(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: fmt.Sprintf("imported %d cookies into isolated profile", n)}
}

// projectList 在 Projects 为空时返回 []。否则返回项目列表的 JSON。
func (t *WorkspaceTools) projectList() ToolResult {
	if t.Projects == nil {
		return ToolResult{Content: "[]"}
	}
	b, _ := json.MarshalIndent(t.Projects.List(), "", "  ")
	return ToolResult{Content: string(b)}
}

// notifyActionable 在 Inbox 为空时只在正文里注明标题。否则推一条 KindAsk，返回收件编号。
func (t *WorkspaceTools) notifyActionable(title, body string) ToolResult {
	if t.Inbox == nil {
		return ToolResult{Content: "inbox unavailable; noted " + title}
	}
	it := t.Inbox.Push(inbox.Item{Kind: inbox.KindAsk, Title: title, Body: body, SessionID: t.SessionID})
	return ToolResult{Content: "inbox " + it.ID}
}

// encodeDataURL 读取文件并做成 data URL。超过 2<<20 字节的部分丢掉。扩展名决定 jpeg、gif、webp，其余按 png。
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
