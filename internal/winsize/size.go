// Package winsize 按工作区计算窗口大小，并把物理像素换成 DIP。
// Fit 在比例小于等于 0 或大于 1 时用 DefaultFrac 0.82。工作区宽或高小于等于 0 时返回 FallbackW 1600、FallbackH 900。
// 下限是 MinW 1080、MinH 680，但不会大于工作区。Overflows 的容差 overflowSlop 是 16。
// CurrentFitRev 是 3。ShouldRefit 在溢出时为真；fitRev 大于等于 3 时为假；更旧的 fitRev 为真。第四、第五个参数没有使用。
package winsize

const (
	// MinW 是 Fit 在工作区够大时采用的最小宽度 1080。工作区更窄时会被压到工作区宽度。
	MinW = 1080
	// MinH 是 Fit 在工作区够高时采用的最小高度 680。工作区更矮时会被压到工作区高度。
	MinH = 680
	// FallbackW 是工作区宽或高无效时的宽度 1600。
	FallbackW = 1600
	// FallbackH 是工作区宽或高无效时的高度 900。
	FallbackH = 900
	// DefaultFrac 是默认相对工作区的比例 0.82。Fit 在传入比例无效时用它。
	DefaultFrac = 0.82
	// CurrentFitRev 是当前适配版本 3。ShouldRefit 见到大于等于它的 fitRev 就保留已保存尺寸，溢出除外。
	CurrentFitRev = 3
	// overflowSlop 是 Overflows 允许超出工作区的像素，值是 16。
	overflowSlop = 16
)

// Fit 返回首次启动窗口的宽高，取工作区的 frac 比例。frac 小于等于 0 或大于 1 时改用 DefaultFrac。
// aw 或 ah 小于等于 0 时返回 FallbackW 和 FallbackH。结果不低于 MinW、MinH，也不大于工作区。
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

// Overflows 报告窗口是否比工作区更大。aw 大于 0 且 w 大于 aw 加 16，或 ah 大于 0 且 h 大于 ah 加 16 时为真。
func Overflows(w, h, aw, ah int) bool {
	if aw > 0 && w > aw+overflowSlop {
		return true
	}
	if ah > 0 && h > ah+overflowSlop {
		return true
	}
	return false
}

// ToDIP 把物理像素换成设备无关像素。dpi 小于等于 96 时原样返回，负数也保留。否则按 px * 96 / dpi 取整。
func ToDIP(px, dpi int) int {
	if dpi <= 96 {
		return px
	}
	return int(float64(px) * 96 / float64(dpi))
}

// hideSentinel 是 HiddenSentinel 的阈值，坐标小于等于 -10000 就算藏到桌面外。
const hideSentinel = -10000

// HiddenSentinel 在 x 或 y 小于等于 -10000 时返回 true。
func HiddenSentinel(x, y int) bool {
	return x <= hideSentinel || y <= hideSentinel
}

// LostOffscreen 报告保存的窗口矩形是否看不清落在虚拟桌面上。
// HiddenSentinel 为真，或宽、高小于等于 0，返回 true。vw 或 vh 小于等于 0 时只认隐藏哨兵，返回 false，从而留住第二块显示器上的窗口。
// 否则与虚拟桌面相交的宽小于 80 或高小于 40 就返回 true。
// x 传给 HiddenSentinel，并与 vx 取较大值，与 x 加 w 和 vx 加 vw 取较小值。
// y 传给 HiddenSentinel，并与 vy 取较大值，与 y 加 h 和 vy 加 vh 取较小值。
// vx 与 x 取较大值，并与 vw 相加后和 x 加 w 取较小值。
// vy 与 y 取较大值，并与 vh 相加后和 y 加 h 取较小值。
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

// KeepInside 把矩形夹进盒子。盒子宽或高小于等于 0 时原样返回。pad 小于 0 当成 0。
// 盒子比矩形还小的一边，把可移动范围收成盒子原点加 pad。
// x 在 boxW 或 boxH 小于等于 0 时原样返回，否则夹在 boxX 加 pad 与 boxX 加 boxW 减 w 减 pad 之间。
// y 在 boxW 或 boxH 小于等于 0 时原样返回，否则夹在 boxY 加 pad 与 boxY 加 boxH 减 h 减 pad 之间。
// w 从横向可移动上限里减去。
// h 从纵向可移动上限里减去。
// boxX 加上 pad 得到横向下限，并参与横向上限。
// boxY 加上 pad 得到纵向下限，并参与纵向上限。
// boxW 小于等于 0 时与 boxH 一起让函数原样返回，否则参与横向上限。
// boxH 小于等于 0 时与 boxW 一起让函数原样返回，否则参与纵向上限。
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

// FullyInside 报告矩形是否整个落在盒子里，边对齐也算在内。盒子或矩形的宽高有小于等于 0 的就返回 false。
// x 要大于等于 boxX，并且 x 加 w 要小于等于 boxX 加 boxW。
// w 小于等于 0 时返回 false，否则加到 x 上再和 boxX 加 boxW 比较。
// h 小于等于 0 时返回 false，否则加到 y 上再和 boxY 加 boxH 比较。
// boxX 参与 x 大于等于 boxX，以及 x 加 w 小于等于 boxX 加 boxW。
// boxY 参与 y 大于等于 boxY，以及 y 加 h 小于等于 boxY 加 boxH。
// boxW 小于等于 0 时返回 false，否则加到 boxX 上再和 x 加 w 比较。
// boxH 小于等于 0 时返回 false，否则加到 boxY 上再和 y 加 h 比较。
func FullyInside(x, y, w, h, boxX, boxY, boxW, boxH int) bool {
	if boxW <= 0 || boxH <= 0 || w <= 0 || h <= 0 {
		return false
	}
	return x >= boxX && y >= boxY && x+w <= boxX+boxW && y+h <= boxY+boxH
}

// ShouldRefit 报告已保存尺寸是否该换成当前相对屏幕的默认值。溢出时为真。
// fitRev 大于等于 CurrentFitRev 时为假。更小的 fitRev 为真。第四、第五个参数没有读取。
// savedW 传给 Overflows。
// savedH 传给 Overflows。
// aw 传给 Overflows。
// ah 传给 Overflows。
func ShouldRefit(savedW, savedH, fitRev, _, _, aw, ah int) bool {
	if Overflows(savedW, savedH, aw, ah) {
		return true
	}
	if fitRev >= CurrentFitRev {
		return false
	}
	return true
}
