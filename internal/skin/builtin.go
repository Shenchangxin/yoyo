// 内置外观只有深色 ink 和浅色 neutral。dim、slate 解析成 ink，paper、mist 解析成 neutral。

package skin

// cloneMap 复制一份 TokenMap。
// src 的键和值复制进新的 TokenMap。
func cloneMap(src TokenMap) TokenMap {
	out := make(TokenMap, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// mergeMap 先复制 base，再用 over 覆盖同名键。
func mergeMap(base, over TokenMap) TokenMap {
	out := cloneMap(base)
	for k, v := range over {
		out[k] = v
	}
	return out
}

// ink 是深色主题的完整令牌。
var ink = TokenMap{
	"--background":       "#161922",
	"--foreground":       "#f4f5f7",
	"--sidebar":          "#0c0e14",
	"--panel":            "#161922",
	"--card":             "rgba(255, 255, 255, 0.075)",
	"--popover":          "rgba(28, 32, 42, 0.88)",
	"--input-bar":        "rgba(22, 26, 34, 0.72)",
	"--lift":             "rgba(255, 255, 255, 0.06)",
	"--border":           "rgba(255, 255, 255, 0.08)",
	"--muted":            "#a3a9b5",
	"--accent":           "#8b9cff",
	"--accent-fg":        "#12141c",
	"--mark":             "#f2965c",
	"--mark-deep":        "#d47b44",
	"--danger":           "#f27474",
	"--success":          "#3dd68c",
	"--warning":          "#f5b851",
	"--ring":             "#8b9cff",
	"--media-surface":    "#090b10",
	"--media-foreground": "#f4f5f7",
	"--glass-blur":       "22px",
	"--glass-bg":         "rgba(22, 26, 34, 0.62)",
	"--glass-stroke":     "rgba(255, 255, 255, 0.14)",
	"--radius":           "0.625rem",
	"--radius-lg":        "1rem",
	"--radius-composer":  "1.625rem",
	"--radius-window":    "1rem",
	"--radius-pane":      "1.5rem",
	"--radius-control":   "0.625rem",
	"--radius-chip":      "999px",
	"--shadow-card":      "0 14px 36px rgba(0, 0, 0, 0.32), inset 0 1px 0 rgba(255, 255, 255, 0.10)",
	"--shadow-composer":  "0 16px 40px rgba(0, 0, 0, 0.38), inset 0 1px 0 rgba(255, 255, 255, 0.12)",
	"--shadow-popover":   "0 24px 60px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.12)",
	"--ctx-system":       "color-mix(in srgb, var(--foreground) 38%, #a3a9b5)",
	"--ctx-tools":        "color-mix(in srgb, var(--accent) 72%, var(--muted))",
	"--ctx-dynamic":      "var(--warning)",
	"--ctx-chat":         "var(--accent)",
	"--ctx-free":         "color-mix(in srgb, var(--muted) 18%, transparent)",
}

// neutral 是浅色主题的完整令牌。
var neutral = TokenMap{
	"--background":       "#f7f8fb",
	"--foreground":       "#1c1f26",
	"--sidebar":          "#e3e7ee",
	"--panel":            "#f7f8fb",
	"--card":             "rgba(255, 255, 255, 0.94)",
	"--popover":          "rgba(255, 255, 255, 0.92)",
	"--input-bar":        "rgba(255, 255, 255, 0.78)",
	"--lift":             "rgba(28, 31, 38, 0.045)",
	"--border":           "rgba(28, 31, 38, 0.08)",
	"--muted":            "#5c6472",
	"--accent":           "#3558f5",
	"--accent-fg":        "#f7f8ff",
	"--mark":             "#c2512a",
	"--mark-deep":        "#9a3e20",
	"--danger":           "#c4313b",
	"--success":          "#0e7a4b",
	"--warning":          "#a16207",
	"--ring":             "#3558f5",
	"--media-surface":    "#12141a",
	"--media-foreground": "#f4f5f7",
	"--glass-blur":       "22px",
	"--glass-bg":         "rgba(255, 255, 255, 0.74)",
	"--glass-stroke":     "rgba(255, 255, 255, 0.78)",
	"--radius":           "0.625rem",
	"--radius-lg":        "1rem",
	"--radius-composer":  "1.625rem",
	"--radius-window":    "1rem",
	"--radius-pane":      "1.5rem",
	"--radius-control":   "0.625rem",
	"--radius-chip":      "999px",
	"--shadow-card":      "0 10px 28px rgba(36, 44, 68, 0.07), inset 0 1px 0 rgba(255, 255, 255, 0.85)",
	"--shadow-composer":  "0 16px 40px rgba(36, 44, 68, 0.10), inset 0 1px 0 rgba(255, 255, 255, 0.9)",
	"--shadow-popover":   "0 24px 60px rgba(28, 34, 52, 0.14), inset 0 1px 0 rgba(255, 255, 255, 0.9)",
	"--ctx-system":       "color-mix(in srgb, var(--foreground) 22%, #5c6472)",
	"--ctx-tools":        "color-mix(in srgb, var(--accent) 55%, var(--muted))",
	"--ctx-dynamic":      "var(--warning)",
	"--ctx-chat":         "var(--accent)",
	"--ctx-free":         "color-mix(in srgb, var(--muted) 16%, transparent)",
}

// Palette 返回一份内置外观的副本。dim 和 slate 等于 ink，paper 和 mist 等于 neutral。未知 id 返回假。
func Palette(id string) (TokenMap, bool) {
	switch id {
	case "ink", "dim", "slate":
		return cloneMap(ink), true
	case "neutral", "paper", "mist":
		return cloneMap(neutral), true
	default:
		return nil, false
	}
}

// PairForBuiltin 给任何内置 id 配上深色 ink 和浅色 neutral。
func PairForBuiltin(_ string) (dark, light string) {
	return "ink", "neutral"
}

// BuiltinInfo 返回 ID 为 builtin.<id> 的 Info，PreserveEvidence 为真。
func BuiltinInfo(id string) Info {
	name := stringsTitle(id)
	dark, light := PairForBuiltin(id)
	return Info{
		ID:               "builtin." + id,
		Name:             name,
		Builtin:          true,
		PreserveEvidence: true,
		Dark:             dark,
		Light:            light,
	}
}

// stringsTitle 只把第一个字节对应的 ASCII 小写字母改成大写，其余原样。
// id 为空时原样返回。否则把第一个字节交给 stringsToUpper，再接上后面的字节。
func stringsTitle(id string) string {
	if id == "" {
		return id
	}
	return stringsToUpper(id[:1]) + id[1:]
}

// stringsToUpper 只把第一个 ASCII 小写字母改成大写。
func stringsToUpper(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}
	return string(r)
}

// BuiltinIDs 只列出深色和浅色。旧 id 仍能由 Palette 解析。
func BuiltinIDs() []string {
	return []string{"ink", "neutral"}
}
