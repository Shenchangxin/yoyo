package winsize

const (
	MinW          = 1080
	MinH          = 680
	FallbackW     = 1600
	FallbackH     = 900
	DefaultFrac   = 0.82
	CurrentFitRev = 3
	overflowSlop  = 16
)

// Fit returns a first-launch window size as a fraction of the work area.
func Fit(aw, ah int, frac float64) (w, h int) {
	if frac <= 0 || frac > 1 {
		frac = DefaultFrac
	}
	if aw <= 0 || ah <= 0 {
		return FallbackW, FallbackH
	}
	w = int(float64(aw) * frac)
	h = int(float64(ah) * frac)
	if w < MinW {
		w = MinW
	}
	if h < MinH {
		h = MinH
	}
	if w > aw {
		w = aw
	}
	if h > ah {
		h = ah
	}
	return w, h
}

// Overflows reports that a window is larger than the work area (DIP).
func Overflows(w, h, aw, ah int) bool {
	if aw > 0 && w > aw+overflowSlop {
		return true
	}
	if ah > 0 && h > ah+overflowSlop {
		return true
	}
	return false
}

// ToDIP converts physical pixels to device-independent pixels.
// Negative values are kept (Windows uses ~-32000 when a window is hidden).
func ToDIP(px, dpi int) int {
	if dpi <= 96 {
		return px
	}
	return int(float64(px) * 96 / float64(dpi))
}

const hideSentinel = -10000

// HiddenSentinel is the off-desktop coordinate Windows reports for a hidden HWND.
func HiddenSentinel(x, y int) bool {
	return x <= hideSentinel || y <= hideSentinel
}

// LostOffscreen reports that a saved window rect is not visibly on the virtual
// desktop. vw/vh <= 0 means the virtual screen is unknown: only the hide
// sentinel is treated as lost, so a window on a second monitor is kept.
func LostOffscreen(x, y, w, h, vx, vy, vw, vh int) bool {
	if HiddenSentinel(x, y) {
		return true
	}
	if w <= 0 || h <= 0 {
		return true
	}
	if vw <= 0 || vh <= 0 {
		return false
	}
	ix1 := max(x, vx)
	iy1 := max(y, vy)
	ix2 := min(x+w, vx+vw)
	iy2 := min(y+h, vy+vh)
	return ix2-ix1 < 80 || iy2-iy1 < 40
}

// KeepInside pins a rectangle inside a box. If the box is smaller than the
// rectangle, it pins to the box origin plus pad.
func KeepInside(x, y, w, h, boxX, boxY, boxW, boxH, pad int) (int, int) {
	if boxW <= 0 || boxH <= 0 {
		return x, y
	}
	if pad < 0 {
		pad = 0
	}
	minX := boxX + pad
	minY := boxY + pad
	maxX := boxX + boxW - w - pad
	maxY := boxY + boxH - h - pad
	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}
	if x < minX {
		x = minX
	}
	if y < minY {
		y = minY
	}
	if x > maxX {
		x = maxX
	}
	if y > maxY {
		y = maxY
	}
	return x, y
}

// FullyInside reports that the rectangle sits entirely within the box.
func FullyInside(x, y, w, h, boxX, boxY, boxW, boxH int) bool {
	if boxW <= 0 || boxH <= 0 || w <= 0 || h <= 0 {
		return false
	}
	return x >= boxX && y >= boxY && x+w <= boxX+boxW && y+h <= boxY+boxH
}

// ShouldRefit reports whether a saved size should be replaced by the current
// screen-relative default. Overflow always refits. A stored fit_rev below
// CurrentFitRev migrates once (including shrink). After CurrentFitRev is
// persisted, an on-screen operator resize is kept.
func ShouldRefit(savedW, savedH, fitRev, _, _, aw, ah int) bool {
	if Overflows(savedW, savedH, aw, ah) {
		return true
	}
	if fitRev >= CurrentFitRev {
		return false
	}
	return true
}
