package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestPromptChangeAppendsWithoutRewritingHead(t *testing.T) {
	ws := t.TempDir()
	st := trace.NewStore(t.TempDir())
	req := RunRequest{
		SessionID: "s",
		User:      "先看项目",
		Workspace: ws,
		Trace:     st,
		Loop:      DefaultLoop(),
	}
	k := newContextKernel(&req)
	first, err := k.seed()
	if err != nil {
		t.Fatal(err)
	}
	head, pins := first[0].Content, ""
	if len(first) > 1 && first[1].Role == RoleDeveloper {
		pins = first[1].Content
	}
	if strings.Contains(head+pins, "blue-door-rule") {
		t.Fatal("rule leaked into the first head")
	}

	if err := os.WriteFile(filepath.Join(ws, "YOYO.md"), []byte("blue-door-rule\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	req.User = "规则变了"
	req.History = MessagesFromEvents(evs)
	k2 := newContextKernel(&req)
	second, err := k2.seed()
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Content != head {
		t.Fatal("system head was rewritten")
	}
	if len(second) > 1 && second[1].Role == RoleDeveloper && second[1].Content != pins {
		t.Fatal("pins head was rewritten")
	}
	if k2.cachePrefix() != head+pins {
		t.Fatalf("cache prefix %q want head", k2.cachePrefix())
	}
	updates := 0
	for _, m := range second {
		if strings.HasPrefix(m.Content, promptUpdateLead) {
			updates++
			if !strings.Contains(m.Content, "blue-door-rule") || !isControlUser(m.Content) {
				t.Fatalf("update %q", m.Content)
			}
		}
	}
	if updates != 1 {
		t.Fatalf("updates %d", updates)
	}

	evs, err = st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	req.User = "再来一次"
	req.History = MessagesFromEvents(evs)
	k3 := newContextKernel(&req)
	third, err := k3.seed()
	if err != nil {
		t.Fatal(err)
	}
	if third[0].Content != head {
		t.Fatal("third turn rewrote the head")
	}
	updates = 0
	for _, m := range third {
		if strings.HasPrefix(m.Content, promptUpdateLead) {
			updates++
		}
	}
	if updates != 1 {
		t.Fatalf("stable prompt must keep one update, got %d", updates)
	}
}
