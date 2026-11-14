// 内置调色板 ink、dim、slate、neutral、paper、mist。dim 和 slate 叠在 ink 上，paper 和 mist 叠在 neutral 上。

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

// ink 是深色调色板的完整令牌。
var ink = TokenMap{
	"--background":       "#13151a",
	"--foreground":       "#f2f4f8",
	"--sidebar":          "#0d0f13",
	"--panel":            "#13151a",
	"--card":             "#1a1d24",
	"--popover":          "#1f232c",
	"--input-bar":        "#1b1f28",
	"--lift":             "#242934",
	"--border":           "color-mix(in srgb, #f2f4f8 8%, transparent)",
	"--muted":            "#8e95a5",
	"--accent":           "#6e85ff",
	"--accent-fg":        "#0d0f13",
	"--mark":             "#f2965c",
	"--mark-deep":        "#d47b44",
	"--danger":           "#f27474",
	"--success":          "#3dd68c",
	"--warning":          "#f5b851",
	"--ring":             "#6e85ff",
	"--media-surface":    "#090b0f",
	"--media-foreground": "#f2f4f8",
	"--glass-blur":       "20px",
	"--glass-bg":         "color-mix(in srgb, var(--sidebar) 76%, transparent)",
	"--glass-stroke":     "color-mix(in srgb, var(--foreground) 8%, transparent)",
	"--radius":           "0.5rem",
	"--radius-lg":        "0.75rem",
	"--radius-composer":  "1.125rem",
	"--radius-window":    "0.75rem",
	"--radius-pane":      "0.75rem",
	"--radius-control":   "0.5rem",
	"--radius-chip":      "0.375rem",
	"--shadow-card":      "inset 0 0.5px 0 color-mix(in srgb, white 6%, transparent)",
	"--shadow-composer":  "0 14px 40px color-mix(in srgb, black 34%, transparent), inset 0 0.5px 0 color-mix(in srgb, white 7%, transparent)",
	"--shadow-popover":   "0 20px 52px color-mix(in srgb, black 48%, transparent), 0 0 0 1px color-mix(in srgb, white 7%, transparent)",
	"--ctx-system":       "color-mix(in srgb, var(--foreground) 38%, #8e95a5)",
	"--ctx-tools":        "color-mix(in srgb, var(--accent) 72%, var(--muted))",
	"--ctx-dynamic":      "var(--warning)",
	"--ctx-chat":         "var(--accent)",
	"--ctx-free":         "color-mix(in srgb, var(--muted) 18%, transparent)",
}

// dimOver 是叠在 ink 上的 dim 差异。
var dimOver = TokenMap{
	"--background": "#202226",
	"--foreground": "#f3f4f6",
	"--sidebar":    "#181a1e",
	"--panel":      "#202226",
	"--card":       "#282b30",
	"--popover":    "#30343b",
	"--input-bar":  "#26292f",
	"--lift":       "#343840",
	"--border":     "color-mix(in srgb, white 8%, transparent)",
	"--muted":      "#969ca8",
}

// slateOver 是叠在 ink 上的 slate 差异。
var slateOver = TokenMap{
	"--background": "#121213",
	"--foreground": "#f5f5f7",
	"--sidebar":    "#0b0b0c",
	"--panel":      "#121213",
	"--card":       "#1b1b1d",
	"--popover":    "#232326",
	"--input-bar":  "#19191b",
	"--lift":       "#27272a",
	"--border":     "color-mix(in srgb, white 8%, transparent)",
	"--muted":      "#8f9096",
	"--ctx-system": "color-mix(in srgb, var(--foreground) 32%, #8a9098)",
}

// neutral 是浅色调色板的完整令牌。
var neutral = TokenMap{
	"--background":       "#ffffff",
	"--foreground":       "#111318",
	"--sidebar":          "#f3f4f6",
	"--panel":            "#ffffff",
	"--card":             "#ffffff",
	"--popover":          "#ffffff",
	"--input-bar":        "#ffffff",
	"--lift":             "#e8ebef",
	"--border":           "color-mix(in srgb, #111318 8%, transparent)",
	"--muted":            "#646b78",
	"--accent":           "#3b5bff",
	"--accent-fg":        "#f8faff",
	"--mark":             "#d25e2c",
	"--mark-deep":        "#a3461e",
	"--danger":           "#d93842",
	"--success":          "#108956",
	"--warning":          "#c87a0a",
	"--ring":             "#3b5bff",
	"--media-surface":    "#0b0d12",
	"--media-foreground": "#f2f4f8",
	"--glass-blur":       "20px",
	"--glass-bg":         "color-mix(in srgb, var(--sidebar) 78%, transparent)",
	"--glass-stroke":     "color-mix(in srgb, var(--foreground) 7%, transparent)",
	"--radius":           "0.5rem",
	"--radius-lg":        "0.75rem",
	"--radius-composer":  "1.125rem",
	"--radius-window":    "0.75rem",
	"--radius-pane":      "0.75rem",
	"--radius-control":   "0.5rem",
	"--radius-chip":      "0.375rem",
	"--shadow-card":      "0 1px 2px color-mix(in srgb, #111318 4%, transparent)",
	"--shadow-composer":  "0 14px 38px color-mix(in srgb, #111318 8%, transparent)",
	"--shadow-popover":   "0 20px 48px color-mix(in srgb, #111318 11%, transparent), 0 0 0 1px color-mix(in srgb, #111318 6%, transparent)",
	"--ctx-system":       "color-mix(in srgb, var(--foreground) 22%, #646b78)",
	"--ctx-tools":        "color-mix(in srgb, var(--accent) 55%, var(--muted))",
	"--ctx-dynamic":      "var(--warning)",
	"--ctx-chat":         "var(--accent)",
	"--ctx-free":         "color-mix(in srgb, var(--muted) 16%, transparent)",
}

// paperOver 是叠在 neutral 上的 paper 差异。
var paperOver = TokenMap{
	"--background": "#f6f3ec",
	"--foreground": "#1a1916",
	"--sidebar":    "#eeeae2",
	"--panel":      "#f6f3ec",
	"--card":       "#fffdf8",
	"--popover":    "#fffdf8",
	"--input-bar":  "#fffdf8",
	"--lift":       "#e7e2d8",
	"--muted":      "#6d6a62",
}

// mistOver 是叠在 neutral 上的 mist 差异。
var mistOver = TokenMap{
	"--background":      "#f6f8fa",
	"--foreground":      "#1b1f24",
	"--sidebar":         "#eef1f4",
	"--panel":           "#f6f8fa",
	"--card":            "#ffffff",
	"--popover":         "#ffffff",
	"--input-bar":       "#ffffff",
	"--lift":            "#e6eaee",
	"--border":          "color-mix(in srgb, #1b1f24 7%, transparent)",
	"--muted":           "#6a7178",
	"--shadow-composer": "0 12px 36px color-mix(in srgb, #1b1f24 6%, transparent)",
	"--shadow-popover":  "0 18px 44px color-mix(in srgb, #1b1f24 9%, transparent), 0 0 0 1px color-mix(in srgb, #1b1f24 5%, transparent)",
	"--ctx-system":      "color-mix(in srgb, var(--foreground) 22%, #7a8aa0)",
}

// Palette 返回一份内置调色板的副本。dim 和 slate 由 ink 合并而来，paper 和 mist 由 neutral 合并而来。未知 id 返回假。
func Palette(id string) (TokenMap, bool) {
	switch id {
	case "ink":
		return cloneMap(ink), true
	case "dim":
		return mergeMap(ink, dimOver), true
	case "slate":
		return mergeMap(ink, slateOver), true
	case "neutral":
		return cloneMap(neutral), true
	case "paper":
		return mergeMap(neutral, paperOver), true
	case "mist":
		return mergeMap(neutral, mistOver), true
	default:
		return nil, false
	}
}

// PairForBuiltin 给 ink、dim、slate 配上浅色 neutral，给 neutral、paper、mist 配上深色 ink。其他 id 返回 ink 和 neutral。
func PairForBuiltin(id string) (dark, light string) {
	switch id {
	case "ink", "dim", "slate":
		return id, "neutral"
	case "neutral", "paper", "mist":
		return "ink", id
	default:
		return "ink", "neutral"
	}
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

// BuiltinIDs 按 ink、dim、slate、neutral、paper、mist 的顺序返回。
func BuiltinIDs() []string {
	return []string{"ink", "dim", "slate", "neutral", "paper", "mist"}
}
