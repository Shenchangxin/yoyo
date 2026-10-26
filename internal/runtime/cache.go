// 提示缓存键。只哈希调用方传入的 harness、前缀和工具 JSON。
// 动态笔记、计划和技能正文不在这三个参数里。压缩用的系统提示片段也在这里选取。

package runtime

import (
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/zeebo/blake3"
)

// PromptCacheKey 返回稳定前缀的十六进制摘要。写入顺序是 harnessHash、一个 0 字节、prefix、一个 0 字节、tools 的 JSON。摘要取 BLAKE3 结果的前 16 字节。
func PromptCacheKey(harnessHash string, prefix string, tools []ToolJSON) string {
	h := blake3.New()
	_, _ = h.Write([]byte(harnessHash))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write([]byte{0})
	raw, _ := json.Marshal(tools)
	_, _ = h.Write(raw)
	return hex16(h.Sum(nil))
}

// hex16 把 sum 编成十六进制。长度超过 16 字节时只取前 16 字节，因此最长 32 个十六进制字符。
func hex16(sum []byte) string {
	const hexd = "0123456789abcdef"
	if len(sum) > 16 {
		sum = sum[:16]
	}
	var b strings.Builder
	b.Grow(len(sum) * 2)
	for _, c := range sum {
		b.WriteByte(hexd[c>>4])
		b.WriteByte(hexd[c&0x0f])
	}
	return b.String()
}

// compactPrompt 返回第一个 Slot 为 compact 且正文非空白的片段。没有时返回 compactSystem。
func compactPrompt(fragments []artifact.PromptFragment) string {
	for _, f := range fragments {
		if f.Slot == "compact" && strings.TrimSpace(f.Text) != "" {
			return f.Text
		}
	}
	return compactSystem
}
