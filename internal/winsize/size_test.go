package winsize

import "testing"

func TestFitFraction(t *testing.T) {
	w, h := Fit(2000, 1200, 0.85)
	if w != 1700 || h != 1020 {
		t.Fatalf("got %dx%d", w, h)
	}
}

func TestFitDefaultFrac(t *testing.T) {
	w, h := Fit(2560, 1440, 0)
	if w != 2099 || h != 1180 {
		t.Fatalf("got %dx%d", w, h)
	}
}

func TestFitFallback(t *testing.T) {
	w, h := Fit(0, 0, DefaultFrac)
	if w != FallbackW || h != FallbackH {
		t.Fatalf("got %dx%d", w, h)
	}
}

func TestFitClampsToWorkArea(t *testing.T) {
	w, h := Fit(1000, 600, DefaultFrac)
	if w != 1000 || h != 600 {
		t.Fatalf("got %dx%d", w, h)
	}
}

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

func TestOverflows(t *testing.T) {
	if !Overflows(3532, 1987, 2560, 1440) {
		t.Fatal("physical-pixel window overflows DIP work area")
	}
	if Overflows(2252, 1267, 2560, 1440) {
		t.Fatal("fitted window must sit inside work area")
	}
}

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
