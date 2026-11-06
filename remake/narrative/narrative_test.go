// Copyright 2026 Hypit.AI
// Translated from @hypit/narrative
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package narrative

import (
	"strings"
	"testing"
)

func TestOneSegmentPartitionsItsToken(t *testing.T) {
	value := sample()
	AssertIdentity(value)
	id := value.ID
	span := SelectionTokenRange(value, Selection{ID: "sel", StartAnchorID: "t0", EndAnchorID: "t1"}, &id)
	if span.TokenStart != 0 || span.TokenEndExclusive != 1 {
		t.Fatalf("%+v", span)
	}
	tokens := TokensForSelection(value, Selection{ID: "sel", StartAnchorID: "t0", EndAnchorID: "t1"}, nil)
	if len(tokens) != 1 || tokens[0].Text != "Hi" {
		t.Fatalf("%+v", tokens)
	}
	other := "elsewhere"
	message := recoverMessage(func() {
		SelectionTokenRange(value, Selection{ID: "sel", StartAnchorID: "t0", EndAnchorID: "t1"}, &other)
	})
	if !strings.Contains(message, "belongs to another Narrative") {
		t.Fatal(message)
	}
}

func TestEmptyNarrativeIsRejected(t *testing.T) {
	message := recoverMessage(func() { AssertIdentity(Narrative{ID: "script"}) })
	if message != "Narrative must contain at least one Segment." {
		t.Fatal(message)
	}
	AssertSegmentRefIdentity(SegmentRef{Kind: "segment", NarrativeID: "script", ID: "s", TokenStart: 0, TokenEndExclusive: 1})
	if Schema.Kind != "object" || SegmentRefSchema.Fields["kind"].Schema.Value != "segment" {
		t.Fatal("schema")
	}
	if Package.Format != "hypit.package@1" || len(Component.Validators) != 4 {
		t.Fatal("activation")
	}
}

func sample() Narrative {
	return Narrative{
		ID: "script",
		Segments: []Segment{{
			ID: "s", StartAnchorID: "s0", EndAnchorID: "s1", TokenStart: 0, TokenEndExclusive: 1,
		}},
		Tokens: []Token{{
			ID: "t", SegmentID: "s", StartAnchorID: "t0", EndAnchorID: "t1", Text: "Hi", Normalized: "hi",
		}},
		Anchors: []Anchor{
			{ID: "s0", Kind: "segment-start", SegmentID: "s"},
			{ID: "t0", Kind: "token-start", SegmentID: "s", TokenID: "t"},
			{ID: "t1", Kind: "token-end", SegmentID: "s", TokenID: "t"},
			{ID: "s1", Kind: "segment-end", SegmentID: "s"},
		},
	}
}

func recoverMessage(fn func()) (message string) {
	defer func() {
		rec := recover()
		err, ok := rec.(error)
		if !ok {
			panic(rec)
		}
		message = err.Error()
	}()
	fn()
	return ""
}
