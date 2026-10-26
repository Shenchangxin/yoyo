// 本文件判断验证码页和过薄的正文。抽样只看前 4000 个 rune。

package webcontent

import (
	"strings"
	"unicode/utf8"
)

// LooksAntibot 判断正文是不是高置信度的拦截页。只检查前 4000 个 rune 的小写文本。
// 同时含 warning: 和 requiring captcha 为真。同时含 just a moment 以及 cloudflare、cf- 或 challenge 为真。
// 含 attention required! | cloudflare、/cdn-cgi/challenge-platform/，或 enable javascript and cookies to continue 也为真。
func LooksAntibot(body string) bool {
	sample := body
	if utf8.RuneCountInString(sample) > 4000 {
		sample = string([]rune(sample)[:4000])
	}
	lower := strings.ToLower(sample)
	if strings.Contains(lower, "warning:") && strings.Contains(lower, "requiring captcha") {
		return true
	}
	if strings.Contains(lower, "just a moment") && (strings.Contains(lower, "cloudflare") || strings.Contains(lower, "cf-") || strings.Contains(lower, "challenge")) {
		return true
	}
	if strings.Contains(lower, "attention required! | cloudflare") {
		return true
	}
	if strings.Contains(lower, "/cdn-cgi/challenge-platform/") {
		return true
	}
	if strings.Contains(lower, "enable javascript and cookies to continue") {
		return true
	}
	return false
}

// ThinExtract 判断去掉标签后的文本够不够当文章。LooksAntibot(raw) 为真、stripped 空白，或 stripped 少于 200 个 rune 时为真。
// raw 长于 4000 字节且 stripped 少于 400 个 rune 时也为真。
func ThinExtract(stripped, raw string) bool {
	if LooksAntibot(raw) {
		return true
	}
	s := strings.TrimSpace(stripped)
	if s == "" {
		return true
	}
	if utf8.RuneCountInString(s) < 200 {
		return true
	}
	if len(raw) > 4000 && utf8.RuneCountInString(s) < 400 {
		return true
	}
	return false
}
