// Generates docs/brand/yoyo.gif: a pixel wordmark with a code stream
// running through the letters, then locking back into "yoyo".
//
//	go run .
package main

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
)

const (
	scale   = 8
	gap     = 2
	padX    = 4
	padY    = 3
	frames  = 42
	hold    = 8
	delayCS = 5
)

// 5×7 pixel letters. Rows are bits, high bit on the left.
var glyphs = map[byte][7]byte{
	'y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b01000, 0b10000},
	'o': {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'0': {0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'/': {0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b00000, 0b00000},
	'{': {0b00110, 0b00100, 0b01000, 0b00100, 0b00100, 0b00100, 0b00110},
	'}': {0b01100, 0b00100, 0b00010, 0b00100, 0b00100, 0b00100, 0b01100},
	';': {0b00000, 0b00100, 0b00100, 0b00000, 0b00100, 0b01000, 0b00000},
	'=': {0b00000, 0b00000, 0b11111, 0b00000, 0b11111, 0b00000, 0b00000},
	'+': {0b00000, 0b00100, 0b00100, 0b11111, 0b00100, 0b00100, 0b00000},
}

var streamChars = []byte{'0', '1', '/', '{', '}', ';', '=', '+'}

var palette = color.Palette{
	color.NRGBA{0x0C, 0x10, 0x0E, 0xFF}, // 0 void
	color.NRGBA{0x1A, 0x3A, 0x24, 0xFF}, // 1 dim stream
	color.NRGBA{0x3D, 0xDC, 0x6E, 0xFF}, // 2 stream
	color.NRGBA{0xD8, 0xFF, 0xE4, 0xFF}, // 3 head / letter
	color.NRGBA{0x14, 0x22, 0x18, 0xFF}, // 4 letter shadow
}

func main() {
	word := []byte("yoyo")
	cols := padX*2 + len(word)*5 + (len(word)-1)*gap
	rows := padY*2 + 7
	w, h := cols*scale, rows*scale

	mask := wordMask(word, cols, rows)
	anim := &gif.GIF{LoopCount: 0}
	var rest *image.Paletted
	for f := 0; f < frames; f++ {
		frame := image.NewPaletted(image.Rect(0, 0, w, h), palette)
		paint(frame, cols, rows, mask, f)
		if f == 0 {
			rest = frame
		}
		anim.Image = append(anim.Image, frame)
		anim.Delay = append(anim.Delay, delayCS)
	}
	for range hold {
		anim.Image = append(anim.Image, rest)
		anim.Delay = append(anim.Delay, delayCS)
	}

	dst := filepath.Join("..", "..", "docs", "brand", "yoyo.gif")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		fatal(err)
	}
	fh, err := os.Create(dst)
	if err != nil {
		fatal(err)
	}
	defer fh.Close()
	if err := gif.EncodeAll(fh, anim); err != nil {
		fatal(err)
	}
}

func wordMask(word []byte, cols, rows int) []bool {
	mask := make([]bool, cols*rows)
	x0 := padX
	for _, ch := range word {
		g := glyphs[ch]
		for y := 0; y < 7; y++ {
			for x := 0; x < 5; x++ {
				if g[y]&(1<<uint(4-x)) != 0 {
					mask[(padY+y)*cols+(x0+x)] = true
				}
			}
		}
		x0 += 5 + gap
	}
	return mask
}

func paint(dst *image.Paletted, cols, rows int, mask []bool, frame int) {
	// 0 locked word, then the stream takes over, then it locks again.
	flow := streamAmount(frame)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			idx := y*cols + x
			cell := byte(0)
			var glyph byte
			if mask[idx] {
				cell, glyph = letterCell(x, y, frame, flow)
			} else if flow > 0 {
				cell, glyph = rainCell(x, y, frame, flow)
			}
			if glyph == 0 {
				if cell != 0 {
					fill(dst, x, y, cell)
				}
			} else {
				fillGlyph(dst, x, y, cell, glyph)
			}
		}
	}
}

// streamAmount is 0 on the first and last frames so the loop holds the word.
func streamAmount(frame int) int {
	const rise = 8
	const fall = 8
	switch {
	case frame < 4:
		return 0
	case frame < 4+rise:
		return (frame - 4) * 255 / rise
	case frame > frames-fall:
		return (frames - frame) * 255 / fall
	default:
		return 255
	}
}

func letterCell(x, y, frame, flow int) (idx byte, glyph byte) {
	if flow < 40 {
		return 3, 0
	}
	// One column of the word is being written by the stream. The rest stays
	// a solid pixel so the mark is readable on every frame.
	scan := (frame * 2) % (padX*2 + 4*5 + 3*gap)
	if x == scan {
		ch := streamChars[hash(x, y, frame)%len(streamChars)]
		return 3, ch
	}
	return 3, 0
}

// 3×5 glyphs drawn inside a rain cell. High bit is the left column.
var mini = map[byte][5]byte{
	'0': {0b111, 0b101, 0b101, 0b101, 0b111},
	'1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'/': {0b001, 0b010, 0b010, 0b100, 0b100},
	'{': {0b011, 0b100, 0b110, 0b100, 0b011},
	'}': {0b110, 0b001, 0b011, 0b001, 0b110},
	';': {0b000, 0b010, 0b000, 0b010, 0b100},
	'=': {0b000, 0b111, 0b000, 0b111, 0b000},
	'+': {0b000, 0b010, 0b111, 0b010, 0b000},
}

func rainCell(x, y, frame, flow int) (idx byte, glyph byte) {
	speed := 1 + hash(x, 0, 1)%3
	shift := hash(x, 1, 2) % 11
	pos := (y + frame*speed + shift) % (12 + hash(x, 2, 3)%4)
	if pos > 7 {
		return 0, 0
	}
	ch := streamChars[hash(x, y, frame/2)%len(streamChars)]
	if pos == 0 {
		return 3, ch
	}
	if pos < 4 {
		return 2, ch
	}
	if flow < 160 {
		return 0, 0
	}
	return 1, ch
}

func fill(dst *image.Paletted, gx, gy int, idx byte) {
	x0, y0 := gx*scale, gy*scale
	// One pixel of gutter so the blocks read as pixels, not a smear.
	for y := 1; y < scale-1; y++ {
		for x := 1; x < scale-1; x++ {
			dst.SetColorIndex(x0+x, y0+y, idx)
		}
	}
}

func fillGlyph(dst *image.Paletted, gx, gy int, idx, glyph byte) {
	g, ok := mini[glyph]
	if !ok {
		fill(dst, gx, gy, idx)
		return
	}
	x0, y0 := gx*scale, gy*scale
	for y := 0; y < 5; y++ {
		for x := 0; x < 3; x++ {
			if g[y]&(1<<uint(2-x)) == 0 {
				continue
			}
			px, py := x0+2+x, y0+1+y
			if image.Pt(px, py).In(dst.Bounds()) {
				dst.SetColorIndex(px, py, idx)
			}
		}
	}
}

func hash(a, b, c int) int {
	n := a*374761393 + b*668265263 + c*2246822519
	n = (n ^ (n >> 13)) * 1274126177
	return n & 0x7fffffff
}

func fatal(err error) {
	os.Stderr.WriteString(err.Error() + "\n")
	os.Exit(1)
}
