// 已加载技能正文的截断规则，以及前缀更新消息的开头。

package runtime

import (
	"strings"
	"unicode/utf8"

	"github.com/zeebo/blake3"
)

const (
	// skillBodyCapRunes 是单份非引导技能正文的 rune 上限。
	skillBodyCapRunes = 5_000
	// skillBootstrapCapRunes 是带引导标记的技能正文 rune 上限。
	skillBootstrapCapRunes = 12_000
	// skillTotalCapRunes 是全部技能正文加在一起的 rune 上限。放不下的整份跳过。
	skillTotalCapRunes = 25_000
)

// promptUpdateLead 是追加在历史后面的前缀更新开头。旧的系统消息留在头部，模型以这一版为准。isControlUser 认得这个前缀。
const promptUpdateLead = "Prompt update. The earlier system message stays only so the cached prefix can be reused. Follow this version.\n提示更新。前面的系统消息留在原处是为了保住缓存前缀。以这一版为准。\n"

// capLoadedSkills 截断已加载技能。带引导标记的单份上限是 skillBootstrapCapRunes，其余是 skillBodyCapRunes，总和不超过 skillTotalCapRunes。
// 引导正文按原顺序先占额度。其余从最新往旧收，输出时较新的仍在后面。放不下的整份跳过。
func capLoadedSkills(bodies []string) []string {
	if len(bodies) == 0 {
		return nil
	}
	var pinned []string
	var rest []string
	for _, body := range bodies {
		if isBootstrapBody(body) {
			pinned = append(pinned, capRunes(body, skillBootstrapCapRunes))
		} else {
			rest = append(rest, body)
		}
	}
	used := 0
	var keepPinned []string
	for _, body := range pinned {
		n := utf8.RuneCountInString(body)
		if used+n > skillTotalCapRunes {
			continue
		}
		used += n
		keepPinned = append(keepPinned, body)
	}
	var newest []string
	for i := len(rest) - 1; i >= 0; i-- {
		body := capRunes(rest[i], skillBodyCapRunes)
		n := utf8.RuneCountInString(body)
		if used+n > skillTotalCapRunes {
			continue
		}
		used += n
		newest = append(newest, body)
	}
	out := append([]string{}, keepPinned...)
	for i := len(newest) - 1; i >= 0; i-- {
		out = append(out, newest[i])
	}
	return out
}

// isBootstrapBody 在正文含引导标记标签时返回真。
func isBootstrapBody(s string) bool {
	return strings.Contains(s, "<"+bootstrapMarker+">")
}

// hotPrefixHash 对热前缀做 blake3，取前 16 字节的十六进制。动态块不计入。
func hotPrefixHash(msgs []Message) string {
	h := blake3.New()
	_, _ = h.Write([]byte(hotPrefixBytes(msgs)))
	return hex16(h.Sum(nil))
}

// hotPrefixBytes 拼热前缀。跳过 RoleMemory 和以 dynMarker 开头的内容，因此动态块不在这段字节里。
func hotPrefixBytes(msgs []Message) string {
	var b strings.Builder
	for _, m := range msgs {
		if m.Role == RoleMemory || strings.HasPrefix(m.Content, dynMarker) {
			continue
		}
		b.WriteString(string(m.Role))
		b.WriteByte(0)
		b.WriteString(m.Content)
		b.WriteByte(0)
		for _, tc := range m.ToolCalls {
			b.WriteString(tc.ID)
			b.WriteByte(0)
			b.WriteString(tc.Name)
			b.WriteByte(0)
			b.WriteString(tc.Arguments)
			b.WriteByte(0)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
