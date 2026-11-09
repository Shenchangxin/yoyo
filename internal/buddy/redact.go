// redact.go 拥有 Buddy 日志和 stderr 的打码：在 diaglog.Redact 之外再抹掉已知的密钥值、Bearer 令牌、token= 和 key= 这类赋值，
// 以及协议帧里名字像密钥的 name/value 条目。
// 安装日志、探测日志、日志尾巴和错误详情在落盘或返回前都经过这里；外部工人的错误原因经导出的 Redact 走同一套规则。

package buddy

import (
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

var (
	// reBearerWord 匹配正文里的 Bearer 令牌。
	reBearerWord = regexp.MustCompile(`(?i)\b(bearer\s+)([A-Za-z0-9._~+/=-]{8,})`)
	// reQuerySecret 匹配 token=、key= 这类查询串或命令行赋值。
	reQuerySecret = regexp.MustCompile(`(?i)\b((?:access_|refresh_|id_)?token|key|api_?key|secret|password|auth)=([^\s&"']+)`)
	// reEnvSecret 匹配 FOO_API_KEY=xxx、GH_TOKEN=xxx 这类环境变量赋值。
	reEnvSecret = regexp.MustCompile(`\b([A-Z][A-Z0-9_]*(?:_KEY|_TOKEN|_SECRET|PASSWORD))=([^\s"']+)`)
	// reNameValue 匹配 ACP 和 MCP 协议里 {"name":"X","value":"..."} 形式的环境变量和请求头条目，例如交给桥的 YOYO_MCP_TOKEN。
	reNameValue = regexp.MustCompile(`("name"\s*:\s*"([^"\\]*)"\s*,\s*"value"\s*:\s*")((?:[^"\\]|\\.)*)(")`)
)

// Redact 是给包外用的打码，规则和 Buddy 自己的日志一样。工人的错误原因和登录详情用它。
func Redact(s string, secrets ...string) string { return redactText(s, secrets...) }

// redactText 打码 s。secrets 是已知的密钥值，至少 6 个字符的才替换，免得误伤普通词。
func redactText(s string, secrets ...string) string {
	if s == "" {
		return s
	}
	for _, v := range secrets {
		if len(v) >= 6 {
			s = strings.ReplaceAll(s, v, "[REDACTED]")
		}
	}
	s = diaglog.Redact(s)
	s = reBearerWord.ReplaceAllString(s, "${1}[REDACTED]")
	s = reQuerySecret.ReplaceAllString(s, "${1}=[REDACTED]")
	s = reEnvSecret.ReplaceAllString(s, "${1}=[REDACTED]")
	s = reNameValue.ReplaceAllStringFunc(s, func(m string) string {
		sub := reNameValue.FindStringSubmatch(m)
		if !SecretLike(sub[2]) {
			return m
		}
		return sub[1] + "[REDACTED]" + sub[4]
	})
	return s
}
