// companion_hit.go 拥有宠物窗口的命中、字幕矩形和贴边夹取。不创建窗口，也不调用系统 API。companion.go 用这些几何决定能不能拖。

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
	// 命中圆比宠物框略小，空角可以穿透；又比 viewBox 内缩大，弹跳和倾斜仍能抓住。
	companionHitFill = 0.88
	// 占位跟着绘制走：168 像素宠物框外的圆角矩形，胯部、弹跳、飘带可以超出身体。
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

// setCompanionCaptionBox 把字幕宽高夹到不小于 0，且不超过气泡最大宽高，再写进原子量。
// w 小于 0 时写成 0，大于 companionBubbleW 时写成 companionBubbleW，然后存进 companionCaptionW。
// h 小于 0 时写成 0，大于 companionBubbleMaxH 时写成 companionBubbleMaxH，然后存进 companionCaptionH。
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

// captionBoxW 在记录的宽度小于最小字幕宽时返回默认气泡宽，大于最大宽时返回最大宽。
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

// captionBoxH 在记录的高度小于最小字幕高时返回默认气泡高，大于最大高时返回最大高。
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

// companionFrameH 在没有字幕时返回空闲高度。有字幕时加上气泡间距、字幕高度和底部留白。
// caption 为假时返回 companionIdleH。为真时用 captionBoxH 加上 companionPetTop、companionPet、companionBubbleGap 和 companionPadBottom。
func companionFrameH(caption bool) int {
	if !caption {
		return companionIdleH
	}
	return companionPetTop + companionPet + companionBubbleGap + int(captionBoxH()+0.5) + companionPadBottom
}

// companionDesignH 在通话交互中返回通话高度，否则返回 companionFrameH。
// caption 在非通话时传给 companionFrameH。通话中返回 companionCallH。
func companionDesignH(caption bool) float64 {
	if companionInteractive.Load() {
		return float64(companionCallH)
	}
	return float64(companionFrameH(caption))
}

// companionChildFit 表示子窗口区域该怎么处理：贴宠物形状、清掉旧裁剪，或跳过。
type companionChildFit int

const (
	companionChildShape companionChildFit = iota
	companionChildClear
	companionChildSkip
)

// companionChildRegion 决定一个子 HWND 怎么裁。父窗口或子窗口小于 8 像素时，通话中清掉裁剪，空闲时跳过过小的子窗口。子窗口相对父窗口过小或过大时，通话中清掉旧的宠物裁剪，空闲时跳过，避免把空闲形状印到还没跟上尺寸的页面上。尺寸接近时贴宠物形状。
// childW 小于 8，或相对 parentW 小于 4/5 或大于 6/5 时，参与过小过大判断。
// childH 小于 8，或相对 parentH 小于 4/5 或大于 6/5 时，参与过小过大判断。
// parentW 小于 8 时返回 companionChildShape。否则作为 4/5 和 6/5 的比较基准。
// parentH 小于 8 时返回 companionChildShape。否则作为 4/5 和 6/5 的比较基准。
// inCall 在子窗口小于 8，或相对父窗口过小、过大时，为真返回 companionChildClear，为假返回 companionChildSkip。
func companionChildRegion(childW, childH, parentW, parentH int, inCall bool) companionChildFit {
	if parentW < 8 || parentH < 8 {
		return companionChildShape
	}
	if childW < 8 || childH < 8 {
		if inCall {
			return companionChildClear
		}
		return companionChildSkip
	}
	undersized := childW < parentW*4/5 || childH < parentH*4/5
	oversized := childW > parentW*6/5 || childH > parentH*6/5
	if !undersized && !oversized {
		return companionChildShape
	}
	if inCall {
		return companionChildClear
	}
	return companionChildSkip
}

type hitBox struct {
	x, y, w, h int
}

// CompanionHit 在窗口本地坐标落在宠物圆盘上时返回真。字幕打开时，圆盘下方的字幕块也算命中。通话中整张卡片都是拖动面。坐标只要和窗口宽高同一套单位即可。窗口小于 8 时返回假。
// lx 在通话中要落在 0 到 winW 之间。空闲时相对宠物圆心算距离，字幕打开时还要落在字幕块内。
// ly 在通话中要落在 0 到 winH 之间。空闲时同样参与圆心距离和字幕块判断。
// winW 小于 8 时返回假。通话中作为右边界。空闲时用来算缩放和字幕块横向位置。
// winH 小于 8 时返回假。通话中作为下边界。空闲时传给 companionDesignH。
// caption 传给 companionDesignH。为假时圆盘未命中就返回假。为真时再测字幕块。
func CompanionHit(lx, ly, winW, winH float64, caption bool) bool {
	if winW < 8 || winH < 8 {
		return false
	}
	if companionInteractive.Load() {
		return lx >= 0 && ly >= 0 && lx <= winW && ly <= winH
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

// companionScale 返回窗口相对设计宽、设计高的缩放，以及两者中较小的那个。
// winW 除以 companionW 得到 sx。
// winH 除以 companionDesignH 得到 sy。
// caption 传给 companionDesignH。
func companionScale(winW, winH float64, caption bool) (sx, sy, s float64) {
	sx = winW / float64(companionW)
	sy = winH / companionDesignH(caption)
	return sx, sy, min(sx, sy)
}

// companionBlobEllipse 给出包住宠物绘制范围的椭圆，上方留弹跳，下方留下巴，水平方向按 companionShapeFillX 放大。
// winW 用来算圆心横坐标，也和 winH、caption 传给 companionScale。
// winH 用来把下边界夹住，也传给 companionScale。
// caption 传给 companionScale。
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

// companionPetRegionRect 是宠物 HWND 占位：168 像素宠物框再加上弹跳、下巴和左右倾斜。这是圆角矩形，身体、弹跳和飘带留在区域里。左右至少留 18 倍缩放的空，避免把圆的左边削掉。
// winW 用来算水平中心，并把矩形右边界夹在窗口内。
// winH 把矩形下边界夹在窗口内。
// caption 传给 companionScale。
func companionPetRegionRect(winW, winH float64, caption bool) (x, y, w, h float64) {
	_, sy, s := companionScale(winW, winH, caption)
	pet := float64(companionPet) * s
	cx := winW * 0.5
	boxTop := float64(companionPetTop) * sy
	boxH := float64(companionPet) * sy
	padX := pet * (companionShapeFillX - 1) * 0.5
	// 多留一点，避免 1 像素的 HWND 和客户区偏差，或呼吸缩放，把圆的左边削掉。
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

// companionRegionCornerRadius 是 HWND 占位的圆角半径。它低于短边的 0.32，避免 GDI 画成圆并把胯部裁掉。半径至少 12。
// rw 与 rh 取较小值后乘 0.32，用来限制半径。
// rh 与 rw 取较小值后乘 0.32，用来限制半径。
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

// roundRectContains 判断点是否在圆角矩形内。角上按圆角半径做圆内测试，直边范围内直接算在里面。
// px 小于 x 或大于 x 加 w 时返回假。rad 不小于 1 时还用来判断是否落在左右圆角。
// py 小于 y 或大于 y 加 h 时返回假。rad 不小于 1 时还用来判断是否落在上下圆角。
// x 作为左边界。
// y 作为上边界。
// w 作为宽度。rad 的两倍大于 w 时，rad 改成 w 的一半。
// h 作为高度。rad 的两倍大于 h 时，rad 改成 h 的一半。
// rad 小于 1 时，点已在矩形内就返回真。否则参与圆角测试。
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

// companionCaptionRect 返回字幕块加上左右和上下留白后的矩形，并夹在窗口内。
// winW 用来算水平中心，并把矩形右边界夹在窗口内。
// winH 把矩形下边界夹在窗口内。
// caption 传给 companionScale。
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

// inEllipse 用归一化坐标判断点是否在椭圆内。rx 或 ry 小于 1 时返回假。
// px 减去 cx 后除以 rx。
// py 减去 cy 后除以 ry。
// cx 从 px 里减去。
// cy 从 py 里减去。
func inEllipse(px, py, cx, cy, rx, ry float64) bool {
	if rx < 1 || ry < 1 {
		return false
	}
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	return dx*dx+dy*dy <= 1
}

// dist2ToBox 返回点到矩形的平方距离。点在矩形内时为 0。
// px 小于 b.x 时改到 b.x，大于 b.x 加 b.w 时改到右边界，再与原来的 px 求差的平方。
// py 小于 b.y 时改到 b.y，大于 b.y 加 b.h 时改到下边界，再与原来的 py 求差的平方。
// b 提供矩形的 x、y、w、h。
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

// nearestBox 按平方距离找最近的矩形。宽或高不大于 0 的矩形跳过。没有候选时返回假。
// px 传给 dist2ToBox。
// py 传给 dist2ToBox。
// boxes 里宽或高不大于 0 的跳过，其余按平方距离取最近的。
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

// clampCompanionTo 先把窗口夹进中心点最近的工作区，边距是 pad，再夹进虚拟屏幕，边距是 4。
// x 先加上 w 的一半传给 nearestBox，再传给 winsize.KeepInside。
// y 先加上 h 的一半传给 nearestBox，再传给 winsize.KeepInside。
// w 参与中心点，并传给 winsize.KeepInside。
// h 参与中心点，并传给 winsize.KeepInside。
// boxes 传给 nearestBox。
// vx 传给第二次 winsize.KeepInside。
// vy 传给第二次 winsize.KeepInside。
// vw 传给第二次 winsize.KeepInside。
// vh 传给第二次 winsize.KeepInside。
func clampCompanionTo(x, y, w, h, pad int, boxes []hitBox, vx, vy, vw, vh int) (int, int) {
	if home, ok := nearestBox(x+w/2, y+h/2, boxes); ok {
		x, y = winsize.KeepInside(x, y, w, h, home.x, home.y, home.w, home.h, pad)
	}
	return winsize.KeepInside(x, y, w, h, vx, vy, vw, vh, 4)
}
