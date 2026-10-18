package runtime

import "testing"

func TestBlockAssemblerDeltaOnly(t *testing.T) {
	a := NewBlockAssembler()
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "he"})
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "llo"})
	blocks := a.Blocks()
	if len(blocks) != 1 || blocks[0].Type != BlockText || blocks[0].Text != "hello" {
		t.Fatalf("%+v", blocks)
	}
}

func TestBlockAssemblerIgnoresStragglersAfterEnd(t *testing.T) {
	a := NewBlockAssembler()
	a.Push(StreamChunk{Type: ChunkBlockStart, Index: 0, BlockType: BlockText})
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "hi"})
	a.Push(StreamChunk{Type: ChunkBlockEnd, Index: 0, Block: &ContentBlock{Type: BlockText, Text: "hi"}})
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: " extra"})
	blocks := a.Blocks()
	if len(blocks) != 1 || blocks[0].Text != "hi" {
		t.Fatalf("%+v", blocks)
	}
}

func TestBlockAssemblerToolCallAndInterrupted(t *testing.T) {
	a := NewBlockAssembler()
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "look"})
	a.Push(StreamChunk{Type: ChunkToolCallDelta, Index: 1, ID: "c1", Name: "read_file", ArgumentsDelta: `{"path":"a"}`})
	got := a.InterruptedBlocks()
	if len(got) != 1 || got[0].Text != "look" {
		t.Fatalf("interrupted %+v", got)
	}
	blocks := a.Blocks()
	if len(blocks) != 2 || blocks[1].Type != BlockToolCall || blocks[1].ID != "c1" {
		t.Fatalf("blocks %+v", blocks)
	}
}

func TestBlockAssemblerMaxTokensDropsTools(t *testing.T) {
	a := NewBlockAssembler()
	a.Push(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "cut"})
	a.Push(StreamChunk{Type: ChunkToolCallDelta, Index: 1, ID: "c1", Name: "write_file", ArgumentsDelta: `{`})
	a.Push(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: FinishMaxTokens}})
	blocks := a.Blocks()
	if len(blocks) != 1 || blocks[0].Text != "cut" {
		t.Fatalf("%+v", blocks)
	}
}

func TestCompactStreamRoundTrip(t *testing.T) {
	acc := NewStreamAccumulator()
	acc.Push(1, StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "he"})
	acc.Push(2, StreamChunk{Type: ChunkTextDelta, Index: 0, Text: "llo"})
	acc.Push(5, StreamChunk{Type: ChunkBlockEnd, Index: 0, Block: &ContentBlock{Type: BlockText, Text: "hello"}})
	recs := acc.Snapshot()
	if len(recs) != 2 || recs[0].Type != "text-chunks" || len(recs[0].Texts) != 2 {
		t.Fatalf("%+v", recs)
	}
	exp := ExpandAssistantStream(recs)
	if len(exp) != 3 || exp[0].Chunk.Text != "he" || exp[1].Chunk.Text != "llo" {
		t.Fatalf("%+v", exp)
	}
	if exp[1].Time != 2 {
		t.Fatalf("time %d", exp[1].Time)
	}
}
