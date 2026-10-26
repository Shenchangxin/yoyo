// genicon 画出圆角方块上的溜溜球标记，并写成 PNG、ICO 和 SVG。不负责安装或打包。在仓库里用 go run 调用。

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var (
	bg  = color.RGBA{0x0A, 0x0A, 0x0A, 0xFF}
	ink = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

// main 写出应用图标：1024 的 PNG、256 的桌面 PNG、iOS 用的 1024 PNG、16、32、48 和 256 的 ICO，以及 SVG。
func main() {
	root := "."
	if _, err := os.Stat("build"); err != nil {
		root = "../.."
	}
	png1024 := render(1024)
	mustWrite(filepath.Join(root, "build", "appicon.png"), png1024)
	mustWrite(filepath.Join(root, "internal", "desktop", "assets", "appicon.png"), render(256))
	mustWrite(filepath.Join(root, "build", "ios", "icon.png"), png1024)
	mustWrite(filepath.Join(root, "build", "windows", "icon.ico"), packICO(
		render(16), render(32), render(48), render(256),
	))
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <rect width="1024" height="1024" rx="224" fill="#0A0A0A"/>
  <g fill="none" stroke="#FFFFFF" stroke-linecap="round">
    <circle cx="512" cy="307" r="172" stroke-width="78"/>
    <line x1="512" y1="479" x2="512" y2="635" stroke-width="33"/>
    <path d="M225 491 L512 675 L799 491" stroke-width="126"/>
    <line x1="512" y1="675" x2="512" y2="880" stroke-width="126"/>
  </g>
  <circle cx="512" cy="307" r="49" fill="#FFFFFF"/>
</svg>
`
	mustWrite(filepath.Join(root, "build", "appicon.icon", "Assets", "wails_icon_vector.svg"), []byte(svg))
}

// mustWrite 创建目录并写文件。失败则 panic。
// path 的父目录按 0o755 创建，文件按 0o644 写入。失败则 panic。
// b 写入该文件。
func mustWrite(path string, b []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		panic(err)
	}
}

// render 按边长光栅化标记。边长不超过 32 时每个方向采样 4 次，否则每个方向采样 3 次，再编码成 PNG。
// size 作为图像宽高。小于等于 32 时每个方向采样 4 次，否则 3 次。
func render(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	samples := 3
	if size <= 32 {
		samples = 4
	}
	inv := 1.0 / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/float64(samples)) * inv
					py := (float64(y) + (float64(sy)+0.5)/float64(samples)) * inv
					c := sample(px, py)
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a += float64(c.A)
				}
			}
			n := float64(samples * samples)
			img.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// sample 在单位正方形里采样圆角底、圆环、轴和字母，并按覆盖混合底色和墨色。
// x 传给 roundedBox、circle 和 capsule。
// y 传给 roundedBox、circle 和 capsule。
func sample(x, y float64) color.RGBA {
	box := roundedBox(x, y, 0.5, 0.5, 0.5, 0.5, 0.22)
	cover := aa(box)
	if cover <= 0 {
		return color.RGBA{}
	}
	ring := math.Abs(circle(x, y, 0.50, 0.30, 0.168)) - 0.038
	axle := circle(x, y, 0.50, 0.30, 0.048)
	yoyo := union(aa(ring), aa(axle))
	str := capsule(x, y, 0.50, 0.468, 0.50, 0.62, 0.016)
	left := capsule(x, y, 0.22, 0.48, 0.50, 0.66, 0.062)
	right := capsule(x, y, 0.78, 0.48, 0.50, 0.66, 0.062)
	stem := capsule(x, y, 0.50, 0.66, 0.50, 0.86, 0.062)
	letter := union(aa(left), union(aa(right), union(aa(stem), aa(str))))
	mark := union(yoyo, letter)
	return mix(bg, ink, mark*cover, cover)
}

// capsule 返回点到一条胶囊线段的有符号距离。
// px 减去线段上的投影点，参与 Hypot。
// py 减去线段上的投影点，参与 Hypot。
// ax 是线段起点的横坐标，并参与投影。
// ay 是线段起点的纵坐标，并参与投影。
// bx 减去 ax 得到横向分量。
// by 减去 ay 得到纵向分量。
// r 从距离里减去。
func capsule(px, py, ax, ay, bx, by, r float64) float64 {
	dx := bx - ax
	dy := by - ay
	len2 := dx*dx + dy*dy
	var h float64
	if len2 > 0 {
		h = clamp(((px-ax)*dx + (py-ay)*dy) / len2)
	}
	return math.Hypot(px-(ax+dx*h), py-(ay+dy*h)) - r
}

// union 把两段覆盖合成一段。
// a 与 b 按 1 减 (1-a) 乘 (1-b) 合成。
// b 与 a 用同一公式合成。
func union(a, b float64) float64 {
	return 1 - ((1 - a) * (1 - b))
}

// roundedBox 返回点到圆角矩形的有符号距离。
// px 与 cx 的差取绝对值后参与横向距离。
// py 与 cy 的差取绝对值后参与纵向距离。
// cx 从 px 里减去。
// cy 从 py 里减去。
// hw 从横向距离里减去，再加回圆角半径。
// hh 从纵向距离里减去，再加回圆角半径。
func roundedBox(px, py, cx, cy, hw, hh, r float64) float64 {
	dx := math.Abs(px-cx) - hw + r
	dy := math.Abs(py-cy) - hh + r
	ox := math.Max(dx, 0)
	oy := math.Max(dy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(dx, dy), 0) - r
}

// circle 返回点到圆的有符号距离。
// px 减去 cx 后参与 Hypot。
// py 减去 cy 后参与 Hypot。
// cx 从 px 里减去。
// cy 从 py 里减去。
func circle(px, py, cx, cy, r float64) float64 {
	return math.Hypot(px-cx, py-cy) - r
}

// aa 把有符号距离收成 0 到 1 的覆盖。带宽是 0.0018。
// d 除以 0.0018 后从 0.5 里减去。结果小于 0 时返回 0，大于 1 时返回 1。
func aa(d float64) float64 {
	// 边长 1024 时这条带更窄，小图上覆盖相对更厚。宽度是 0.0018。
	w := 0.0018
	t := 0.5 - d/w
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// mix 按前景覆盖混合背景色和前景色。覆盖不大于 0 时返回透明色。
// bg 的 R、G、B 按 1 减 fgA 加权。
// fg 的 R、G、B 按夹紧后的 fgA 加权。
// fgA 交给 clamp 后作为前景权重。
// cover 小于等于 0 时返回透明色，否则乘 255 后作为 alpha。
func mix(bg, fg color.RGBA, fgA, cover float64) color.RGBA {
	if cover <= 0 {
		return color.RGBA{}
	}
	t := clamp(fgA)
	r := float64(bg.R)*(1-t) + float64(fg.R)*t
	g := float64(bg.G)*(1-t) + float64(fg.G)*t
	b := float64(bg.B)*(1-t) + float64(fg.B)*t
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(255 * cover)}
}

// clamp 把值夹到 0 和 1。
// v 小于 0 时返回 0，大于 1 时返回 1，否则原样返回。
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// packICO 把多张 PNG 打成 ICO。宽或高达到 256 时，目录项里的宽高字节写 0。
// pngs 的长度写入目录计数。每张解码出宽高，大于等于 256 的一边在目录里写 0，字节原样接在目录后面。
func packICO(pngs ...[]byte) []byte {
	count := uint16(len(pngs))
	var dir bytes.Buffer
	_ = binary.Write(&dir, binary.LittleEndian, uint16(0))
	_ = binary.Write(&dir, binary.LittleEndian, uint16(1))
	_ = binary.Write(&dir, binary.LittleEndian, count)
	offset := 6 + 16*len(pngs)
	var body bytes.Buffer
	for _, p := range pngs {
		cfg, err := png.DecodeConfig(bytes.NewReader(p))
		if err != nil {
			panic(err)
		}
		w, h := cfg.Width, cfg.Height
		wb, hb := byte(w), byte(h)
		if w >= 256 {
			wb = 0
		}
		if h >= 256 {
			hb = 0
		}
		dir.Write([]byte{wb, hb, 0, 0})
		_ = binary.Write(&dir, binary.LittleEndian, uint16(1))
		_ = binary.Write(&dir, binary.LittleEndian, uint16(32))
		_ = binary.Write(&dir, binary.LittleEndian, uint32(len(p)))
		_ = binary.Write(&dir, binary.LittleEndian, uint32(offset))
		body.Write(p)
		offset += len(p)
	}
	out := append(dir.Bytes(), body.Bytes()...)
	return out
}
