package runtime

import (
	"strings"
	"testing"
)

func TestReadThreadListsAndSends(t *testing.T) {
	sent := ""
	tools := &WorkspaceTools{
		SessionID: "here",
		ListThreads: func() []ThreadRef {
			return []ThreadRef{{ID: "abc", Title: "Weekly"}, {ID: "here", Title: "self"}, {ID: "abc/tasks/x", Title: "child"}}
		},
		SendThread: func(id, text string) error {
			sent = id + ":" + text
			return nil
		},
		Sessions: func(id string) []Message {
			return []Message{{Role: RoleUser, Content: "hello from " + id}}
		},
	}
	list := tools.readThread("", "", "")
	if list.Err != nil || !strings.Contains(list.Content, "`abc`") || strings.Contains(list.Content, "tasks") {
		t.Fatalf("%+v", list)
	}
	got := tools.readThread("abc", "hello", "")
	if got.Err != nil || !strings.Contains(got.Content, "hello from abc") {
		t.Fatalf("%+v", got)
	}
	send := tools.readThread("abc", "", "steer please")
	if send.Err != nil || sent != "abc:steer please" {
		t.Fatalf("%+v sent=%s", send, sent)
	}
}
