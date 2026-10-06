package desktop

import "github.com/Shenchangxin/yoyo/internal/winsize"

const (
	companionW         = 232
	companionH         = 226
	companionPad       = 8
	companionPet       = 148
	companionPetTop    = 24
	companionBubbleGap = 6
	companionBubbleW   = 168
	companionBubbleH   = 40
	// Hit disk is slightly inside the 148px box so empty corners pass through,
	// but larger than the viewBox inset so bounce/lean still grab.
	companionHitFill = 0.88
	// Occupancy follows paint: the HWND ellipse hugs the blob chin (~4px)
	// and keeps bounce headroom above the crown, not a stretched oval below.
	companionShapeFillX = 1.08
	companionBlobInset  = 0.06
	companionBounceHead = 28
	companionChinPad    = 4
)

type hitBox struct {
	x, y, w, h int
}

// CompanionHit is true when a window-local point sits on the blob disk
// or, when caption is on, on the caption chip under it. Units only need
// to match the window size (DIP or physical).
func CompanionHit(lx, ly, winW, winH float64, caption bool) bool {
	if winW < 8 || winH < 8 {
		return false
	}
	sx := winW / float64(companionW)
	sy := winH / float64(companionH)
	cx := winW * 0.5
	cy := (float64(companionPetTop) + float64(companionPet)*0.5) * sy
	r := float64(companionPet) * 0.5 * companionHitFill * min(sx, sy)
	dx := lx - cx
	dy := ly - cy
	if dx*dx+dy*dy <= r*r {
		return true
	}
	if !caption {
		return false
	}
	bw := float64(companionBubbleW) * sx
	bh := float64(companionBubbleH) * sy
	bx := (winW - bw) * 0.5
	by := (float64(companionPetTop+companionPet) + float64(companionBubbleGap)) * sy
	return lx >= bx && lx <= bx+bw && ly >= by && ly <= by+bh
}

func companionScale(winW, winH float64) (sx, sy, s float64) {
	sx = winW / float64(companionW)
	sy = winH / float64(companionH)
	return sx, sy, min(sx, sy)
}

func companionBlobEllipse(winW, winH float64) (cx, cy, rx, ry float64) {
	_, sy, s := companionScale(winW, winH)
	cx = winW * 0.5
	boxTop := float64(companionPetTop) * sy
	boxH := float64(companionPet) * sy
	inset := float64(companionPet) * companionBlobInset * sy
	paintTop := boxTop + inset
	paintBot := boxTop + boxH - inset
	top := paintTop - float64(companionBounceHead)*sy
	if top < 0 {
		top = 0
	}
	bot := paintBot + float64(companionChinPad)*sy
	cy = (top + bot) * 0.5
	ry = (bot - top) * 0.5
	rx = float64(companionPet) * 0.5 * companionShapeFillX * s
	return cx, cy, rx, ry
}

func companionCaptionRect(winW, winH float64) (x, y, w, h float64) {
	sx, sy, _ := companionScale(winW, winH)
	w = float64(companionBubbleW) * sx
	h = float64(companionBubbleH) * sy
	x = (winW - w) * 0.5
	y = (float64(companionPetTop+companionPet) + float64(companionBubbleGap)) * sy
	return x, y, w, h
}

func inEllipse(px, py, cx, cy, rx, ry float64) bool {
	if rx < 1 || ry < 1 {
		return false
	}
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	return dx*dx+dy*dy <= 1
}

func dist2ToBox(px, py int, b hitBox) int {
	x, y := px, py
	if x < b.x {
		x = b.x
	} else if right := b.x + b.w; x > right {
		x = right
	}
	if y < b.y {
		y = b.y
	} else if bottom := b.y + b.h; y > bottom {
		y = bottom
	}
	dx := px - x
	dy := py - y
	return dx*dx + dy*dy
}

func nearestBox(px, py int, boxes []hitBox) (hitBox, bool) {
	var best hitBox
	found := false
	bestD := 0
	for _, b := range boxes {
		if b.w <= 0 || b.h <= 0 {
			continue
		}
		d := dist2ToBox(px, py, b)
		if !found || d < bestD {
			best = b
			bestD = d
			found = true
		}
	}
	return best, found
}

func clampCompanionTo(x, y, w, h, pad int, boxes []hitBox, vx, vy, vw, vh int) (int, int) {
	if home, ok := nearestBox(x+w/2, y+h/2, boxes); ok {
		x, y = winsize.KeepInside(x, y, w, h, home.x, home.y, home.w, home.h, pad)
	}
	return winsize.KeepInside(x, y, w, h, vx, vy, vw, vh, 4)
}
