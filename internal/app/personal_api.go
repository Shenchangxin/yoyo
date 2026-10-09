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

func (a *App) InboxMarkRead(id string) {
	if a.Inbox != nil {
		a.Inbox.MarkRead(id)
	}
}

func (a *App) MemoryForget(id string) bool {
	if a.Memory == nil {
		return false
	}
	return a.Memory.Forget(id)
}

func (a *App) MemoryWrite(kind, text, project string) memory.Item {
	if a.Memory == nil {
		return memory.Item{}
	}
	return a.Memory.Write(memory.Item{Kind: memory.Kind(kind), Text: text, Project: project})
}

func (a *App) ConnectorCatalog() []connector.CatalogEntry {
	return connector.Catalog()
}

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

func (a *App) ConnectorStoreToken(id, token string) error {
	if token == "" || id == "" {
		return fmt.Errorf("connector: missing token")
	}
	a.Vault.Set("connector."+id, token)
	return nil
}

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

func (a *App) AdmitExpert(dir string) (expert.Pack, error) {
	return expert.Admit(dir, a.Home.Skills(), a.wasmPub)
}

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
		"url":        v.URL,
		"title":      v.Title,
		"lane":       v.Lane,
		"profile":    v.Profile,
		"headed":     v.Headed,
		"live":       v.Live,
		"text":       v.Text,
		"screenshot": v.Screenshot,
		"preview":    preview,
		"log":        v.Log,
	}
}

func (a *App) BrowserTakeover() error {
	return a.BrowserTakeoverAt("")
}

func (a *App) BrowserTakeoverAt(rel string) error {
	if a == nil || a.Browser == nil {
		return fmt.Errorf("no isolated browser")
	}
	if abs := a.resolveTakeoverFile(rel); abs != "" {
		return a.Browser.StartTakeoverAt(abs)
	}
	return a.Browser.StartTakeover()
}

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

func (a *App) takeoverFileAllowed(abs, preview string) bool {
	abs = filepath.Clean(abs)
	if preview != "" && filepath.Clean(preview) == abs {
		return true
	}
	if filepath.IsAbs(abs) {
		// Inspector already showed this file; the session workspace is often
		// not Config.Workspace, so an existing absolute path is the page to open.
		return true
	}
	for _, root := range a.takeoverRoots() {
		if capability.WithinWorkspace(root, abs) {
			return true
		}
	}
	return false
}

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

func (a *App) PhoneApprove(id, decision string) error {
	return a.ResolveApproval(id, decision)
}

func (a *App) PhoneSteer(sessionID, text string) error {
	if text == "" {
		return fmt.Errorf("empty steer")
	}
	a.Steer(sessionID, text)
	return nil
}

func (a *App) PhoneSchedule(j schedule.Job) schedule.Job {
	return a.ScheduleCreate(j)
}

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
