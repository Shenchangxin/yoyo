package runtime

import "testing"

func TestAllToolJSONDropsDuplicateNames(t *testing.T) {
	tools := &WorkspaceTools{
		ChatOverlay:  true,
		Advertised:   append([]string(nil), ChatCoreTools...),
		ExtraEnabled: []string{"ask_user", "cut_get"},
		Extra: map[string]ExtraTool{
			"ask_user": {JSON: fn("ask_user", "canvas duplicate", map[string]any{"type": "object"})},
			"cut_get":  {JSON: fn("cut_get", "read the clip", map[string]any{"type": "object"})},
		},
	}
	seen := map[string]int{}
	for _, j := range AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		seen[name]++
		if seen[name] > 1 {
			t.Fatalf("duplicate tool %s", name)
		}
	}
	if seen["ask_user"] != 1 || seen["cut_get"] != 1 {
		t.Fatalf("missing tools: %#v", seen)
	}
}
