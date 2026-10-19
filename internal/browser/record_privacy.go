package browser

import (
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/safeguard"
)

var sensitiveField = []string{
	"password", "token", "secret", "key", "code", "otp", "auth", "session", "jwt",
}

var sensitiveQuery = []string{
	"token", "access_token", "refresh_token", "code", "state", "session",
	"key", "secret", "password", "auth", "jwt",
}

func looksSensitiveField(hay string) bool {
	low := strings.ToLower(hay)
	for _, k := range sensitiveField {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

func redactRecordValue(target RecordTarget, value string) string {
	hay := strings.Join([]string{
		target.Type, target.Name, target.NameAttr, target.ID, target.Placeholder, target.Label, target.Text,
	}, " ")
	if looksSensitiveField(hay) || looksSecretValue(value) {
		return "{{sensitive_value}}"
	}
	if len(value) > 2000 {
		return "{{redacted_value}}"
	}
	return value
}

func looksSecretValue(s string) bool {
	return safeguard.LooksSecretPaste(s)
}

func redactRecordURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return raw
	}
	q := u.Query()
	changed := false
	for _, k := range sensitiveQuery {
		if q.Has(k) {
			q.Set(k, "***")
			changed = true
		}
	}
	if !changed {
		return raw
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func skipReplayURL(raw string) bool {
	low := strings.ToLower(strings.TrimSpace(raw))
	if low == "" || low == "about:blank" {
		return true
	}
	if strings.HasPrefix(low, "chrome://") || strings.HasPrefix(low, "devtools://") || strings.HasPrefix(low, "edge://") {
		return true
	}
	return false
}
