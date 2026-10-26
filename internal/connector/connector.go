// Package connector 把账号和草稿写进目录下的 connectors.json。账号上的 VaultKey 是钥匙名字，为空时 Connect 写成 connector.<id>。
// SaveToken 把令牌 JSON 交给 TokenStore，不写入这份 JSON。草稿的正文会写进 JSON。
package connector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Kind 是账号种类。
type Kind string

const (
	// KindMail 是邮件，值为 mail。
	KindMail Kind = "mail"
	// KindCalendar 是日历，值为 calendar。
	KindCalendar Kind = "calendar"
	// KindDrive 是网盘或文档，值为 drive。
	KindDrive Kind = "drive"
	// KindIM 是即时消息，值为 im。
	KindIM Kind = "im"
)

// Account 是一个已连接账号。VaultKey 是钥匙名字，不是令牌原文。
type Account struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Provider  string    `json:"provider"`
	Label     string    `json:"label"`
	Endpoint  string    `json:"endpoint,omitempty"`
	VaultKey  string    `json:"vault_key,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Draft 是一封待发邮件或一条待写日程。正文保存在连接器 JSON 里。
type Draft struct {
	ID            string   `json:"id"`
	Account       string   `json:"account"`
	To            string   `json:"to"`
	Subject       string   `json:"subject"`
	Body          string   `json:"body"`
	Sent          bool     `json:"sent"`
	Cc            string   `json:"cc,omitempty"`
	Bcc           string   `json:"bcc,omitempty"`
	ThreadID      string   `json:"thread_id,omitempty"`
	ReplyTo       string   `json:"reply_to,omitempty"`
	Attachments   []string `json:"attachments,omitempty"`
	Location      string   `json:"location,omitempty"`
	TimeZone      string   `json:"time_zone,omitempty"`
	AllDay        bool     `json:"all_day,omitempty"`
	Start         string   `json:"start,omitempty"`
	End           string   `json:"end,omitempty"`
	Attendees     []string `json:"attendees,omitempty"`
	CalendarID    string   `json:"calendar_id,omitempty"`
	EventID       string   `json:"event_id,omitempty"`
	TargetVersion string   `json:"target_version,omitempty"`
	Recurrence    string   `json:"recurrence,omitempty"`
	Op            string   `json:"op,omitempty"`
}

// Broker 保存账号、草稿和本地样例。令牌放在 tokens，不放进 path 指向的 JSON。
type Broker struct {
	mu     sync.Mutex
	path   string
	acct   []Account
	drafts []Draft
	local  map[string][]map[string]string
	tokens TokenStore
	http   *http.Client
}

// Open 按 0o755 创建目录，路径为 dir/connectors.json。读文件失败时仍返回空的 Broker。
func Open(dir string) (*Broker, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	b := &Broker{path: filepath.Join(dir, "connectors.json"), local: map[string][]map[string]string{}}
	_ = b.load()
	return b, nil
}

// Connect 追加账号。ID 为空时用供应商、种类和时分秒拼出。VaultKey 为空时写成 connector.<id>。
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

// List 返回账号切片的副本。
func (b *Broker) List() []Account {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Account(nil), b.acct...)
}

// Disconnect 按 ID 删掉账号并 flush。找到则返回真。它不清理 TokenStore。
// id 用来比较账号的 ID，相等的条目删掉。
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

// Read 先尝试 liveRead。失败则在本地样例里按查询做子串匹配。仍没有则返回一条说明没有令牌也没有本地项的占位记录。
// id 传给 find，并用来取 local。找不到时错误文案带上 id。
// query 传给 liveRead。失败后转小写，空则保留全部本地项，否则要求 subject、body、title 拼起来的小写文本包含它。占位记录里也会写入 query。
func (b *Broker) Read(id, query string) ([]map[string]string, error) {
	b.mu.Lock()
	a, ok := b.find(id)
	local := b.local[id]
	b.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown %s", id)
	}
	if live, err := b.liveRead(a, query); err == nil {
		return live, nil
	}
	q := strings.ToLower(query)
	var out []map[string]string
	for _, it := range local {
		blob := strings.ToLower(it["subject"] + " " + it["body"] + " " + it["title"])
		if q == "" || strings.Contains(blob, q) {
			out = append(out, it)
		}
	}
	if len(out) == 0 {
		out = append(out, map[string]string{
			"provider": a.Provider,
			"kind":     string(a.Kind),
			"note":     "no live token and no local items; complete OAuth or seed the mailbox",
			"query":    query,
		})
	}
	return out, nil
}

// Draft 追加一条未发送草稿并 flush。ID 用当前 UTC 时间拼出。
// account 写入草稿的 Account。
// to 写入草稿的 To。
// subject 写入草稿的 Subject。
// body 写入草稿的 Body。
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

// Send 在 LooksExfilDraft 命中时返回错误，不标记已发送。带 recurrence 时返回错误。供应商不是 local 且非空时调用 liveSend。
// id 用来比较草稿的 ID。找不到时返回 connector: unknown draft。发送成功后仍用 id 确认同一条再标成已发送。
func (b *Broker) Send(id string) (Draft, error) {
	b.mu.Lock()
	var d Draft
	var acct Account
	idx := -1
	for i := range b.drafts {
		if b.drafts[i].ID != id {
			continue
		}
		if LooksExfilDraft(b.drafts[i].To, b.drafts[i].Body) {
			b.mu.Unlock()
			return Draft{}, fmt.Errorf("connector: no-exfil blocked send")
		}
		d = b.drafts[i]
		acct, _ = b.find(d.Account)
		idx = i
		break
	}
	b.mu.Unlock()
	if idx < 0 {
		return Draft{}, fmt.Errorf("connector: unknown draft")
	}
	if strings.TrimSpace(d.Recurrence) != "" {
		return Draft{}, fmt.Errorf("connector: recurring series must be edited in the calendar app (422)")
	}
	local := strings.EqualFold(acct.Provider, "local") || acct.Provider == ""
	if !local {
		if err := b.liveSend(acct, d); err != nil {
			return Draft{}, err
		}
	}
	b.mu.Lock()
	if idx < len(b.drafts) && b.drafts[idx].ID == id {
		b.drafts[idx].Sent = true
		d = b.drafts[idx]
	}
	_ = b.flush()
	b.mu.Unlock()
	return d, nil
}

// PutDraft 按 ID 替换草稿，没有则追加。ID 为空时新生成一个。
// d 的 ID 为空时用当前 UTC 时间生成。已有相同 ID 则整条替换，否则追加。
func (b *Broker) PutDraft(d Draft) Draft {
	b.mu.Lock()
	defer b.mu.Unlock()
	if d.ID == "" {
		d.ID = time.Now().UTC().Format("draft-150405.000000000")
	}
	for i := range b.drafts {
		if b.drafts[i].ID == d.ID {
			b.drafts[i] = d
			_ = b.flush()
			return d
		}
	}
	b.drafts = append(b.drafts, d)
	_ = b.flush()
	return d
}

// GetDraft 按 ID 查找草稿。
// id 用来比较草稿的 ID。
func (b *Broker) GetDraft(id string) (Draft, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, d := range b.drafts {
		if d.ID == id {
			return d, true
		}
	}
	return Draft{}, false
}

// Drafts 返回草稿切片的副本。
func (b *Broker) Drafts() []Draft {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Draft(nil), b.drafts...)
}

// Seed 替换该账号的本地样例，不写进 JSON。
// id 当作 local 的键。
// items 整份写入该键，替换原有样例。
func (b *Broker) Seed(id string, items []map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.local[id] = items
}

// find 按 ID 在内存账号里查找。调用方需已处理锁。
// id 用来比较账号的 ID。
func (b *Broker) find(id string) (Account, bool) {
	for _, a := range b.acct {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

// load 从 connectors.json 读出账号和草稿。
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

// flush 把账号和草稿写成缩进 JSON，先写临时文件，权限 0o600，再改名替换。
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
