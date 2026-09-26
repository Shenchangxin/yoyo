package desktop

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/winsize"
)

func TestCompanionHitDisk(t *testing.T) {
	w, h := float64(companionW), float64(companionH)
	cx, cy := w/2, float64(companionPetTop)+float64(companionPet)/2
	if !CompanionHit(cx, cy, w, h, false) {
		t.Fatal("center of the blob must hit")
	}
	if !CompanionHit(cx+40, cy-20, w, h, false) {
		t.Fatal("inside the disk must hit")
	}
	if CompanionHit(cx+72, cy, w, h, false) {
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

func TestCompanionHitCaption(t *testing.T) {
	w, h := float64(companionW), float64(companionH)
	bx := w / 2
	by := float64(companionPetTop+companionPet+companionBubbleGap) + float64(companionBubbleH)/2
	if CompanionHit(bx, by, w, h, false) {
		t.Fatal("caption slot is empty while hidden")
	}
	if !CompanionHit(bx, by, w, h, true) {
		t.Fatal("visible caption must hit")
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
