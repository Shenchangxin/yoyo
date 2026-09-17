package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestMeasureContextCountsPrefixAndHistory(t *testing.T) {
	loop := DefaultLoop()
	idle := MeasureContext(MeasureOpts{Loop: loop, ModelWindow: 128_000, Tools: &WorkspaceTools{Depth: 1}})
	if idle.Window != 128_000 {
		t.Fatalf("window %d", idle.Window)
	}
	if idle.PrefixTokens <= 0 || idle.SchemaTokens <= 0 || idle.Tokens <= 0 {
		t.Fatalf("idle ledger %+v", idle)
	}
	if idle.Tokens < idle.PrefixTokens+idle.SchemaTokens {
		t.Fatalf("tokens %d < prefix+schema %d+%d", idle.Tokens, idle.PrefixTokens, idle.SchemaTokens)
	}
	withChat := MeasureContext(MeasureOpts{
		Loop:        loop,
		ModelWindow: 128_000,
		Tools:       &WorkspaceTools{Depth: 1},
		History: []Message{
			{Role: RoleUser, Content: "hello from the user, please look at this workspace"},
			{Role: RoleAssistant, Content: "I will inspect the files next."},
		},
	})
	if withChat.Tokens <= idle.Tokens {
		t.Fatalf("history should add tokens: idle=%d chat=%d", idle.Tokens, withChat.Tokens)
	}
	chat := withChat.Tokens - withChat.PrefixTokens - withChat.SchemaTokens - withChat.DynamicTokens
	if chat <= 0 {
		t.Fatalf("expected chat slice, %+v", withChat)
	}
}

func TestShapeFromEventsReadsTurnEndLedger(t *testing.T) {
	evs := []trace.Event{{
		Type: trace.TypeTurnEnd,
		Payload: map[string]any{
			"ok": true, "tokens": 4200, "window": 128000, "prefix_tokens": 800,
			"schema_tokens": 1200, "dynamic_tokens": 40, "budget": 99000,
			"provider_prompt": 4100, "layers": []any{"budget"},
		},
	}}
	rep := ShapeFromEvents(evs, 0)
	if rep.Tokens != 4200 || rep.PrefixTokens != 800 || rep.ProviderPrompt != 4100 || rep.Window != 128000 {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Layers) != 1 || rep.Layers[0] != "budget" {
		t.Fatalf("layers %+v", rep.Layers)
	}
}
