// 本文件把照片和头像压到适合存下和传给界面的大小。生图服务常给 2–4 MiB 的 PNG，手机照片更大；
// 不压的话存 blob 会失败，界面也要来回传几兆的 base64。不调用模型。

package idol

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // 注册 GIF 解码
	"image/jpeg"
	"image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 WebP 解码
)

// keepAsIs 是不需要重编码的图片上限：不超过它的原样存，界面会自己缩放。
const keepAsIs = 1 << 20

// ShrinkImage 把超过 keepAsIs 的图片缩到长边不超过 maxSide，再编码：带透明的存 PNG，其余存质量 88 的 JPEG。
// 不超过 keepAsIs 的、认不出格式的，以及重编码后反而更大的，都原样返回。
func ShrinkImage(b []byte, maxSide int) ([]byte, error) {
	if len(b) <= keepAsIs {
		return b, nil
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(b)); err != nil {
		return b, nil
	}
	src, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return b, nil
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			h = h * maxSide / w
			w = maxSide
		} else {
			w = w * maxSide / h
			h = maxSide
		}
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)
	var out bytes.Buffer
	if hasAlpha(dst) {
		if err := png.Encode(&out, dst); err != nil {
			return nil, err
		}
	} else {
		flat := image.NewRGBA(dst.Bounds())
		draw.Draw(flat, flat.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		draw.Draw(flat, flat.Bounds(), dst, image.Point{}, draw.Over)
		if err := jpeg.Encode(&out, flat, &jpeg.Options{Quality: 88}); err != nil {
			return nil, err
		}
	}
	if out.Len() >= len(b) {
		return b, nil
	}
	return out.Bytes(), nil
}

// hasAlpha 判断图里有没有不完全不透明的像素。
func hasAlpha(img *image.RGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0xff {
			return true
		}
	}
	return false
}
