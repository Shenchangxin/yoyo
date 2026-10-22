package desktop

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/winsize"
)

func TestCompanionCallWindowIsDraggable(t *testing.T) {
	companionInteractive.Store(true)
	t.Cleanup(func() { companionInteractive.Store(false) })
	w, h := float64(companionCallW), float64(companionCallH)
	if !CompanionHit(w/2, h*0.55, w, h, false) {
		t.Fatal("call figure must be a drag target")
	}
	if !CompanionHit(24, h-40, w, h, false) {
		t.Fatal("call card must be draggable away from the idle blob")
	}
	if CompanionHit(-4, h/2, w, h, false) || CompanionHit(w+4, 10, w, h, false) {
		t.Fatal("outside the call card must miss")
	}
}

func TestCompanionChildRegionDuringCall(t *testing.T) {
	parentW, parentH := companionCallW, companionCallH
	if companionChildRegion(parentW, parentH, parentW, parentH, true) != companionChildShape {
		t.Fatal("a webview that matches the call window should take the call region")
	}
	if companionChildRegion(companionW, companionIdleH, parentW, parentH, true) != companionChildClear {
		t.Fatal("a webview still at idle size must drop the pet clip during a call")
	}
	if companionChildRegion(40, 40, parentW, parentH, true) != companionChildClear {
		t.Fatal("a tiny child must not keep a pet clip during a call")
	}
	if companionChildRegion(companionW, companionIdleH, parentW, parentH, false) != companionChildSkip {
		t.Fatal("idle must not stamp the pet clip onto a not-yet-resized child")
	}
	if companionChildRegion(companionW, companionIdleH, companionW, companionIdleH, false) != companionChildShape {
		t.Fatal("idle webview that matches the pet window should keep the blob region")
	}
}

func TestCompanionCallKeepsBottomRight(t *testing.T) {
	x, y := companionCallOrigin(1000, 700, companionW, companionIdleH)
	if x+companionCallW != 1000+companionW || y+companionCallH != 700+companionIdleH {
		t.Fatalf("call origin %d,%d moved the idle corner", x, y)
	}
}

func TestCompanionHitDisk(t *testing.T) {
	w, h := float64(companionW), float64(companionIdleH)
	cx, cy := w/2, float64(companionPetTop)+float64(companionPet)/2
	if !CompanionHit(cx, cy, w, h, false) {
		t.Fatal("center of the blob must hit")
	}
	if !CompanionHit(cx+40, cy-20, w, h, false) {
		t.Fatal("inside the disk must hit")
	}
	if CompanionHit(cx+96, cy, w, h, false) {
		t.Fatal("empty corners around the blob must pass through")
	}
	if CompanionHit(2, 2, w, h, false) {
		t.Fatal("top-left chrome must pass through")
	}
	if CompanionHit(w-2, 2, w, h, false) {
		t.Fatal("top-right chrome must pass through")
	}
	if CompanionHit(2, h-2, w, h, false) {
		t.Fatal("bottom corner without a caption must pass through")
	}
}

func TestCompanionIdleWindowCoversPaint(t *testing.T) {
	w, h := float64(companionW), float64(companionIdleH)
	rx, ry, rw, rh := companionPetRegionRect(w, h, false)
	paintTop := float64(companionPetTop) + float64(companionPet)*companionBlobInset
	paintBot := float64(companionPetTop) + float64(companionPet)*(1-companionBlobInset)
	boxTop := float64(companionPetTop)
	boxBot := float64(companionPetTop + companionPet)
	if ry > boxTop+0.5 {
		t.Fatalf("idle region clips the pet box crown: top=%.1f boxTop=%.1f", ry, boxTop)
	}
	if ry+rh < boxBot-0.5 {
		t.Fatalf("idle region clips the pet box chin: bot=%.1f boxBot=%.1f", ry+rh, boxBot)
	}
	if ry > paintTop+0.5 || ry+rh < paintBot-0.5 {
		t.Fatalf("idle region misses paint %.1f–%.1f vs %.1f–%.1f", paintTop, paintBot, ry, ry+rh)
	}
	if rh < float64(companionPet)*0.9 {
		t.Fatalf("idle region too short: h=%.1f", rh)
	}
	if !CompanionHit(w/2, paintTop+4, w, h, false) {
		t.Fatal("crown of idle pet must hit")
	}
	if !CompanionHit(w/2, paintBot-4, w, h, false) {
		t.Fatal("chin of idle pet must hit")
	}
	if rx <= 2 || rx+rw >= w-2 {
		t.Fatalf("region ate the side chrome: x=%.1f w=%.1f", rx, rw)
	}
}

func TestCompanionShapeLeavesChrome(t *testing.T) {
	w, h := float64(companionW), float64(companionIdleH)
	x, y, rw, rh := companionPetRegionRect(w, h, false)
	cx := w * 0.5
	cy := float64(companionPetTop) + float64(companionPet)/2
	if cx < x || cx > x+rw || cy < y || cy > y+rh {
		t.Fatal("blob center must sit inside the window region")
	}
	if x <= 2 && y <= 2 {
		t.Fatal("top-left chrome must sit outside the window region")
	}
	if x+rw >= w-2 && y <= 2 {
		t.Fatal("top-right chrome must sit outside the window region")
	}
	if x <= 2 {
		t.Fatal("left chrome beside the blob must sit outside the window region")
	}
	if x <= 2 && y+rh >= h-2 {
		t.Fatal("bottom-left chrome must sit outside the window region")
	}
	paintTop := float64(companionPetTop) + float64(companionPet)*companionBlobInset
	paintBot := float64(companionPetTop) + float64(companionPet)*(1-companionBlobInset)
	if y > paintTop {
		t.Fatalf("region clips the crown: top=%.1f paintTop=%.1f", y, paintTop)
	}
	if y+rh < paintBot {
		t.Fatalf("region clips the chin: bot=%.1f paintBot=%.1f", y+rh, paintBot)
	}
}

func TestCompanionRegionCoversBlobHips(t *testing.T) {
	w, h := float64(companionW), float64(companionIdleH)
	x, y, rw, rh := companionPetRegionRect(w, h, false)
	_, _, s := companionScale(w, h, false)
	rad := companionRegionCornerRadius(rw, rh, s)
	if rad*2 >= min(rw, rh)*0.9 {
		t.Fatalf("HWND corner %v is a circle clip of %v×%v", rad, rw, rh)
	}
	inset := float64(companionPet) * companionBlobInset
	boxX := (w - float64(companionPet)) / 2
	boxY := float64(companionPetTop)
	hips := [][2]float64{
		{boxX + inset, boxY + inset},
		{boxX + float64(companionPet) - inset, boxY + inset},
		{boxX + inset, boxY + float64(companionPet) - inset},
		{boxX + float64(companionPet) - inset, boxY + float64(companionPet) - inset},
		{w / 2, boxY + 2},
		{w / 2, boxY + float64(companionPet) - 2},
	}
	for _, p := range hips {
		if !roundRectContains(p[0], p[1], x, y, rw, rh, rad) {
			t.Fatalf("blob hip %.1f,%.1f clipped by round-rect x=%.1f y=%.1f %v×%v r=%.1f", p[0], p[1], x, y, rw, rh, rad)
		}
	}
}

func TestCompanionHitCaption(t *testing.T) {
	w, h := float64(companionW), float64(companionFrameH(true))
	bx := w / 2
	by := float64(companionPetTop+companionPet+companionBubbleGap) + float64(companionBubbleH)/2
	if CompanionHit(bx, by, w, float64(companionIdleH), false) {
		t.Fatal("caption slot is empty while hidden")
	}
	if !CompanionHit(bx, by, w, h, true) {
		t.Fatal("visible caption must hit")
	}
	if CompanionHit(2, by, w, h, true) {
		t.Fatal("chrome beside a visible caption must pass through")
	}
	if CompanionHit(w-2, by, w, h, true) {
		t.Fatal("right chrome beside a visible caption must pass through")
	}
}

func TestCompanionCaptionGrowsWithContent(t *testing.T) {
	setCompanionCaptionBox(0, 0)
	t.Cleanup(func() { setCompanionCaptionBox(0, 0) })
	w := float64(companionW)
	h1 := float64(companionFrameH(true))
	_, y1, cw1, ch1 := companionCaptionRect(w, h1, true)
	if cw1 < float64(companionBubbleW)-1 || ch1 < float64(companionBubbleH)-1 {
		t.Fatalf("default caption %v×%v", cw1, ch1)
	}
	setCompanionCaptionBox(200, 88)
	h2 := float64(companionFrameH(true))
	_, y2, cw2, ch2 := companionCaptionRect(w, h2, true)
	if y2 != y1 {
		t.Fatalf("caption top moved %v → %v", y1, y2)
	}
	if cw2 < 199 || ch2 < 87 {
		t.Fatalf("measured caption %v×%v", cw2, ch2)
	}
	contentBot := float64(companionPetTop+companionPet+companionBubbleGap) + 88
	if y2+ch2 < contentBot+float64(companionCaptionPadBot)-0.5 {
		t.Fatalf("caption region clips the bottom: bot=%.1f content=%.1f", y2+ch2, contentBot)
	}
	by := y2 + ch2/2
	if !CompanionHit(w/2, by, w, h2, true) {
		t.Fatal("tall caption must remain hittable")
	}
	if CompanionHit(2, by, w, h2, true) {
		t.Fatal("chrome beside a tall caption must pass through")
	}
}

func TestCompanionCaptionStaysOnScreen(t *testing.T) {
	setCompanionCaptionBox(220, 140)
	t.Cleanup(func() { setCompanionCaptionBox(0, 0) })
	boxes := []hitBox{{0, 0, 1920, 1080}}
	x := 1920 - companionW - companionPad
	y := 1080 - companionIdleH - companionPad
	h := companionFrameH(true)
	_, ny := clampCompanionTo(x, y, companionW, h, companionBottomClearance, boxes, 0, 0, 1920, 1080)
	if ny+h > 1080-companionBottomClearance {
		t.Fatalf("caption clipped at bottom: y=%d h=%d", ny, h)
	}
	if companionFrameH(false) != companionIdleH {
		t.Fatalf("idle frame %d", companionFrameH(false))
	}
	if h <= companionIdleH {
		t.Fatalf("caption frame should grow, h=%d", h)
	}
}

func TestClampCompanionToNearestWorkArea(t *testing.T) {
	boxes := []hitBox{
		{0, 0, 1920, 1080},
		{1920, 200, 800, 800},
	}
	// Gap above the second monitor, right of the first.
	x, y := clampCompanionTo(2000, 20, companionW, companionH, companionPad, boxes, 0, 0, 2720, 1080)
	onFirst := winsize.FullyInside(x, y, companionW, companionH, boxes[0].x, boxes[0].y, boxes[0].w, boxes[0].h)
	onSecond := winsize.FullyInside(x, y, companionW, companionH, boxes[1].x, boxes[1].y, boxes[1].w, boxes[1].h)
	if !onFirst && !onSecond {
		t.Fatalf("gap %d,%d must snap onto a monitor", x, y)
	}
	if y < 8 {
		t.Fatalf("must not sit above the work area, y=%d", y)
	}
}

func TestClampCompanionStaysOnPrimary(t *testing.T) {
	boxes := []hitBox{{0, 0, 1920, 1080}}
	x, y := clampCompanionTo(1900, 1000, companionW, companionH, companionPad, boxes, 0, 0, 1920, 1080)
	if x+companionW > 1920-companionPad {
		t.Fatalf("right edge %d", x)
	}
	if y+companionH > 1080-companionPad {
		t.Fatalf("bottom edge %d", y)
	}
}
