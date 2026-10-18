package runtime

import (
	"strconv"
	"strings"
)

type partialBlock struct {
	blockType         string
	text              string
	toolCallID        string
	toolCallName      string
	toolCallArguments string
	block             *ContentBlock
}

// BlockAssembler is the single canonical assembly algorithm from StreamChunk
// to ContentBlock. Tolerant of delta-only protocols (no block-start/end);
// deltas arriving for an index already closed by block-end are ignored.
type BlockAssembler struct {
	partials map[int]*partialBlock
	order    []int
	usage    *TokenUsage
	finish   *FinishReason
}

func NewBlockAssembler() *BlockAssembler {
	return &BlockAssembler{partials: map[int]*partialBlock{}}
}

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

func (a *BlockAssembler) ensure(index int, blockType string) *partialBlock {
	p := a.partials[index]
	if p == nil {
		p = &partialBlock{blockType: blockType}
		a.partials[index] = p
		a.order = append(a.order, index)
	}
	return p
}

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

// Blocks returns assembled blocks in stream order. Max-token truncation
// drops tool-call blocks that cannot be executed safely.
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

// InterruptedBlocks is the prefix an interrupted stream can safely finalize:
// closed and open text/reasoning with non-whitespace content. Tool calls are
// omitted because interruption precedes dispatch.
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

func (a *BlockAssembler) Usage() *TokenUsage { return a.usage }

func (a *BlockAssembler) Finish() FinishReason {
	if a.finish == nil {
		return FinishReason{Kind: FinishStop}
	}
	return *a.finish
}

