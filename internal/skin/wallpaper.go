// 把壁纸和预览图限制在长边以内。未超过长边的图原样返回，超过的改成 JPEG。宽乘高超过 MaxDecodePixels 则拒绝。

package skin

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"path"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	// MaxWallpaperEdge 是壁纸长边上限，值为 3840。超过的图会缩小后再保存。
	MaxWallpaperEdge = 3840
	// MaxPreviewEdge 是预览图长边上限，值为 1280。
	MaxPreviewEdge = 1280
	// MaxDecodePixels 是宽乘高的解码上限，值为 80000000。超过则拒绝解码。
	MaxDecodePixels = 80_000_000
)

// fitImage 在宽乘高超过 MaxDecodePixels 时返回错误。未超过 maxEdge 的原样返回。超过的缩小后用质量 86 编成 JPEG，扩展名改成 .jpg。maxEdge 不大于 0 时用 MaxWallpaperEdge。
// body 长度为 0 时返回错误。传给 imageConfig 和 decodeAnyImage。未缩放时原样返回。
// name 写进错误文案。未缩放时原样返回，缩放后传给 withExt，扩展名是 .jpg。
func fitImage(body []byte, name string, maxEdge int) ([]byte, string, error) {
	if len(body) == 0 {
		return nil, "", fmt.Errorf("%s is empty", name)
	}
	if maxEdge <= 0 {
		maxEdge = MaxWallpaperEdge
	}
	cfg, err := imageConfig(body, name)
	if err != nil {
		return nil, "", fmt.Errorf("%s: not a valid image", name)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, "", fmt.Errorf("%s: not a valid image", name)
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxDecodePixels {
		return nil, "", fmt.Errorf("%s: image too large to decode", name)
	}
	src, err := decodeAnyImage(body, name)
	if err != nil {
		return nil, "", fmt.Errorf("%s: not a valid image", name)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := fitEdge(w, h, maxEdge)
	if nw == w && nh == h {
		return body, name, nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 86}); err != nil {
		return nil, "", fmt.Errorf("%s: encode: %w", name, err)
	}
	return buf.Bytes(), withExt(name, ".jpg"), nil
}

// imageConfig 读取图片尺寸。扩展名是 .webp 时用 webp 解码配置。
// body 放进 bytes.NewReader。
// name 的扩展名转成小写后等于 .webp 时走 webp.DecodeConfig，否则走 image.DecodeConfig。
func imageConfig(body []byte, name string) (image.Config, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.DecodeConfig(r)
	}
	cfg, _, err := image.DecodeConfig(r)
	return cfg, err
}

// decodeAnyImage 解码图片。扩展名是 .webp 时走 webp，否则走 image.Decode。
// body 放进 bytes.NewReader。
// name 的扩展名转成小写后等于 .webp 时走 webp.Decode，否则走 image.Decode。
func decodeAnyImage(body []byte, name string) (image.Image, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.Decode(r)
	}
	img, _, err := image.Decode(r)
	return img, err
}

// fitEdge 把较长边收到 maxEdge，另一边按比例缩小，至少为 1。两边都未超过时原样返回。
// w 小于等于 maxEdge 且 h 也是时原样返回。w 大于等于 h 且需要缩小时，返回的宽是 maxEdge，高是 h 乘 maxEdge 除以 w，至少为 1。
// h 小于等于 maxEdge 且 w 也是时原样返回。h 更大且需要缩小时，返回的高是 maxEdge，宽是 w 乘 maxEdge 除以 h，至少为 1。
func fitEdge(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		return maxEdge, max(1, h*maxEdge/w)
	}
	return max(1, w*maxEdge/h), maxEdge
}

// withExt 替换文件扩展名，保留目录。
// name 取出目录和去掉扩展名的文件名。目录是 . 或空时只返回文件名加 ext。
// ext 接在去掉原扩展名的文件名后面。
func withExt(name, ext string) string {
	dir := path.Dir(name)
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	if dir == "." || dir == "" {
		return base + ext
	}
	return dir + "/" + base + ext
}
