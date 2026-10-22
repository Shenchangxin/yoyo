package desktop

import (
	"sync/atomic"

	"github.com/Shenchangxin/yoyo/internal/winsize"
)

var companionInteractive atomic.Bool
var companionClickThrough atomic.Bool
var companionCaptionOn atomic.Bool

const (
	companionPet             = 168
	companionPetTop          = 40
	companionBubbleGap       = 8
	companionBubbleW         = 228
	companionBubbleH         = 40
	companionBubbleMaxH      = 148
	companionPadBottom       = 24
	companionW               = 268
	companionIdleH           = companionPetTop + companionPet + companionPadBottom
	companionH               = companionIdleH + companionBubbleGap + companionBubbleMaxH
	companionCallW           = 380
	companionCallH           = 640
	companionPad             = 8
	companionBottomClearance = 24
	companionCaptionMinW     = 64
	companionCaptionMinH     = 28
	// Hit disk is slightly inside the pet box so empty corners pass through,
	// but larger than the viewBox inset so bounce/lean still grab.
	companionHitFill = 0.88
	// Occupancy follows paint: a rounded rect around the 168px pet box,
	// not a circle, so bounce/lean/ribbons/hips can overshoot the body.
	companionShapeFillX    = 1.14
	companionBlobInset     = 0.06
	companionBounceHead    = 36
	companionChinPad       = 10
	companionRegionCorner  = 28
	companionCaptionPadX   = 8
	companionCaptionPadTop = 4
	companionCaptionPadBot = 12
	companionCaptionCorner = 16
)

var companionCaptionW atomic.Int32
var companionCaptionH atomic.Int32

func setCompanionCaptionBox(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	if w > companionBubbleW {
		w = companionBubbleW
	}
	if h > companionBubbleMaxH {
		h = companionBubbleMaxH
	}
	companionCaptionW.Store(int32(w))
	companionCaptionH.Store(int32(h))
}

func captionBoxW() float64 {
	w := int(companionCaptionW.Load())
	if w < companionCaptionMinW {
		return float64(companionBubbleW)
	}
	if w > companionBubbleW {
		return float64(companionBubbleW)
	}
	return float64(w)
}

func captionBoxH() float64 {
	h := int(companionCaptionH.Load())
	if h < companionCaptionMinH {
		return float64(companionBubbleH)
	}
	if h > companionBubbleMaxH {
		return float64(companionBubbleMaxH)
	}
	return float64(h)
}

func companionFrameH(caption bool) int {
	if !caption {
		return companionIdleH
	}
	return companionPetTop + companionPet + companionBubbleGap + int(captionBoxH()+0.5) + companionPadBottom
}

func companionDesignH(caption bool) float64 {
	if companionInteractive.Load() {
		return float64(companionCallH)
	}
	return float64(companionFrameH(caption))
}

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
	sy := winH / companionDesignH(caption)
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
	bw := captionBoxW() * sx
	bh := captionBoxH() * sy
	bx := (winW - bw) * 0.5
	by := (float64(companionPetTop+companionPet) + float64(companionBubbleGap)) * sy
	return lx >= bx && lx <= bx+bw && ly >= by && ly <= by+bh
}

func companionScale(winW, winH float64, caption bool) (sx, sy, s float64) {
	sx = winW / float64(companionW)
	sy = winH / companionDesignH(caption)
	return sx, sy, min(sx, sy)
}

func companionBlobEllipse(winW, winH float64, caption bool) (cx, cy, rx, ry float64) {
	_, sy, s := companionScale(winW, winH, caption)
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
	if bot > winH {
		bot = winH
	}
	cy = (top + bot) * 0.5
	ry = (bot - top) * 0.5
	rx = float64(companionPet) * 0.5 * companionShapeFillX * s
	return cx, cy, rx, ry
}

// companionPetRegionRect is the HWND occupancy for the blob: the 168px pet
// box plus bounce/chin/lean. A stadium (round rect) of this size is not a
// circle clip, so the cream body, bounce and ribbons stay visible.
func companionPetRegionRect(winW, winH float64, caption bool) (x, y, w, h float64) {
	_, sy, s := companionScale(winW, winH, caption)
	pet := float64(companionPet) * s
	cx := winW * 0.5
	boxTop := float64(companionPetTop) * sy
	boxH := float64(companionPet) * sy
	padX := pet * (companionShapeFillX - 1) * 0.5
	// Extra room so a 1px HWND/client mismatch or the breathe scale
	// does not shave the left of the circle.
	if padX < 18*s {
		padX = 18 * s
	}
	padTop := float64(companionBounceHead) * sy
	padBot := float64(companionChinPad) * sy
	w = pet + padX*2
	h = boxH + padTop + padBot
	x = cx - w*0.5
	y = boxTop - padTop
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > winW {
		w = winW - x
	}
	if y+h > winH {
		h = winH - y
	}
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
	}
	return x, y, w, h
}

// companionRegionCornerRadius is the visual corner radius of the HWND
// occupancy. Keep it well below half the short side so GDI never draws a
// stadium/circle that clips the blob hips.
func companionRegionCornerRadius(rw, rh, s float64) float64 {
	rad := float64(companionRegionCorner) * s
	if rad < 12*s {
		rad = 12 * s
	}
	if rad < 12 {
		rad = 12
	}
	short := rw
	if rh < short {
		short = rh
	}
	cap := short * 0.32
	if rad > cap {
		rad = cap
	}
	return rad
}

func roundRectContains(px, py, x, y, w, h, rad float64) bool {
	if px < x || py < y || px > x+w || py > y+h {
		return false
	}
	if rad < 1 {
		return true
	}
	if rad*2 > w {
		rad = w / 2
	}
	if rad*2 > h {
		rad = h / 2
	}
	cx := px
	cy := py
	if px < x+rad {
		cx = x + rad
	} else if px > x+w-rad {
		cx = x + w - rad
	} else {
		return true
	}
	if py < y+rad {
		cy = y + rad
	} else if py > y+h-rad {
		cy = y + h - rad
	} else {
		return true
	}
	dx := px - cx
	dy := py - cy
	return dx*dx+dy*dy <= rad*rad
}

func companionCaptionRect(winW, winH float64, caption bool) (x, y, w, h float64) {
	sx, sy, _ := companionScale(winW, winH, caption)
	w = captionBoxW() * sx
	h = captionBoxH() * sy
	x = (winW-w)*0.5 - float64(companionCaptionPadX)*sx
	y = (float64(companionPetTop+companionPet)+float64(companionBubbleGap))*sy - float64(companionCaptionPadTop)*sy
	w += float64(companionCaptionPadX) * 2 * sx
	h += float64(companionCaptionPadTop+companionCaptionPadBot) * sy
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > winW {
		w = winW - x
	}
	if y+h > winH {
		h = winH - y
	}
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
