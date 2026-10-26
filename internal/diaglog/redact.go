// 本文件在日志落盘前抹掉密钥、PEM 和赋值形式的令牌。
// 替换文本是 [REDACTED]。PayloadPreview 在调用方没有给正数上限时截到 512 个符文。

package diaglog

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// reBearer 匹配 Authorization 头以及后面的令牌，大小写不敏感。
	reBearer = regexp.MustCompile(`(?i)(authorization:\s*(?:bearer\s+)?)(\S+)`)
	// reSK 匹配以 sk- 开头、后面至少 8 个字母数字或下划线、连字符的词。
	reSK = regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9_-]{8,}`)
	// rePEM 匹配一对 BEGIN 和 END 的 PEM 块。
	rePEM = regexp.MustCompile(`-----BEGIN [A-Z ]+-----[\s\S]*?-----END [A-Z ]+-----`)
	// reJSONSecret 匹配 JSON 里名为 api_key、secret、password、token、search_key、authorization 的字符串值。
	reJSONSecret = regexp.MustCompile(`(?i)("(api[_-]?key|secret|password|token|search_key|authorization)"\s*:\s*")([^"]*)(")`)
	// reAssign 匹配 YOYO_API_KEY、OPENAI_API_KEY、ANTHROPIC_API_KEY、DEEPSEEK_API_KEY、API_KEY、SEARCH_KEY 的赋值。
	reAssign = regexp.MustCompile(`(?i)\b(YOYO_API_KEY|OPENAI_API_KEY|ANTHROPIC_API_KEY|DEEPSEEK_API_KEY|API_KEY|SEARCH_KEY)=([^\s]+)`)
)

// redacted 是替换后的占位文本 [REDACTED]。
const redacted = "[REDACTED]"

// Redact 抹掉 PEM、Bearer 令牌、sk- 密钥、JSON 秘密字段和环境变量赋值。空字符串原样返回。
// PEM 换成占位文本加 PEM，sk- 换成占位文本加 KEY。Bearer 和赋值保留前面的名字。
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = rePEM.ReplaceAllString(s, redacted+" PEM")
	s = reBearer.ReplaceAllString(s, "${1}"+redacted)
	s = reSK.ReplaceAllString(s, redacted+" KEY")
	s = reJSONSecret.ReplaceAllString(s, "${1}"+redacted+"${4}")
	s = reAssign.ReplaceAllString(s, "${1}="+redacted)
	return s
}

// RedactBytes 把字节当字符串脱敏后再转回字节。
// b 转成字符串交给 Redact，结果再转回字节。
func RedactBytes(b []byte) []byte {
	return []byte(Redact(string(b)))
}

// PayloadPreview 先脱敏，再把连续空白收成一个空格。maxRunes 小于等于 0 时用 512。
// 符文数不超过上限时原样返回，否则截断并在末尾加省略号。
func PayloadPreview(s string, maxRunes int) string {
	s = Redact(s)
	s = strings.Join(strings.Fields(s), " ")
	if maxRunes <= 0 {
		maxRunes = 512
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "…"
}

// LooksSecretKey 判断字段名像不像密钥。比较前转成小写并去掉空白。
// 精确命中 api_key、apikey、key、secret、password、token、authorization、search_key、searchkey 时为真。
// 否则名字里含 secret 或 password，或以后缀 _key 结尾时为真。
// name 先转成小写并去掉首尾空白，再做精确比较，以及 secret、password 包含和 _key 后缀判断。
func LooksSecretKey(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "api_key", "apikey", "key", "secret", "password", "token", "authorization", "search_key", "searchkey":
		return true
	default:
		return strings.Contains(n, "secret") || strings.Contains(n, "password") || strings.HasSuffix(n, "_key")
	}
}
