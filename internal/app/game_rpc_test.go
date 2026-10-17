package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameCreateBindAndCompile(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Game == nil {
		t.Fatal("game engine missing")
	}
	raw, err := a.VideoCall("game.create", map[string]any{
		"title": "Relic run", "intent": "Collect three relics and reach the shrine.", "session_id": "sess-game",
	})
	if err != nil {
		t.Fatal(err)
	}
	row, _ := raw.(map[string]any)
	id := strings.TrimSpace(fmt.Sprint(row["id"]))
	if id == "" || id == "<nil>" {
		t.Fatalf("create id: %#v", raw)
	}
	if fmt.Sprint(row["stage"]) != "compiled" && fmt.Sprint(row["playable"]) == "" {
		t.Fatalf("expected compiled playable: %#v", row)
	}
	got, err := a.VideoCall("video.session.project", map[string]any{"session_id": "sess-game"})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := got.(map[string]any)
	if strings.TrimSpace(fmt.Sprint(st["game_id"])) != id && strings.TrimSpace(fmt.Sprint(st["id"])) != id {
		t.Fatalf("bind not visible: %#v", got)
	}
	hist, err := a.VideoCall("video.history", nil)
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := hist.([]map[string]any)
	found := false
	for _, r := range rows {
		if fmt.Sprint(r["kind"]) == "game" && fmt.Sprint(r["id"]) == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("history missing game: %#v", hist)
	}
}
