package runtime

import "encoding/json"

// AssistantStreamRecord is the lossless compact form of one attempt's chunks.
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

type packedRun struct {
	kind     string // text-chunks | reasoning-chunks | tool-call-chunks | chunk
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

// AssistantStreamAccumulator packs consecutive same-index deltas without
// retaining a second raw-chunk list.
type AssistantStreamAccumulator struct {
	runs []packedRun
}

func NewStreamAccumulator() *AssistantStreamAccumulator {
	return &AssistantStreamAccumulator{}
}

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

// ExpandAssistantStream reconstitutes timed chunks from compact records.
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

type timedChunk struct {
	Time  int64
	Chunk StreamChunk
}

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
