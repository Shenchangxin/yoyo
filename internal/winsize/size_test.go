// 本文件锁住 Fit 的比例、下限、DIP 换算、离屏判断和 ShouldRefit。由 go test 调用。

package winsize

import "testing"

// TestFitFraction 锁住 2000×1200 乘 0.85 得到 1700×1020。
func TestFitFraction(t *testing.T) {
	w, h := Fit(2000, 1200, 0.85)
	if w != 1700 || h != 1020 {
		t.Fatalf("got %dx%d", w, h)
	}
}

// TestFitDefaultFrac 锁住比例 0 时按 DefaultFrac 计算 2560×1440。
func TestFitDefaultFrac(t *testing.T) {
	w, h := Fit(2560, 1440, 0)
	if w != 2099 || h != 1180 {
		t.Fatalf("got %dx%d", w, h)
	}
}

// TestFitFallback 锁住工作区为 0 时返回 FallbackW 和 FallbackH。
func TestFitFallback(t *testing.T) {
	w, h := Fit(0, 0, DefaultFrac)
	if w != FallbackW || h != FallbackH {
		t.Fatalf("got %dx%d", w, h)
	}
}

// TestFitClampsToWorkArea 锁住 1000×600 的工作区不会被抬到 MinW、MinH 以上。
func TestFitClampsToWorkArea(t *testing.T) {
	w, h := Fit(1000, 600, DefaultFrac)
	if w != 1000 || h != 600 {
		t.Fatalf("got %dx%d", w, h)
	}
}

// TestToDIP 锁住 144 DPI 下 3840 变成 2560，96 DPI 和负数原样保留，0 仍是 0。
func TestToDIP(t *testing.T) {
	if ToDIP(3840, 144) != 2560 {
		t.Fatalf("150%% 3840 -> %d", ToDIP(3840, 144))
	}
	if ToDIP(1920, 96) != 1920 {
		t.Fatal("96 dpi must stay")
	}
	if ToDIP(-32000, 96) != -32000 {
		t.Fatal("hidden sentinel must stay signed")
	}
	if ToDIP(0, 144) != 0 {
		t.Fatal("zero stays zero")
	}
}

// TestLostOffscreen 锁住隐藏坐标和完全离开虚拟桌面为真，屏内窗口、未知虚拟屏幕和第二块显示器为假。
func TestLostOffscreen(t *testing.T) {
	if !LostOffscreen(-32000, -32000, 1600, 900, 0, 0, 2560, 1440) {
		t.Fatal("Windows hide sentinel")
	}
	if !LostOffscreen(8000, 0, 1600, 900, 0, 0, 2560, 1440) {
		t.Fatal("fully off the virtual desktop")
	}
	if LostOffscreen(200, 80, 1600, 900, 0, 0, 2560, 1440) {
		t.Fatal("on-screen window must be kept")
	}
	if LostOffscreen(2560, 100, 1600, 900, 0, 0, 0, 0) {
		t.Fatal("unknown virtual screen must not clobber a second monitor")
	}
	if LostOffscreen(2700, 100, 1600, 900, 0, 0, 5120, 1440) {
		t.Fatal("second monitor in the virtual desktop must be kept")
	}
}

// TestKeepInside 锁住越出左上角和右下角会被夹进内边距，已经在内部的坐标不动。
func TestKeepInside(t *testing.T) {
	x, y := KeepInside(-80, -40, 176, 208, 0, 0, 1920, 1080, 8)
	if x != 8 || y != 8 {
		t.Fatalf("off top-left -> %d,%d", x, y)
	}
	x, y = KeepInside(1900, 1000, 176, 208, 0, 0, 1920, 1080, 8)
	if x != 1920-176-8 || y != 1080-208-8 {
		t.Fatalf("off bottom-right -> %d,%d", x, y)
	}
	x, y = KeepInside(100, 80, 176, 208, 0, 0, 1920, 1080, 8)
	if x != 100 || y != 80 {
		t.Fatalf("on-screen moved: %d,%d", x, y)
	}
}

// TestFullyInside 锁住完全在盒子里为真，右边越出为假。
func TestFullyInside(t *testing.T) {
	if !FullyInside(100, 80, 176, 208, 0, 0, 1920, 1080) {
		t.Fatal("on-screen must be fully inside")
	}
	if FullyInside(1800, 80, 176, 208, 0, 0, 1920, 1080) {
		t.Fatal("hanging off the right edge")
	}
}

// TestOverflows 锁住明显大于 DIP 工作区的窗口为真，Fit 出来的窗口为假。
func TestOverflows(t *testing.T) {
	if !Overflows(3532, 1987, 2560, 1440) {
		t.Fatal("physical-pixel window overflows DIP work area")
	}
	if Overflows(2252, 1267, 2560, 1440) {
		t.Fatal("fitted window must sit inside work area")
	}
}

// TestShouldRefitMigratesOldDefault 锁住 fitRev 0 和 2 会重适配，当前 fitRev 下的默认尺寸和操作者改过的尺寸保留，溢出即使 fitRev 已是当前值也重适配。
func TestShouldRefitMigratesOldDefault(t *testing.T) {
	fitW, fitH := Fit(2560, 1440, DefaultFrac)
	if !ShouldRefit(1440, 900, 0, fitW, fitH, 2560, 1440) {
		t.Fatal("old 1440x900 should refit")
	}
	if !ShouldRefit(2252, 1267, 2, fitW, fitH, 2560, 1440) {
		t.Fatal("fit_rev 2 88% default should shrink to current fit")
	}
	if ShouldRefit(fitW, fitH, CurrentFitRev, fitW, fitH, 2560, 1440) {
		t.Fatal("current fit must stick")
	}
	if ShouldRefit(1200, 800, CurrentFitRev, fitW, fitH, 2560, 1440) {
		t.Fatal("operator resize after migration must stick")
	}
	if !ShouldRefit(3532, 1987, CurrentFitRev, fitW, fitH, 2560, 1440) {
		t.Fatal("overflow must refit even after migration")
	}
}
