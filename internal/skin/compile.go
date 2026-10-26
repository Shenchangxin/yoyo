// 把用户令牌叠到内置调色板上。证据色被锁住时，用底色盖过用户的证据色。

package skin

import (
	"fmt"
	"strconv"
	"strings"
)

// Compile 把 mode 不是 light 的都当成 dark。用户令牌叠到 ink 或 neutral 上。EvidenceLocked 为真时去掉用户的证据色，再用底色的证据色盖回去。对比度不通过且处理后的用户层非空时返回错误。返回的 Tokens 是处理后的用户层，不是合并后的全表。
func Compile(p Pack, mode string) (Resolved, error) {
	if mode != "light" {
		mode = "dark"
	}
	baseID := "ink"
	if mode == "light" {
		baseID = "neutral"
	}
	base, _ := Palette(baseID)
	over := p.Dark
	if mode == "light" {
		over = p.Light
	}
	if over == nil {
		over = TokenMap{}
	}
	merged := mergeMap(base, over)
	if p.Manifest.EvidenceLocked() {
		over = StripEvidence(over)
		merged = mergeEvidence(base, merged)
	}
	merged = applyRadiusScale(merged, p.Manifest.Materials.RadiusScale)
	over = applyRadiusScale(over, p.Manifest.Materials.RadiusScale)
	if p.Manifest.Materials.GlassBlur != "" {
		merged["--glass-blur"] = p.Manifest.Materials.GlassBlur
	}
	rep := ContrastOf(merged)
	res := Resolved{
		ID:               p.Manifest.ID,
		Name:             p.Manifest.Name,
		Mode:             mode,
		PreserveEvidence: p.Manifest.EvidenceLocked(),
		Tokens:           over,
		WindowRGB:        WindowRGB(merged),
		Materials:        p.Manifest.Materials.Normalize(),
		Contrast:         rep,
	}
	if p.Manifest.Wallpaper != "" && p.HasCap(CapWallpaper) {
		res.WallpaperFile = p.Manifest.Wallpaper
		res.WallpaperURL = AssetPrefix + p.Manifest.ID + "/assets/" + p.Manifest.Wallpaper
	}
	if p.Manifest.FontSans != "" {
		res.Fonts = append(res.Fonts, FontFace{
			Family: "YoyoSkinSans",
			URL:    AssetPrefix + p.Manifest.ID + "/assets/fonts/" + p.Manifest.FontSans,
			Role:   "sans",
		})
	}
	if p.Manifest.FontMono != "" {
		res.Fonts = append(res.Fonts, FontFace{
			Family: "YoyoSkinMono",
			URL:    AssetPrefix + p.Manifest.ID + "/assets/fonts/" + p.Manifest.FontMono,
			Role:   "mono",
		})
	}
	if !rep.Pass && len(over) > 0 {
		return res, fmt.Errorf("%s", rep.Reason)
	}
	return res, nil
}

// CompileBuiltin 按调色板 id 取令牌。未知 id 在 light 下用 neutral，否则用 ink。PreserveEvidence 为真，Builtin 为真。
// palette 传给 Palette。失败时，mode 为 light 则改用 neutral 并把 palette 改成 neutral，否则改用 ink 并把 palette 改成 ink。
// mode 不是 light 时改成 dark。
func CompileBuiltin(palette, mode string) Resolved {
	if mode != "light" {
		mode = "dark"
	}
	tokens, ok := Palette(palette)
	if !ok {
		if mode == "light" {
			tokens, _ = Palette("neutral")
			palette = "neutral"
		} else {
			tokens, _ = Palette("ink")
			palette = "ink"
		}
	}
	return Resolved{
		ID:               "builtin." + palette,
		Name:             stringsTitle(palette),
		Builtin:          true,
		Mode:             mode,
		PreserveEvidence: true,
		Tokens:           tokens,
		WindowRGB:        WindowRGB(tokens),
		Materials:        Materials{Grain: 0.04, WallpaperFit: "cover", WallpaperDim: 0.45, RadiusScale: 1},
		Contrast:         ContrastOf(tokens),
	}
}

// mergeEvidence 复制 over，再把 base 里属于证据色的键盖回去。
func mergeEvidence(base, over TokenMap) TokenMap {
	out := cloneMap(over)
	for k, v := range base {
		if IsEvidence(k) {
			out[k] = v
		}
	}
	return out
}

// applyRadiusScale 在 scale 小于等于 0 或等于 1 时原样返回。否则缩放以 --radius 开头的那一组尺寸。
// tokens 在 scale 小于等于 0 或等于 1 时原样返回，否则复制后改写 --radius、--radius-lg、--radius-composer、--radius-window、--radius-pane、--radius-control、--radius-chip。
func applyRadiusScale(tokens TokenMap, scale float64) TokenMap {
	if scale <= 0 || scale == 1 {
		return tokens
	}
	out := cloneMap(tokens)
	for _, name := range []string{"--radius", "--radius-lg", "--radius-composer", "--radius-window", "--radius-pane", "--radius-control", "--radius-chip"} {
		v, ok := out[name]
		if !ok {
			continue
		}
		if scaled, ok := scaleDim(v, scale); ok {
			out[name] = scaled
		}
	}
	return out
}

// scaleDim 只缩放带 rem、px 或 em 后缀的数。没有这些后缀时返回假。
// v 去掉首尾空白。后缀是 rem、px 或 em 时去掉后缀再解析成浮点数。
func scaleDim(v string, scale float64) (string, bool) {
	v = strings.TrimSpace(v)
	unit := ""
	switch {
	case strings.HasSuffix(v, "rem"):
		unit = "rem"
	case strings.HasSuffix(v, "px"):
		unit = "px"
	case strings.HasSuffix(v, "em"):
		unit = "em"
	default:
		return "", false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(v, unit)), 64)
	if err != nil {
		return "", false
	}
	return strconv.FormatFloat(n*scale, 'f', 4, 64) + unit, true
}
