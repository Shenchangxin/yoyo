package skin

import (
	"fmt"
	"strconv"
	"strings"
)

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

func mergeEvidence(base, over TokenMap) TokenMap {
	out := cloneMap(over)
	for k, v := range base {
		if IsEvidence(k) {
			out[k] = v
		}
	}
	return out
}

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
