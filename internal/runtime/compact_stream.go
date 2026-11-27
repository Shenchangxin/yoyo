// 把一次模型尝试的流式块压成可回放的记录，再展开成带时间的块。
// 循环在 chat 里 Push，结算时 Snapshot。不执行工具，也不改模型请求。

package runtime

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
