package app_test

import (
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func leadWait(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestDesktopTakesTheLeadAndTheDaemonFollows 模拟同一个 home 上先开 yoyo daemon、再开桌面：
// 桌面接过后台，daemon 跟着读，把桌面写进主会话的短讯补发给自己的界面；桌面关掉之后 daemon 接回去。
func TestDesktopTakesTheLeadAndTheDaemonFollows(t *testing.T) {
	root := t.TempDir()
	daemon, err := app.OpenManaged(root, evalsDir(t), "daemon")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = daemon.Close() }()
	if got := daemon.LeadState(); got.Role != "lead" || got.Holder != "daemon" {
		t.Fatalf("daemon alone = %+v", got)
	}

	desk, err := app.OpenManaged(root, evalsDir(t), "desktop")
	if err != nil {
		t.Fatal(err)
	}
	deskOpen := true
	defer func() {
		if deskOpen {
			_ = desk.Close()
		}
	}()
	leadWait(t, "the desktop to lead", func() bool {
		return desk.LeadState().Role == "lead" && daemon.LeadState().Role == "follow"
	})
	if h := daemon.LeadState().Holder; h != "desktop" {
		t.Fatalf("holder seen by the daemon = %q", h)
	}

	meta, err := desk.MainSession()
	if err != nil {
		t.Fatal(err)
	}
	ch, unsub := daemon.Hub.Subscribe(meta.ID)
	defer unsub()
	// 两拍以上：跟着的进程第一次看到这个会话时从文件尾开始读。
	time.Sleep(5 * time.Second)
	const text = "「评审」今天到期。"
	if !desk.DeskPush(app.DeskNote{SessionID: meta.ID, Kind: "speak", Slot: "cal:1", Day: "2026-11-28", Text: text}) {
		t.Fatal("the leader could not push")
	}
	deadline := time.After(15 * time.Second)
	for seen := false; !seen; {
		select {
		case ev := <-ch:
			seen = ev.Payload["text"] == text
		case <-deadline:
			t.Fatal("the follower never showed the leader's note")
		}
	}

	_ = desk.Close()
	deskOpen = false
	leadWait(t, "the daemon to take the lead back", func() bool { return daemon.LeadState().Role == "lead" })
}

// TestToolRoleNeverLeads 确认一次性的命令行不抢后台。
func TestToolRoleNeverLeads(t *testing.T) {
	root := t.TempDir()
	cli, err := app.OpenManaged(root, evalsDir(t), "tool")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()
	if got := cli.LeadState(); got.Role != "follow" || got.Holder != "" {
		t.Fatalf("tool = %+v", got)
	}
	daemon, err := app.OpenManaged(root, evalsDir(t), "daemon")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = daemon.Close() }()
	if got := daemon.LeadState(); got.Role != "lead" {
		t.Fatalf("a running command line must not keep the daemon from leading: %+v", got)
	}
}
