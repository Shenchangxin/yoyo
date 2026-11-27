// testfixtures_test.go 放只给测试用的辅助函数。
// 它们原先是生产代码里的导出函数，但没有任何生产调用方；挪到这里后实现原样不动，测试照旧调用。
// Compact 给整形测试一个只做确定性投影的入口；ExpandAssistantStream 用来验证流式压缩记录能还原；WrapBootstrap 构造带引导标记的技能正文。
// timedChunk 和 bootstrapAlready 只被这几个辅助函数用到，也一起放在这里。

package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// timedChunk 是展开后的一块和它的时间。时间单位与 AssistantStreamAccumulator.Push 传入的 time 相同。
type timedChunk struct {
	Time  int64
	Chunk StreamChunk
}

// bootstrapAlready 是 WrapBootstrap 塞进正文的一句说明，声明引导技能已经加载、不要再 load_skill。
const bootstrapAlready = "IMPORTANT: The using-superpowers (or pack bootstrap) skill content is included below. It is ALREADY LOADED — you are currently following it. Do NOT call load_skill for this bootstrap skill again."

// Compact 是评测用的稳定入口，只做确定性整形，不调用模型。返回投影和 Shape 的 Note。
func Compact(msgs []Message, loop artifact.LoopPreset) ([]Message, string) {
	out, rep := Shape(msgs, ShapeOpts{Loop: loop})
	return out, rep.Note
}

// ExpandAssistantStream 把压缩记录展开成带时间的块。第一片用 Time0，后面的片累加对应的 DT。工具记录的 Name 为空时，展开的块也不带名字。类型对不上的记录跳过。
// records 按 Type 展开，对不上的类型跳过。
func ExpandAssistantStream(records []AssistantStreamRecord) []timedChunk {
	var out []timedChunk
	for _, rec := range records {
		switch rec.Type {
		case "text-chunks", "reasoning-chunks":
			delta := ChunkTextDelta
			if rec.Type == "reasoning-chunks" {
				delta = ChunkReasoningDelta
			}
			t := rec.Time0
			for i, text := range rec.Texts {
				if i > 0 && i-1 < len(rec.DT) {
					t += rec.DT[i-1]
				}
				out = append(out, timedChunk{Time: t, Chunk: StreamChunk{Type: delta, Index: rec.Index, Text: text}})
			}
		case "tool-call-chunks":
			t := rec.Time0
			for i, arg := range rec.Args {
				if i > 0 && i-1 < len(rec.DT) {
					t += rec.DT[i-1]
				}
				ch := StreamChunk{Type: ChunkToolCallDelta, Index: rec.Index, ID: rec.ID, ArgumentsDelta: arg}
				if rec.Name != "" {
					ch.Name = rec.Name
				}
				out = append(out, timedChunk{Time: t, Chunk: ch})
			}
		case "chunk":
			if rec.Chunk != nil {
				out = append(out, timedChunk{Time: rec.Time, Chunk: *rec.Chunk})
			}
		}
	}
	return out
}

// WrapBootstrap 用 bootstrapMarker 包住说明、正文和 mapping。正文为空时写明该技能已启用但 SKILL.md 为空。ApplyPackBootstrap 不调用本函数。
// name 只在 body 去掉空白后为空时写进那句空正文说明。
// body 先去掉首尾空白，非空则写入标记内部。
func WrapBootstrap(name, body, mapping string) string {
	body = strings.TrimSpace(body)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(bootstrapMarker)
	b.WriteString(">\nYou have superpowers.\n\n")
	b.WriteString("**")
	b.WriteString(bootstrapAlready)
	b.WriteString("**\n\n")
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	} else {
		b.WriteString("Bootstrap skill ")
		b.WriteString(name)
		b.WriteString(" was enabled but its SKILL.md body was empty.\n")
	}
	if strings.TrimSpace(mapping) != "" {
		b.WriteByte('\n')
		b.WriteString(strings.TrimSpace(mapping))
		b.WriteByte('\n')
	}
	b.WriteString("</")
	b.WriteString(bootstrapMarker)
	b.WriteString(">\n")
	return b.String()
}
