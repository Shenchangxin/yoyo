// yoyo-mark 生成 docs/brand/yoyo.gif：像素字 yoyo 里穿过代码流，再锁回 yoyo。不改其他品牌文件。在本目录用 go run 调用。

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

// 5×7 像素字母。每一行是位，高位在左。
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
	color.NRGBA{0x0C, 0x10, 0x0E, 0xFF}, // 0 底
	color.NRGBA{0x1A, 0x3A, 0x24, 0xFF}, // 1 暗的代码流
	color.NRGBA{0x3D, 0xDC, 0x6E, 0xFF}, // 2 代码流
	color.NRGBA{0xD8, 0xFF, 0xE4, 0xFF}, // 3 字头或字母
	color.NRGBA{0x14, 0x22, 0x18, 0xFF}, // 4 字母阴影
}

// main 按帧画出 yoyo，再把第一帧多放 hold 次，写到 docs/brand/yoyo.gif。
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

// wordMask 按 5 乘 7 的字形把单词画进布尔网格。
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

// paint 按这一帧的流量把字模和代码雨画进调色板图像。
func paint(dst *image.Paletted, cols, rows int, mask []bool, frame int) {
	// 流量为 0 时锁住单词，然后代码流盖过来，最后再锁回去。
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

// streamAmount 在 frame 小于 4 时返回 0，小于 4 加 rise 时升向 255，大于 frames 减 fall 时降回 0，其余返回 255。rise 和 fall 都是 8。
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

// letterCell 在流量小于 40 时返回实心字母色。扫描列上的格子改成代码字符。
func letterCell(x, y, frame, flow int) (idx byte, glyph byte) {
	if flow < 40 {
		return 3, 0
	}
	// 单词有一列正在被代码流改写，其余保持实心像素，这样每一帧都能认出标记。
	scan := (frame * 2) % (padX*2 + 4*5 + 3*gap)
	if x == scan {
		ch := streamChars[hash(x, y, frame)%len(streamChars)]
		return 3, ch
	}
	return 3, 0
}

// 雨点格子里的 3×5 字形。高位在左列。
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

// rainCell 按列滚动代码字符。位置大于 7 时不画。流量小于 160 时不画最淡的拖尾。
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

// fill 用调色板下标填一块格子，四边各留 1 个像素。
func fill(dst *image.Paletted, gx, gy int, idx byte) {
	x0, y0 := gx*scale, gy*scale
	// 留出一像素缝，让色块看起来像像素，而不是糊成一片。
	for y := 1; y < scale-1; y++ {
		for x := 1; x < scale-1; x++ {
			dst.SetColorIndex(x0+x, y0+y, idx)
		}
	}
}

// fillGlyph 在格子里画 3 乘 5 的小字形。没有该字形时改成整块填充。
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

// hash 用三个整数算出非负散列。
func hash(a, b, c int) int {
	n := a*374761393 + b*668265263 + c*2246822519
	n = (n ^ (n >> 13)) * 1274126177
	return n & 0x7fffffff
}

// fatal 把错误写到标准错误并以状态 1 退出。
func fatal(err error) {
	os.Stderr.WriteString(err.Error() + "\n")
	os.Exit(1)
}
