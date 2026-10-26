// 把一次模型尝试的流式块压成可回放的记录，再展开成带时间的块。
// 循环在 chat 里 Push，结算时 Snapshot。不执行工具，也不改模型请求。

package runtime

import "encoding/json"

// AssistantStreamRecord 是一次尝试里一块压缩记录，展开后能回到原来的块和时间。
type AssistantStreamRecord struct {
	Type  string       `json:"type"`
	Time0 int64        `json:"time0,omitempty"`
	Time  int64        `json:"time,omitempty"`
	Index int          `json:"index,omitempty"`
	DT    []int64      `json:"dt,omitempty"`
	Texts []string     `json:"texts,omitempty"`
	ID    string       `json:"id,omitempty"`
	Name  string       `json:"name,omitempty"`
	Args  []string     `json:"args,omitempty"`
	Chunk *StreamChunk `json:"chunk,omitempty"`
}

// packedRun 是累加器里尚未落成记录的一段。kind 是 text-chunks、reasoning-chunks、tool-call-chunks 或 chunk。
type packedRun struct {
	kind     string // text-chunks、reasoning-chunks、tool-call-chunks 或 chunk
	time0    int64
	lastTime int64
	index    int
	dt       []int64
	texts    []string
	id       string
	name     string
	hasName  bool
	args     []string
	chunk    StreamChunk
}

// AssistantStreamAccumulator 把同一块序号上连续的增量并成一段，不再另存一份原始块列表。
type AssistantStreamAccumulator struct {
	runs []packedRun
}

// NewStreamAccumulator 返回空的累加器。
func NewStreamAccumulator() *AssistantStreamAccumulator {
	return &AssistantStreamAccumulator{}
}

// Push 收下一块。文本和推理在上一段种类、序号都相同且时间差不小于 0 时并进去；时间倒退就新开一段。
// 工具参数增量在 id 或名字为空时整块留下。两者都非空时，上一段是同一序号、同一 id、同一名字且时间差不小于 0 才合并。其他类型整块留下。
// time 作为新段的起点，或与上一段 lastTime 相减得到间隔。
// chunk 按 Type 处理，并读取 Index、Text、ID、Name 和 ArgumentsDelta。
func (a *AssistantStreamAccumulator) Push(time int64, chunk StreamChunk) {
	switch chunk.Type {
	case ChunkTextDelta, ChunkReasoningDelta:
		kind := "text-chunks"
		if chunk.Type == ChunkReasoningDelta {
			kind = "reasoning-chunks"
		}
		if n := len(a.runs); n > 0 {
			prev := &a.runs[n-1]
			if prev.kind == kind && prev.index == chunk.Index {
				gap := time - prev.lastTime
				if gap >= 0 {
					prev.dt = append(prev.dt, gap)
					prev.texts = append(prev.texts, chunk.Text)
					prev.lastTime = time
					return
				}
			}
		}
		a.runs = append(a.runs, packedRun{
			kind:     kind,
			time0:    time,
			lastTime: time,
			index:    chunk.Index,
			texts:    []string{chunk.Text},
		})
	case ChunkToolCallDelta:
		if chunk.ID == "" || chunk.Name == "" {
			a.runs = append(a.runs, packedRun{kind: "chunk", time0: time, lastTime: time, chunk: chunk})
			return
		}
		if n := len(a.runs); n > 0 {
			prev := &a.runs[n-1]
			sameName := prev.hasName == (chunk.Name != "") && prev.name == chunk.Name
			if prev.kind == "tool-call-chunks" && prev.index == chunk.Index && prev.id == chunk.ID && sameName {
				gap := time - prev.lastTime
				if gap >= 0 {
					prev.dt = append(prev.dt, gap)
					prev.args = append(prev.args, chunk.ArgumentsDelta)
					prev.lastTime = time
					return
				}
			}
		}
		a.runs = append(a.runs, packedRun{
			kind:     "tool-call-chunks",
			time0:    time,
			lastTime: time,
			index:    chunk.Index,
			id:       chunk.ID,
			name:     chunk.Name,
			hasName:  chunk.Name != "",
			args:     []string{chunk.ArgumentsDelta},
		})
	default:
		a.runs = append(a.runs, packedRun{kind: "chunk", time0: time, lastTime: time, chunk: chunk})
	}
}

// Snapshot 把各段拷成记录。接收者为 nil 或还没有段时返回 nil。拷贝切片，之后的 Push 不会改到已返回的记录。名字只在 hasName 时写入。
func (a *AssistantStreamAccumulator) Snapshot() []AssistantStreamRecord {
	if a == nil || len(a.runs) == 0 {
		return nil
	}
	out := make([]AssistantStreamRecord, 0, len(a.runs))
	for _, r := range a.runs {
		switch r.kind {
		case "text-chunks", "reasoning-chunks":
			out = append(out, AssistantStreamRecord{
				Type:  r.kind,
				Time0: r.time0,
				Index: r.index,
				DT:    append([]int64(nil), r.dt...),
				Texts: append([]string(nil), r.texts...),
			})
		case "tool-call-chunks":
			rec := AssistantStreamRecord{
				Type:  r.kind,
				Time0: r.time0,
				Index: r.index,
				DT:    append([]int64(nil), r.dt...),
				ID:    r.id,
				Args:  append([]string(nil), r.args...),
			}
			if r.hasName {
				rec.Name = r.name
			}
			out = append(out, rec)
		default:
			ch := r.chunk
			out = append(out, AssistantStreamRecord{Type: "chunk", Time: r.time0, Chunk: &ch})
		}
	}
	return out
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

// timedChunk 是展开后的一块和它的时间。时间单位与 Push 传入的 time 相同。
type timedChunk struct {
	Time  int64
	Chunk StreamChunk
}

// recordsAsAny 把记录编成 JSON 再解成 []any。空切片、编码失败或解码失败时返回 nil。
func recordsAsAny(records []AssistantStreamRecord) []any {
	if len(records) == 0 {
		return nil
	}
	b, err := json.Marshal(records)
	if err != nil {
		return nil
	}
	var out []any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
