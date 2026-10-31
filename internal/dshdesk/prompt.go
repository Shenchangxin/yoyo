// prompt.go 把这一轮的提示碎片收成一段系统提示。
// 不调用模型。Host 在创建代理时把它交给 dsh.Options.Prompt。

package dshdesk

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// PromptText 按顺序拼接 frags 的正文，再接上 profile 和 page。空段跳过，段与段之间空一行。
func PromptText(frags []artifact.PromptFragment, profile, page string) string {
	var b strings.Builder
	write := func(text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(text)
	}
	for _, frag := range frags {
		write(frag.Text)
	}
	write(profile)
	write(page)
	return b.String()
}
