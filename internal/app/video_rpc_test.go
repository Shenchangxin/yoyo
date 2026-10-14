package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRPCStringTreatsMissingAsEmpty(t *testing.T) {
	if got := rpcString(map[string]any{"project_id": "p1"}, "id"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := rpcString(map[string]any{"id": nil}, "id"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := rpcString(nil, "id"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := rpcString(map[string]any{"id": " p1 "}, "id"); got != "p1" {
		t.Fatalf("got %q", got)
	}
}

func TestStudioBindUsesProjectIDWhenIDMissing(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.VideoCall("video.studio.create", map[string]any{"title": "通勤穿搭"})
	if err != nil {
		t.Fatal(err)
	}
	row, _ := raw.(map[string]any)
	id := strings.TrimSpace(fmt.Sprint(row["id"]))
	if id == "" || id == "<nil>" {
		t.Fatalf("create id: %#v", raw)
	}
	if _, err := a.VideoCall("video.studio.bind", map[string]any{
		"session_id": "sess-bind",
		"project_id": id,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := a.VideoCall("video.session.project", map[string]any{"session_id": "sess-bind"})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := got.(map[string]any)
	if strings.TrimSpace(fmt.Sprint(st["studio_id"])) != id && strings.TrimSpace(fmt.Sprint(st["id"])) != id {
		t.Fatalf("bind not visible: %#v", got)
	}
}
