// 把流式块组装成 ContentBlock。允许只有增量、没有开始和结束帧的协议。
// 某一序号已被结束帧关闭后，再来的增量被忽略。loop 在读模型流时调用这里。

package runtime

import (
	"strconv"
	"strings"
)

// partialBlock 是一个尚未收口或已由结束帧定稿的块。
type partialBlock struct {
	blockType         string
	text              string
	toolCallID        string
	toolCallName      string
	toolCallArguments string
	block             *ContentBlock
}

// BlockAssembler 是从 StreamChunk 到 ContentBlock 的唯一组装算法。容忍只有增量的协议（没有块开始和结束）；某个序号已被结束帧关闭后再到的增量会被忽略。
type BlockAssembler struct {
	partials map[int]*partialBlock
	order    []int
	usage    *TokenUsage
	finish   *FinishReason
}

// NewBlockAssembler 返回空的组装器。
func NewBlockAssembler() *BlockAssembler {
	return &BlockAssembler{partials: map[int]*partialBlock{}}
}

// Push 吃进一块流。同一序号的开始帧只采纳第一次。结束帧之后该序号的增量被丢掉。用量和结束原因各保留最后一次。
// chunk 按 Type 处理开始帧、文本或推理增量、工具调用增量、结束帧、用量和结束原因。
func (a *BlockAssembler) Push(chunk StreamChunk) {
	if a.partials == nil {
		a.partials = map[int]*partialBlock{}
	}
	switch chunk.Type {
	case ChunkBlockStart:
		if _, ok := a.partials[chunk.Index]; ok {
			return
		}
		a.order = append(a.order, chunk.Index)
		a.partials[chunk.Index] = &partialBlock{blockType: chunk.BlockType}
	case ChunkTextDelta, ChunkReasoningDelta:
		kind := BlockText
		if chunk.Type == ChunkReasoningDelta {
			kind = BlockReasoning
		}
		p := a.ensure(chunk.Index, kind)
		if p.block != nil {
			return
		}
		p.text += chunk.Text
	case ChunkToolCallDelta:
		p := a.ensure(chunk.Index, BlockToolCall)
		if p.block != nil {
			return
		}
		if chunk.ID != "" {
			p.toolCallID = chunk.ID
		}
		if chunk.Name != "" {
			p.toolCallName = chunk.Name
		}
		p.toolCallArguments += chunk.ArgumentsDelta
	case ChunkBlockEnd:
		kind := BlockText
		if chunk.Block != nil && chunk.Block.Type != "" {
			kind = chunk.Block.Type
		}
		p := a.ensure(chunk.Index, kind)
		if p.block != nil {
			return
		}
		p.block = chunk.Block
	case ChunkUsage:
		a.usage = chunk.Usage
	case ChunkFinish:
		a.finish = chunk.Reason
	}
}

// ensure 取出或创建该序号的半成品，并在新建时记下顺序。
// index 是 partials 的键。
// blockType 只在该键还没有半成品时写入。
func (a *BlockAssembler) ensure(index int, blockType string) *partialBlock {
	p := a.partials[index]
	if p == nil {
		p = &partialBlock{blockType: blockType}
		a.partials[index] = p
		a.order = append(a.order, index)
	}
	return p
}

// assemble 在已有结束帧时用那一帧。工具调用没有 id 时用 call- 加序号。
// p 已有结束帧时返回那一帧，否则按 blockType 组装文本或工具调用。
// index 在工具调用 id 为空时拼进 call- 前缀。
func (a *BlockAssembler) assemble(p *partialBlock, index int) ContentBlock {
	if p.block != nil {
		return *p.block
	}
	switch p.blockType {
	case BlockText:
		return ContentBlock{Type: BlockText, Text: p.text}
	case BlockReasoning:
		return ContentBlock{Type: BlockReasoning, Text: p.text}
	case BlockToolCall:
		id := p.toolCallID
		if id == "" {
			id = "call-" + strconv.Itoa(index)
		}
		return ContentBlock{Type: BlockToolCall, ID: id, Name: p.toolCallName, Arguments: p.toolCallArguments}
	default:
		return ContentBlock{Type: p.blockType, Text: p.text}
	}
}

// all 按流顺序返回全部块，包括工具调用。
func (a *BlockAssembler) all() []ContentBlock {
	out := make([]ContentBlock, 0, len(a.order))
	for _, idx := range a.order {
		p := a.partials[idx]
		if p == nil {
			continue
		}
		out = append(out, a.assemble(p, idx))
	}
	return out
}

// Blocks 按流顺序返回组装好的块。输出 token 上限截断时丢掉工具调用块，因为那些块不能安全执行。
func (a *BlockAssembler) Blocks() []ContentBlock {
	all := a.all()
	if a.Finish().Kind != FinishMaxTokens {
		return all
	}
	out := make([]ContentBlock, 0, len(all))
	for _, b := range all {
		if b.Type == BlockToolCall {
			continue
		}
		out = append(out, b)
	}
	return out
}

// InterruptedBlocks 是被打断的流可以安全收口的前缀：已关闭和仍开着的、含非空白内容的文本或推理。工具调用被省略，因为打断发生在分发之前。
func (a *BlockAssembler) InterruptedBlocks() []ContentBlock {
	var out []ContentBlock
	for _, idx := range a.order {
		p := a.partials[idx]
		if p == nil {
			continue
		}
		typ := p.blockType
		if p.block != nil {
			typ = p.block.Type
		}
		if typ != BlockText && typ != BlockReasoning {
			continue
		}
		b := a.assemble(p, idx)
		if (b.Type == BlockText || b.Type == BlockReasoning) && strings.TrimSpace(b.Text) != "" {
			out = append(out, b)
		}
	}
	return out
}

// Usage 返回最近一次用量块。还没有则返回 nil。
func (a *BlockAssembler) Usage() *TokenUsage { return a.usage }

// Finish 返回结束原因。还没有则是正常停止。
func (a *BlockAssembler) Finish() FinishReason {
	if a.finish == nil {
		return FinishReason{Kind: FinishStop}
	}
	return *a.finish
}
