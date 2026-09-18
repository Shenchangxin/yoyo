package connector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindMail     Kind = "mail"
	KindCalendar Kind = "calendar"
	KindDrive    Kind = "drive"
	KindIM       Kind = "im"
)

type Account struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Provider  string    `json:"provider"`
	Label     string    `json:"label"`
	Endpoint  string    `json:"endpoint,omitempty"`
	VaultKey  string    `json:"vault_key,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Draft struct {
	ID      string `json:"id"`
	Account string `json:"account"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Sent    bool   `json:"sent"`
}

type Broker struct {
	mu      sync.Mutex
	path    string
	acct    []Account
	drafts  []Draft
	local   map[string][]map[string]string
}

func Open(dir string) (*Broker, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	b := &Broker{path: filepath.Join(dir, "connectors.json"), local: map[string][]map[string]string{}}
	_ = b.load()
	return b, nil
}

func (b *Broker) Connect(a Account) Account {
	b.mu.Lock()
	defer b.mu.Unlock()
	if a.ID == "" {
		a.ID = a.Provider + "-" + string(a.Kind) + "-" + time.Now().UTC().Format("150405")
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.VaultKey == "" {
		a.VaultKey = "connector." + a.ID
	}
	b.acct = append(b.acct, a)
	_ = b.flush()
	return a
}

func (b *Broker) List() []Account {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Account(nil), b.acct...)
}

func (b *Broker) Disconnect(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.acct[:0]
	ok := false
	for _, a := range b.acct {
		if a.ID == id {
			ok = true
			continue
		}
		out = append(out, a)
	}
	b.acct = out
	_ = b.flush()
	return ok
}

func (b *Broker) Read(id, query string) ([]map[string]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a, ok := b.find(id)
	if !ok {
		return nil, fmt.Errorf("connector: unknown %s", id)
	}
	items := b.local[id]
	q := strings.ToLower(query)
	var out []map[string]string
	for _, it := range items {
		blob := strings.ToLower(it["subject"] + " " + it["body"] + " " + it["title"])
		if q == "" || strings.Contains(blob, q) {
			out = append(out, it)
		}
	}
	if len(out) == 0 {
		out = append(out, map[string]string{
			"provider": a.Provider,
			"kind":     string(a.Kind),
			"note":     "no local items; connect a real MCP/OAuth pack to read live mail/calendar",
			"query":    query,
		})
	}
	return out, nil
}

func (b *Broker) Draft(account, to, subject, body string) Draft {
	b.mu.Lock()
	defer b.mu.Unlock()
	d := Draft{
		ID:      time.Now().UTC().Format("draft-150405.000000000"),
		Account: account,
		To:      to,
		Subject: subject,
		Body:    body,
	}
	b.drafts = append(b.drafts, d)
	_ = b.flush()
	return d
}

func (b *Broker) Send(id string) (Draft, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.drafts {
		if b.drafts[i].ID != id {
			continue
		}
		if LooksExfilDraft(b.drafts[i].To, b.drafts[i].Body) {
			return Draft{}, fmt.Errorf("connector: no-exfil blocked send")
		}
		b.drafts[i].Sent = true
		_ = b.flush()
		return b.drafts[i], nil
	}
	return Draft{}, fmt.Errorf("connector: unknown draft")
}

func (b *Broker) Drafts() []Draft {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Draft(nil), b.drafts...)
}

func (b *Broker) Seed(id string, items []map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.local[id] = items
}

func (b *Broker) find(id string) (Account, bool) {
	for _, a := range b.acct {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

func (b *Broker) load() error {
	raw, err := os.ReadFile(b.path)
	if err != nil {
		return err
	}
	var doc struct {
		Acct   []Account `json:"accounts"`
		Drafts []Draft   `json:"drafts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	b.acct = doc.Acct
	b.drafts = doc.Drafts
	return nil
}

func (b *Broker) flush() error {
	doc := struct {
		Acct   []Account `json:"accounts"`
		Drafts []Draft   `json:"drafts"`
	}{b.acct, b.drafts}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}
