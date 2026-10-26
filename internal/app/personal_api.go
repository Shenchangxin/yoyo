// 收件箱、记忆、连接器令牌、专家目录、浏览器接管和隔离日程任务。接管文件用绝对路径或预览路径，再经 takeoverFileAllowed 过滤。

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/expert"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// InboxMarkRead 把一条收件箱标为已读。收件箱为空则返回。
func (a *App) InboxMarkRead(id string) {
	if a.Inbox != nil {
		a.Inbox.MarkRead(id)
	}
}

// MemoryForget 忘掉一条记忆。记忆库为空返回 false。
func (a *App) MemoryForget(id string) bool {
	if a.Memory == nil {
		return false
	}
	return a.Memory.Forget(id)
}

// MemoryWrite 按种类、文本和项目写入一条记忆。记忆库为空返回零值。
func (a *App) MemoryWrite(kind, text, project string) memory.Item {
	if a.Memory == nil {
		return memory.Item{}
	}
	return a.Memory.Write(memory.Item{Kind: memory.Kind(kind), Text: text, Project: project})
}

// ConnectorCatalog 返回连接器目录。
func (a *App) ConnectorCatalog() []connector.CatalogEntry {
	return connector.Catalog()
}

// ConnectorAuthURL 返回授权链接和 state。连接器为空返回 no connectors。
func (a *App) ConnectorAuthURL(provider, clientID, redirect string) (map[string]any, error) {
	if a.Connectors == nil {
		return nil, fmt.Errorf("no connectors")
	}
	u, state, err := a.Connectors.AuthURL(provider, clientID, redirect)
	if err != nil {
		return nil, err
	}
	return map[string]any{"url": u, "state": state}, nil
}

// ConnectorStoreToken 把令牌写进 vault 的 connector.<id>。id 或令牌为空返回 connector: missing token。
func (a *App) ConnectorStoreToken(id, token string) error {
	if token == "" || id == "" {
		return fmt.Errorf("connector: missing token")
	}
	a.Vault.Set("connector."+id, token)
	return nil
}

// ConnectorComplete 用授权码换令牌并 Connect 一个 KindMail 账号。client id 空则读 YOYO_<PROVIDER>_CLIENT_ID。secret 先读 vault 的 connector.<provider>.secret，再读 YOYO_<PROVIDER>_CLIENT_SECRET。redirect 空则用 http://127.0.0.1:3080/oauth。
func (a *App) ConnectorComplete(provider, code, clientID, secret, redirect string) (connector.Account, error) {
	var zero connector.Account
	if a.Connectors == nil {
		return zero, fmt.Errorf("no connectors")
	}
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("YOYO_" + strings.ToUpper(provider) + "_CLIENT_ID"))
	}
	if secret == "" {
		secret, _ = a.Vault.Lease("connector." + provider + ".secret")
		if secret == "" {
			secret = strings.TrimSpace(os.Getenv("YOYO_" + strings.ToUpper(provider) + "_CLIENT_SECRET"))
		}
	}
	if redirect == "" {
		redirect = "http://127.0.0.1:3080/oauth"
	}
	tok, err := a.Connectors.Exchange(provider, code, clientID, secret, redirect)
	if err != nil {
		return zero, err
	}
	acct := a.Connectors.Connect(connector.Account{Provider: provider, Kind: connector.KindMail, Label: provider})
	if err := a.Connectors.SaveToken(acct.VaultKey, tok); err != nil {
		return acct, err
	}
	return acct, nil
}

// SetSessionConnectors 把账号名单写成该会话的连接器允许列表，并返回会话元数据。
func (a *App) SetSessionConnectors(id string, accounts []string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	if a.Threads != nil {
		a.Threads.SetConnectorAllow(id, accounts)
	}
	return m, nil
}

// TriggerWebhook 触发一条日程后在后台跑 runIsolatedJob。找不到任务返回 unknown webhook job。
func (a *App) TriggerWebhook(id string) error {
	if a.Schedule == nil {
		return fmt.Errorf("no scheduler")
	}
	j, ok := a.Schedule.Trigger(id)
	if !ok {
		return fmt.Errorf("unknown webhook job")
	}
	go a.runIsolatedJob(j)
	return nil
}

// AdmitExpert 把目录交给 expert.Admit，公钥用 a.wasmPub。该公钥来自 Open 对 YOYO_WASM_PUBKEY 的解析，本函数不读环境变量。
func (a *App) AdmitExpert(dir string) (expert.Pack, error) {
	return expert.Admit(dir, a.Home.Skills(), a.wasmPub)
}

// DistillExpert 把蒸馏出的 Markdown 写到家目录技能夹下该名字的 SKILL.md，并返回路径。
func (a *App) DistillExpert(name, description, body string) (string, error) {
	md := expert.Distill(name, description, body)
	dir := filepath.Join(a.Home.Skills(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// BrowserView 返回当前浏览器视图。浏览器为空时 lane 为 isolated、live 为 false。预览文件能相对工作区时改成斜杠相对路径。
func (a *App) BrowserView() map[string]any {
	if a == nil || a.Browser == nil {
		return map[string]any{"lane": "isolated", "live": false, "log": []any{}}
	}
	v := a.Browser.View()
	preview := ""
	if abs := strings.TrimSpace(v.Preview); abs != "" {
		ws := a.Workspace()
		if ws != "" {
			if rel, err := filepath.Rel(ws, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				preview = filepath.ToSlash(rel)
			}
		}
		if preview == "" {
			preview = filepath.ToSlash(abs)
		}
	}
	return map[string]any{
		"url":         v.URL,
		"title":       v.Title,
		"lane":        v.Lane,
		"profile":     v.Profile,
		"profileName": v.ProfileName,
		"recording":   v.Recording,
		"tabCount":    v.TabCount,
		"headed":      v.Headed,
		"live":        v.Live,
		"text":        v.Text,
		"screenshot":  v.Screenshot,
		"preview":     preview,
		"log":         v.Log,
	}
}

// BrowserTakeover 调用 BrowserTakeoverAt，相对路径为空。
func (a *App) BrowserTakeover() error {
	return a.BrowserTakeoverAt("")
}

// BrowserTakeoverAt 能解析出文件时 StartTakeoverAt，否则 StartTakeover。浏览器为空返回 no isolated browser。
func (a *App) BrowserTakeoverAt(rel string) error {
	if a == nil || a.Browser == nil {
		return fmt.Errorf("no isolated browser")
	}
	if abs := a.resolveTakeoverFile(rel); abs != "" {
		return a.Browser.StartTakeoverAt(abs)
	}
	return a.Browser.StartTakeover()
}

// resolveTakeoverFile 依次尝试绝对路径、接管根下的相对路径和预览文件。候选必须存在、不是目录，并通过 takeoverFileAllowed。相对路径是预览路径后缀时也采用预览文件。
func (a *App) resolveTakeoverFile(rel string) string {
	rel = strings.TrimSpace(rel)
	preview := ""
	if a != nil && a.Browser != nil {
		preview = strings.TrimSpace(a.Browser.PreviewPath())
	}
	var cands []string
	if rel != "" {
		if filepath.IsAbs(rel) {
			cands = append(cands, filepath.Clean(rel))
		} else {
			for _, root := range a.takeoverRoots() {
				cands = append(cands, filepath.Join(root, rel))
			}
		}
	}
	if preview != "" {
		cands = append(cands, filepath.Clean(preview))
	}
	seen := map[string]bool{}
	for _, p := range cands {
		p = filepath.Clean(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		if a.takeoverFileAllowed(p, preview) {
			return p
		}
	}
	if rel != "" && preview != "" {
		slashP := filepath.ToSlash(preview)
		slashR := strings.TrimPrefix(filepath.ToSlash(rel), "./")
		if slashR != "" && strings.HasSuffix(slashP, slashR) {
			if st, err := os.Stat(preview); err == nil && !st.IsDir() {
				return filepath.Clean(preview)
			}
		}
	}
	return ""
}

// takeoverRoots 收集配置工作区和 Home.Workspace，按大小写去重。
func (a *App) takeoverRoots() []string {
	var roots []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		for _, e := range roots {
			if strings.EqualFold(e, p) {
				return
			}
		}
		roots = append(roots, p)
	}
	if a != nil {
		add(a.Workspace())
		if a.Home != nil {
			add(a.Home.Workspace())
		}
	}
	return roots
}

// takeoverFileAllowed 在路径等于预览文件时放行。绝对路径直接放行。其余路径要落在某个接管根里，判断用 capability.WithinWorkspace。
func (a *App) takeoverFileAllowed(abs, preview string) bool {
	abs = filepath.Clean(abs)
	if preview != "" && filepath.Clean(preview) == abs {
		return true
	}
	if filepath.IsAbs(abs) {
		// 检查器已经展示过这个文件。会话工作区常常不是 Config.Workspace，已存在的绝对路径就是要打开的页面。
		return true
	}
	for _, root := range a.takeoverRoots() {
		if capability.WithinWorkspace(root, abs) {
			return true
		}
	}
	return false
}

// ReviewQueue 汇总待审批、连接器草稿、浏览器日志、电脑录制、收件箱、awaiting_review 的提案，以及 waiting_input 或 waiting_approval 的任务。
func (a *App) ReviewQueue() map[string]any {
	drafts := []connector.Draft{}
	if a.Connectors != nil {
		drafts = a.Connectors.Drafts()
	}
	browser := []map[string]any{}
	if a.Browser != nil {
		browser = a.Browser.Log()
	}
	recording := a.ComputerRecording()
	pending := 0
	offers := []capability.Offer{}
	if a.Gate != nil {
		offers = a.Gate.Pending()
		pending = len(offers)
	}
	proposals := []any{}
	waiting := []any{}
	if a.Personal != nil {
		s := a.Personal.Snapshot()
		for _, p := range s.Proposals {
			if p.Status == "awaiting_review" {
				proposals = append(proposals, p)
			}
		}
		for _, t := range s.Tasks {
			if t.Status == "waiting_input" || t.Status == "waiting_approval" {
				waiting = append(waiting, t)
			}
		}
	}
	return map[string]any{
		"approvals": pending,
		"offers":    offers,
		"drafts":    drafts,
		"browser":   browser,
		"computer":  recording,
		"inbox":     a.InboxList(),
		"proposals": proposals,
		"waiting":   waiting,
	}
}

// PhoneApprove 把决定交给 ResolveApproval。
func (a *App) PhoneApprove(id, decision string) error {
	return a.ResolveApproval(id, decision)
}

// PhoneSteer 在文本非空时 Steer 该会话。空文本返回 empty steer。
func (a *App) PhoneSteer(sessionID, text string) error {
	if text == "" {
		return fmt.Errorf("empty steer")
	}
	a.Steer(sessionID, text)
	return nil
}

// PhoneSchedule 调用 ScheduleCreate。
func (a *App) PhoneSchedule(j schedule.Job) schedule.Job {
	return a.ScheduleCreate(j)
}

// runIsolatedJob 在 Isolate 为真且工作区非空时用 runtime.IsolateWorkspace 换目录，再 NewSession 并 StartSendOpts。失败记到 Schedule.Record。
func (a *App) runIsolatedJob(j schedule.Job) {
	ws := j.Workspace
	if ws == "" {
		ws = a.Workspace()
	}
	cleanup := func() {}
	if j.Isolate && ws != "" {
		var err error
		ws, cleanup, err = runtime.IsolateWorkspace(ws)
		if err != nil {
			a.Schedule.Record(j.ID, err)
			return
		}
	}
	defer cleanup()
	meta, err := a.NewSession(ws)
	if err != nil {
		a.Schedule.Record(j.ID, err)
		return
	}
	if a.Inbox != nil {
		a.Inbox.Push(inbox.Item{Kind: inbox.KindHeartbeat, Title: "Job " + j.ID, Body: j.Prompt, SessionID: meta.ID})
	}
	err = a.StartSendOpts(meta.ID, j.Prompt, false, nil)
	a.Schedule.Record(j.ID, err)
}
