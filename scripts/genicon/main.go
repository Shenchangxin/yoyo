// Generates the Yoyo app mark (black rounded square + white yo-yo) as PNG/ICO.
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

func mustWrite(path string, b []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		panic(err)
	}
}

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

func union(a, b float64) float64 {
	return 1 - ((1 - a) * (1 - b))
}

func roundedBox(px, py, cx, cy, hw, hh, r float64) float64 {
	dx := math.Abs(px-cx) - hw + r
	dy := math.Abs(py-cy) - hh + r
	ox := math.Max(dx, 0)
	oy := math.Max(dy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(dx, dy), 0) - r
}

func circle(px, py, cx, cy, r float64) float64 {
	return math.Hypot(px-cx, py-cy) - r
}

func aa(d float64) float64 {
	// ~1px at 1024, thicker relative coverage at small sizes
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

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

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
