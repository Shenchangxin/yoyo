// 闭合的令牌表和能力名。不在 Specs 里的名字不能通过 ValidateName。

package skin

import (
	"strings"
	"unicode"
)

// Kind 是令牌值的种类。
type Kind string

const (
	// KindColor 是颜色，值为 color。
	KindColor Kind = "color"
	// KindDimension 是尺寸，值为 dimension。
	KindDimension Kind = "dimension"
	// KindShadow 是阴影，值为 shadow。
	KindShadow Kind = "shadow"
	// KindNumber 是数字，值为 number。ValidateValue 拒绝覆盖这种令牌。
	KindNumber Kind = "number"
)

// Group 是令牌在界面上的分组。
type Group string

const (
	// GroupCanvas 是画布，值为 canvas。
	GroupCanvas Group = "canvas"
	// GroupChrome 是框架，值为 chrome。
	GroupChrome Group = "chrome"
	// GroupText 是文字，值为 text。
	GroupText Group = "text"
	// GroupEvidence 是证据色，值为 evidence。
	GroupEvidence Group = "evidence"
	// GroupFeedback 是反馈色，值为 feedback。
	GroupFeedback Group = "feedback"
	// GroupMotion 是动效，值为 motion。
	GroupMotion Group = "motion"
)

// Spec 是一个允许设置的令牌：名字、种类和分组。
type Spec struct {
	Name  string
	Kind  Kind
	Group Group
}

const (
	// CapTokens 是令牌能力，值为 tokens。
	CapTokens = "tokens"
	// CapWallpaper 是壁纸能力，值为 wallpaper。
	CapWallpaper = "wallpaper"
	// CapFonts 是字体能力，值为 fonts。
	CapFonts = "fonts"
	// CapMaterials 是材质能力，值为 materials。
	CapMaterials = "materials"
)

// KnownCaps 是允许出现在清单里的能力名。
var KnownCaps = []string{CapTokens, CapWallpaper, CapFonts, CapMaterials}

// Specs 是允许设置的全部令牌。不在这张表里的名字不能通过 ValidateName。
var Specs = []Spec{
	{Name: "--background", Kind: KindColor, Group: GroupCanvas},
	{Name: "--panel", Kind: KindColor, Group: GroupCanvas},
	{Name: "--media-surface", Kind: KindColor, Group: GroupCanvas},
	{Name: "--media-foreground", Kind: KindColor, Group: GroupCanvas},
	{Name: "--sidebar", Kind: KindColor, Group: GroupChrome},
	{Name: "--card", Kind: KindColor, Group: GroupChrome},
	{Name: "--popover", Kind: KindColor, Group: GroupChrome},
	{Name: "--input-bar", Kind: KindColor, Group: GroupChrome},
	{Name: "--lift", Kind: KindColor, Group: GroupChrome},
	{Name: "--border", Kind: KindColor, Group: GroupChrome},
	{Name: "--glass-bg", Kind: KindColor, Group: GroupChrome},
	{Name: "--glass-stroke", Kind: KindColor, Group: GroupChrome},
	{Name: "--glass-blur", Kind: KindDimension, Group: GroupChrome},
	{Name: "--shadow-card", Kind: KindShadow, Group: GroupChrome},
	{Name: "--shadow-composer", Kind: KindShadow, Group: GroupChrome},
	{Name: "--shadow-popover", Kind: KindShadow, Group: GroupChrome},
	{Name: "--foreground", Kind: KindColor, Group: GroupText},
	{Name: "--muted", Kind: KindColor, Group: GroupText},
	{Name: "--accent", Kind: KindColor, Group: GroupEvidence},
	{Name: "--accent-fg", Kind: KindColor, Group: GroupEvidence},
	{Name: "--ring", Kind: KindColor, Group: GroupEvidence},
	{Name: "--mark", Kind: KindColor, Group: GroupEvidence},
	{Name: "--mark-deep", Kind: KindColor, Group: GroupEvidence},
	{Name: "--ctx-chat", Kind: KindColor, Group: GroupEvidence},
	{Name: "--ctx-tools", Kind: KindColor, Group: GroupEvidence},
	{Name: "--success", Kind: KindColor, Group: GroupFeedback},
	{Name: "--warning", Kind: KindColor, Group: GroupFeedback},
	{Name: "--danger", Kind: KindColor, Group: GroupFeedback},
	{Name: "--ctx-dynamic", Kind: KindColor, Group: GroupFeedback},
	{Name: "--ctx-system", Kind: KindColor, Group: GroupText},
	{Name: "--ctx-free", Kind: KindColor, Group: GroupChrome},
	{Name: "--radius", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-lg", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-composer", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-window", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-pane", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-control", Kind: KindDimension, Group: GroupChrome},
	{Name: "--radius-chip", Kind: KindDimension, Group: GroupChrome},
}

// evidenceNames 是证据色令牌名。EvidenceLocked 为真时，这些名字用内置底色，不用用户值。
var evidenceNames = map[string]struct{}{
	"--accent": {}, "--accent-fg": {}, "--ring": {},
	"--mark": {}, "--mark-deep": {},
	"--ctx-chat": {}, "--ctx-tools": {},
	"--success": {}, "--warning": {}, "--danger": {}, "--ctx-dynamic": {},
}

// specByName 按名字索引 Specs。
var specByName = func() map[string]Spec {
	m := make(map[string]Spec, len(Specs))
	for _, s := range Specs {
		m[s.Name] = s
	}
	return m
}()

// Lookup 按名字返回 Spec。不在表里时返回假。
func Lookup(name string) (Spec, bool) {
	s, ok := specByName[name]
	return s, ok
}

// IsEvidence 报告该名字是不是证据色。
func IsEvidence(name string) bool {
	_, ok := evidenceNames[name]
	return ok
}

// KnownCap 报告去掉空白后的名字是否在 KnownCaps 里。
func KnownCap(name string) bool {
	n := strings.TrimSpace(name)
	for _, c := range KnownCaps {
		if c == n {
			return true
		}
	}
	return false
}

// ValidUserID 拒绝空串和 builtin. 前缀。长度须在 3 到 64，首字符是小写字母，其余是数字、小写字母、点、下划线或减号。
func ValidUserID(id string) bool {
	if id == "" || strings.HasPrefix(id, "builtin.") {
		return false
	}
	if len(id) < 3 || len(id) > 64 {
		return false
	}
	for i, r := range id {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
			continue
		}
		if unicode.IsDigit(r) || (r >= 'a' && r <= 'z') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// BuiltinPaletteID 接受 ink、dim、slate、neutral、paper、mist，以及它们的 builtin. 前缀。其他返回假。
func BuiltinPaletteID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	switch id {
	case "ink", "dim", "slate", "neutral", "paper", "mist":
		return id, true
	}
	if strings.HasPrefix(id, "builtin.") {
		p := strings.TrimPrefix(id, "builtin.")
		return BuiltinPaletteID(p)
	}
	return "", false
}

// NormalizeSkinID 对内置调色板和非法用户 id 返回空串，合法用户 id 原样返回。
func NormalizeSkinID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if _, ok := BuiltinPaletteID(id); ok {
		return ""
	}
	if !ValidUserID(id) {
		return ""
	}
	return id
}
