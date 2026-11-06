package connector

import (
	"fmt"
	"strings"
	"time"
)

// UseResult 是连上之后读到的第一页。
type UseResult struct {
	Account Account             `json:"account"`
	Items   []map[string]string `json:"items,omitempty"`
}

// Use 按目录把一个应用连上并读一页。令牌进 TokenStore，不进 connectors.json。
func (b *Broker) Use(provider, token, endpoint string) (UseResult, error) {
	entry, ok := entryBy(provider)
	if !ok {
		return UseResult{}, fmt.Errorf("connector: 未知应用 %s", provider)
	}
	provider = entry.Provider
	if entry.Access == "source" {
		return UseResult{}, fmt.Errorf("connector: %s 在工作来源里配置", entry.Label)
	}
	switch provider {
	case "dingtalk", "wecom", "wechat":
		return UseResult{}, fmt.Errorf("connector: %s 在消息里配置机器人", entry.Label)
	}
	acct := b.upsert(Account{Provider: provider, Kind: entry.Kind, Label: entry.Label, Endpoint: strings.TrimSpace(endpoint)})
	if strings.TrimSpace(token) != "" {
		if err := b.SaveToken(acct.VaultKey, Token{AccessToken: strings.TrimSpace(token)}); err != nil {
			return UseResult{Account: acct}, err
		}
	}
	items, err := b.liveRead(acct, "")
	return UseResult{Account: acct, Items: items}, err
}

func (b *Broker) upsert(a Account) Account {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.acct {
		if strings.EqualFold(b.acct[i].Provider, a.Provider) {
			if a.Endpoint != "" {
				b.acct[i].Endpoint = a.Endpoint
			}
			if a.Label != "" {
				b.acct[i].Label = a.Label
			}
			_ = b.flush()
			return b.acct[i]
		}
	}
	if a.ID == "" {
		a.ID = a.Provider + "-" + string(a.Kind)
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
