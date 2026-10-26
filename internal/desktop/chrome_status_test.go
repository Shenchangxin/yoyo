package desktop

import "testing"

// TestChromeStatusPriority 锁住暂停压过工作和审批，审批压过工作，有回合时是工作，否则空闲。
func TestChromeStatusPriority(t *testing.T) {
	if chromeStatusOf(true, true, 2) != chromePaused {
		t.Fatal("pause outranks work and approval")
	}
	if chromeStatusOf(false, true, 1) != chromeApproval {
		t.Fatal("approval outranks work")
	}
	if chromeStatusOf(false, true, 0) != chromeWorking {
		t.Fatal("running turn is working")
	}
	if chromeStatusOf(false, false, 0) != chromeIdle {
		t.Fatal("idle")
	}
}

// TestChromeLineAndTooltip 锁住英文空闲行、中文处理中行，以及 Windows 上暂停提示带 Ctrl+Shift+Y。wake 为假时不加第二行。
func TestChromeLineAndTooltip(t *testing.T) {
	en := nativeCopy("en")
	if en.chromeLine(chromeIdle) != "Yoyo · Idle" {
		t.Fatalf("idle line %q", en.chromeLine(chromeIdle))
	}
	zh := nativeCopy("zh-CN")
	if zh.chromeLine(chromeWorking) != "Yoyo · 处理中" {
		t.Fatalf("working line %q", zh.chromeLine(chromeWorking))
	}
	tip := en.trayTooltip(chromePaused, true)
	if tip != "Yoyo · Paused\nCtrl+Shift+Y to show" {
		t.Fatalf("wake tooltip %q", tip)
	}
	if en.trayTooltip(chromeIdle, false) != "Yoyo · Idle" {
		t.Fatal("no wake hint off Windows")
	}
	if n := len([]rune(en.trayTooltip(chromeApproval, true))); n > 127 {
		t.Fatalf("windows tooltip %d runes", n)
	}
}

// TestCompanionMenuCopy 锁住中英文宠物菜单文案都非空，设置项以省略号结尾。
func TestCompanionMenuCopy(t *testing.T) {
	for _, loc := range []string{"en", "zh"} {
		n := nativeCopy(loc)
		for _, s := range []string{
			n.Companion, n.FocusMode, n.ResetPosition, n.OpenChat, n.VoiceCall, n.HangUp,
			n.ReviewApproval, n.StartAtLogin, n.StopTask, n.SettingsEllipsis,
			n.StatusIdle, n.StatusPaused, n.StatusApproval, n.HideCompanion,
		} {
			if s == "" {
				t.Fatalf("empty native string for %s", loc)
			}
		}
		if n.SettingsEllipsis[len(n.SettingsEllipsis)-len("…"):] != "…" {
			t.Fatalf("settings must use an ellipsis, got %q", n.SettingsEllipsis)
		}
	}
}

// TestSliceLen 锁住 nil 的长度是 0，两元素切片是 2，标量是 0。
func TestSliceLen(t *testing.T) {
	if sliceLen(nil) != 0 {
		t.Fatal("nil")
	}
	if sliceLen([]string{"a", "b"}) != 2 {
		t.Fatal("strings")
	}
	if sliceLen(3) != 0 {
		t.Fatal("scalar")
	}
}

// TestCompanionContextMenuID 锁住右键菜单 id 是 yoyo-companion，和前端的 custom-contextmenu 一致。
func TestCompanionContextMenuID(t *testing.T) {
	if companionContextMenuID != "yoyo-companion" {
		t.Fatalf("css --custom-contextmenu must match %q", companionContextMenuID)
	}
}

// TestCompanionIgnoresHits 锁住默认接收点击，专注模式穿透，语音通话时即使专注也不穿透。
func TestCompanionIgnoresHits(t *testing.T) {
	s := &Service{}
	if s.companionIgnoresHits() {
		t.Fatal("default must receive hits")
	}
	s.companionFocus.Store(true)
	if !s.companionIgnoresHits() {
		t.Fatal("focus mode must pass clicks through")
	}
	s.companionCall.Store(true)
	if s.companionIgnoresHits() {
		t.Fatal("a voice call needs the pet interactive")
	}
}

// TestSetCompanionCallNoop 锁住空闲时挂断仍空闲，打开通话后保持交互，重复打开不拆掉状态，挂断后清掉通话标志。
func TestSetCompanionCallNoop(t *testing.T) {
	s := &Service{}
	s.SetCompanionCall(false)
	if s.companionCall.Load() || companionInteractive.Load() {
		t.Fatal("idle hangup must stay idle")
	}
	s.SetCompanionCall(true)
	if !s.companionCall.Load() || !companionInteractive.Load() {
		t.Fatal("voice on")
	}
	s.SetCompanionCall(true)
	if !s.companionCall.Load() || !companionInteractive.Load() {
		t.Fatal("same-state call must keep chrome")
	}
	s.SetCompanionCall(false)
	if s.companionCall.Load() || companionInteractive.Load() {
		t.Fatal("hangup must clear call chrome")
	}
}

// TestHideCompanionEndsCall 锁住隐藏宠物会标成已收起，并结束通话。
func TestHideCompanionEndsCall(t *testing.T) {
	s := &Service{}
	s.companionCall.Store(true)
	companionInteractive.Store(true)
	s.HideCompanion()
	if !s.companionDismissed.Load() {
		t.Fatal("hide must dismiss the pet")
	}
	if s.companionCall.Load() || companionInteractive.Load() {
		t.Fatal("hide pet must hang up")
	}
}

// TestResetCompanionWithoutWindow 锁住没有宠物窗口时 ResetCompanion 直接返回。
func TestResetCompanionWithoutWindow(t *testing.T) {
	s := &Service{}
	s.ResetCompanion()
}

// TestAgentWorkingWithoutApp 锁住空 Service 不在工作，agentBusy 为真时算正在工作。
func TestAgentWorkingWithoutApp(t *testing.T) {
	s := &Service{}
	if s.agentWorking() {
		t.Fatal("empty service is idle")
	}
	s.agentBusy.Store(true)
	if !s.agentWorking() {
		t.Fatal("pulse busy")
	}
}

// TestOpenCompanionChatWithoutApp 锁住没有 App、RPC 和窗口时 OpenCompanionChat 也能返回。
func TestOpenCompanionChatWithoutApp(t *testing.T) {
	s := &Service{}
	s.OpenCompanionChat()
}
