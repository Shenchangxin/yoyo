// 本文件锁住空闲的 View 不启动 Chrome，以及操作日志会留在 View.Log 里。
// 由 go test 调用。

package browser

import "testing"

// TestViewIdleHasNoChrome 锁住空闲视图的 Live 为假、通道是 isolated、没有截图、配置名是 default，标签数和录制都是零。
func TestViewIdleHasNoChrome(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v := h.View()
	if v.Live {
		t.Fatal("idle view must not spawn Chromium")
	}
	if v.Lane != "isolated" {
		t.Fatalf("lane %q", v.Lane)
	}
	if v.Screenshot != "" {
		t.Fatal("no frame yet")
	}
	if v.ProfileName != "default" {
		t.Fatalf("profileName %q", v.ProfileName)
	}
	if v.TabCount != 0 || v.Recording {
		t.Fatalf("idle view leaked chrome state: %+v", v)
	}
}

// TestViewKeepsJournal 锁住 open 和 click 两条日志会按顺序出现在 View 里。
func TestViewKeepsJournal(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h.record("open", "https://example.com")
	h.record("click", "#go")
	v := h.View()
	if len(v.Log) != 2 {
		t.Fatalf("log %d", len(v.Log))
	}
	if v.Log[0]["op"] != "open" || v.Log[1]["op"] != "click" {
		t.Fatalf("%+v", v.Log)
	}
}
