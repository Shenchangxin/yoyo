package connector

import (
	"fmt"
	"runtime"
	"strings"
)

func isPhone(provider string) bool {
	entry, ok := entryBy(provider)
	if !ok || entry.Access == "local" || entry.Access == "oauth" || entry.Access == "source" {
		return false
	}
	switch entry.Platform {
	case "ios", "android":
		return true
	}
	switch provider {
	case "mac-calendar", "mac-contacts", "mac-mail", "mac-notes", "mac-reminders", "mac-operate", "linux-operate":
		return true
	default:
		return false
	}
}

// readPhone 读手机同步过来的条目。在对应系统上，操作电脑就是本机命令行。
func (b *Broker) readPhone(provider string) ([]map[string]string, bool, error) {
	if !isPhone(provider) {
		return nil, false, nil
	}
	if provider == "mac-operate" && runtime.GOOS == "darwin" || provider == "linux-operate" && runtime.GOOS == "linux" {
		return []map[string]string{{"provider": provider, "title": "操作这台电脑", "text": "命令仍然要你批准"}}, true, nil
	}
	b.mu.Lock()
	items := append([]map[string]string(nil), b.phone[provider]...)
	b.mu.Unlock()
	if len(items) == 0 {
		return nil, true, fmt.Errorf("connector: 手机还没有把 %s 同步过来", provider)
	}
	return items, true, nil
}

// PutPhone 记下手机同步来的条目。下一次读取会用到。
func (b *Broker) PutPhone(provider string, items []map[string]string) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return
	}
	b.mu.Lock()
	if b.phone == nil {
		b.phone = map[string][]map[string]string{}
	}
	b.phone[provider] = append([]map[string]string(nil), items...)
	b.mu.Unlock()
}
