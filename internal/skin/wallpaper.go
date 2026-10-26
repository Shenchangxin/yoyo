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
func imageConfig(body []byte, name string) (image.Config, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.DecodeConfig(r)
	}
	cfg, _, err := image.DecodeConfig(r)
	return cfg, err
}

// decodeAnyImage 解码图片。扩展名是 .webp 时走 webp，否则走 image.Decode。
func decodeAnyImage(body []byte, name string) (image.Image, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.Decode(r)
	}
	img, _, err := image.Decode(r)
	return img, err
}

// fitEdge 把较长边收到 maxEdge，另一边按比例缩小，至少为 1。两边都未超过时原样返回。
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
func withExt(name, ext string) string {
	dir := path.Dir(name)
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	if dir == "." || dir == "" {
		return base + ext
	}
	return dir + "/" + base + ext
}
