// 锁住允许列表，以及非 Windows 在未设置 YOYO_CU_DISPLAY 时 Capture 会拒绝。

package computeruse

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCaptureRefusesOperatorScreen 把 YOYO_CU_DISPLAY 设为空。非 Windows 上 Capture 必须返回含 refusing 的错误。Windows 上没有错误则直接通过。
func TestCaptureRefusesOperatorScreen(t *testing.T) {
	t.Setenv("YOYO_CU_DISPLAY", "")
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = h.Capture(filepath.Join(t.TempDir(), "x.png"))
	if err == nil {
		if runtime.GOOS == "windows" {
			return
		}
		t.Fatal("must refuse primary screen")
	}
	if runtime.GOOS != "windows" && !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("want refuse, got %v", err)
	}
}

// TestAllowlistBlocksOfflist 确认未允许的应用使 Act 失败。允许后事件的 Display 非空，Recording 有一条。
func TestAllowlistBlocksOfflist(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Act("notepad", "click", "ok"); err == nil {
		t.Fatal("offlist")
	}
	h.Allow("notepad")
	ev, err := h.Act("notepad", "click", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Display == "" {
		t.Fatal("display")
	}
	if len(h.Recording()) != 1 {
		t.Fatal("journal")
	}
}
