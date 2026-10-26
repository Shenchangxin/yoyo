// 本文件在记下录制值之前打码。敏感字段或看起来像密钥的值换成 {{sensitive_value}}。
// 地址查询里的 token、code、session 等键改成 ***。chrome://、devtools://、edge:// 和 about:blank 不进入回放。

package browser

import (
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/safeguard"
)

// sensitiveField 是字段名里会出现的敏感词。这张表是数据。
var sensitiveField = []string{
	"password", "token", "secret", "key", "code", "otp", "auth", "session", "jwt",
}

// sensitiveQuery 是地址查询里要打码的键。这张表是数据。
var sensitiveQuery = []string{
	"token", "access_token", "refresh_token", "code", "state", "session",
	"key", "secret", "password", "auth", "jwt",
}

// looksSensitiveField 看拼起来的字段文本里有没有 sensitiveField 中的词。
func looksSensitiveField(hay string) bool {
	low := strings.ToLower(hay)
	for _, k := range sensitiveField {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// redactRecordValue 在字段名敏感或值像密钥时返回 {{sensitive_value}}。长于 2000 个字符时返回 {{redacted_value}}。其余原样返回。
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

// looksSecretValue 交给 safeguard.LooksSecretPaste 判断这段文字像不像密钥。
func looksSecretValue(s string) bool {
	return safeguard.LooksSecretPaste(s)
}

// redactRecordURL 把查询里 sensitiveQuery 那些键的值改成 ***。没有这些键或解析失败时原样返回。
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

// skipReplayURL 在空白、about:blank，或以 chrome://、devtools://、edge:// 开头时返回真。这些地址不写成回放步骤。
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
