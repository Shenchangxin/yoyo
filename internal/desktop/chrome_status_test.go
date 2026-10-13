package desktop

import "testing"

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

func TestCompanionContextMenuID(t *testing.T) {
	if companionContextMenuID != "yoyo-companion" {
		t.Fatalf("css --custom-contextmenu must match %q", companionContextMenuID)
	}
}

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

func TestResetCompanionWithoutWindow(t *testing.T) {
	s := &Service{}
	s.ResetCompanion()
}

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

func TestOpenCompanionChatWithoutApp(t *testing.T) {
	s := &Service{}
	s.OpenCompanionChat()
}
