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
	return map[string]any{
		"url":        v.URL,
		"title":      v.Title,
		"lane":       v.Lane,
		"profile":    v.Profile,
		"headed":     v.Headed,
		"live":       v.Live,
		"text":       v.Text,
		"screenshot": v.Screenshot,
		"log":        v.Log,
	}
}

func (a *App) BrowserTakeover() error {
	if a == nil || a.Browser == nil {
		return fmt.Errorf("no isolated browser")
	}
	return a.Browser.StartTakeover()
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
	return map[string]any{
		"approvals": pending,
		"offers":    offers,
		"drafts":    drafts,
		"browser":   browser,
		"computer":  recording,
		"inbox":     a.InboxList(),
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
