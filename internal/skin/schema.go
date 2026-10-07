package skin

import (
	"strings"
	"unicode"
)

type Kind string

const (
	KindColor     Kind = "color"
	KindDimension Kind = "dimension"
	KindShadow    Kind = "shadow"
	KindNumber    Kind = "number"
)

type Group string

const (
	GroupCanvas   Group = "canvas"
	GroupChrome   Group = "chrome"
	GroupText     Group = "text"
	GroupEvidence Group = "evidence"
	GroupFeedback Group = "feedback"
	GroupMotion   Group = "motion"
)

type Spec struct {
	Name  string
	Kind  Kind
	Group Group
}

const (
	CapTokens    = "tokens"
	CapWallpaper = "wallpaper"
	CapFonts     = "fonts"
	CapMaterials = "materials"
)

var KnownCaps = []string{CapTokens, CapWallpaper, CapFonts, CapMaterials}

// Specs is the closed token contract. Skins may only set these names.
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

var evidenceNames = map[string]struct{}{
	"--accent": {}, "--accent-fg": {}, "--ring": {},
	"--mark": {}, "--mark-deep": {},
	"--ctx-chat": {}, "--ctx-tools": {},
	"--success": {}, "--warning": {}, "--danger": {}, "--ctx-dynamic": {},
}

var specByName = func() map[string]Spec {
	m := make(map[string]Spec, len(Specs))
	for _, s := range Specs {
		m[s.Name] = s
	}
	return m
}()

func Lookup(name string) (Spec, bool) {
	s, ok := specByName[name]
	return s, ok
}

func IsEvidence(name string) bool {
	_, ok := evidenceNames[name]
	return ok
}

func KnownCap(name string) bool {
	n := strings.TrimSpace(name)
	for _, c := range KnownCaps {
		if c == n {
			return true
		}
	}
	return false
}

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
