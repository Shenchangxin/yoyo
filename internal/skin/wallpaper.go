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
	// Long edge stored for wallpapers. Phone/4K/5K photos are accepted then downsampled.
	MaxWallpaperEdge = 3840
	MaxPreviewEdge   = 1280
	// Decode budget: 80MP covers 48MP phone sensors with headroom; larger is treated as hostile.
	MaxDecodePixels = 80_000_000
)

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

func imageConfig(body []byte, name string) (image.Config, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.DecodeConfig(r)
	}
	cfg, _, err := image.DecodeConfig(r)
	return cfg, err
}

func decodeAnyImage(body []byte, name string) (image.Image, error) {
	r := bytes.NewReader(body)
	if strings.ToLower(path.Ext(name)) == ".webp" {
		return webp.Decode(r)
	}
	img, _, err := image.Decode(r)
	return img, err
}

func fitEdge(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		return maxEdge, max(1, h*maxEdge/w)
	}
	return max(1, w*maxEdge/h), maxEdge
}

func withExt(name, ext string) string {
	dir := path.Dir(name)
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	if dir == "." || dir == "" {
		return base + ext
	}
	return dir + "/" + base + ext
}
