// 锁住 read_thread 列出其他会话时跳过自己和子会话，能按查询取片段，也能把消息送进另一场会话。@session 注入带标题，当前会话不倒出正文。

package runtime

import (
	"strings"
	"testing"
)

// TestReadThreadListsAndSends 锁住列表含 abc 且不含 tasks，读取能命中 hello，发送记成 abc:steer please。FormatSessionMentions 对其他会话带标题和正文，对当前会话不附消息。
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
	overlay := FormatSessionMentions([]Mention{{Kind: "session", Ref: "abc"}}, "here", tools, 1600)
	if !strings.Contains(overlay, "Weekly") || !strings.Contains(overlay, "hello from abc") {
		t.Fatalf("overlay=%s", overlay)
	}
}
