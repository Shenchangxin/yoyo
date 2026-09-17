package runtime

import (
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/zeebo/blake3"
)

// PromptCacheKey is a stable prefix hash: identity + pins + advertised tool
// schemas + harness. Dynamic notes/plan/skills must not be included.
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

func compactPrompt(fragments []artifact.PromptFragment) string {
	for _, f := range fragments {
		if f.Slot == "compact" && strings.TrimSpace(f.Text) != "" {
			return f.Text
		}
	}
	return compactSystem
}
