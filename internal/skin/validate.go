// 检查令牌名和值。拒绝 url(、expression、@import、javascript，以及非 ASCII。

package skin

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// reHex 匹配 # 后跟 3、4、6 或 8 位十六进制。
	reHex = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	// reVar 匹配 var(--名字)。
	reVar = regexp.MustCompile(`^var\(--[a-z0-9-]+\)$`)
	// reDim 匹配带 px、rem、em 或 % 的尺寸，以及单独的 0。
	reDim = regexp.MustCompile(`^(?:0|[0-9]*\.?[0-9]+)(?:px|rem|em|%)$`)
	// reOklch 匹配 oklch()。
	reOklch = regexp.MustCompile(`^oklch\(\s*[0-9.%\s./+-]+\s*\)$`)
	// reRGB 匹配 rgb、rgba、hsl、hsla。
	reRGB = regexp.MustCompile(`^(?:rgb|hsl)a?\(\s*[0-9.%\s,./+-]+\s*\)$`)
	// reColorMix 匹配 color-mix(in srgb, ...)。
	reColorMix = regexp.MustCompile(`^color-mix\(\s*in\s+srgb\s*,.+\)$`)
	// reTokenName 匹配 -- 加小写字母、数字和减号。
	reTokenName = regexp.MustCompile(`^--[a-z0-9-]+$`)
	// reSafeIdent 匹配文件名里允许的字母、数字、点、下划线和减号。
	reSafeIdent = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidateName 要求名字匹配令牌形式，并且在 Specs 里。
func ValidateName(name string) error {
	if !reTokenName.MatchString(name) {
		return fmt.Errorf("unknown token %q", name)
	}
	if _, ok := Lookup(name); !ok {
		return fmt.Errorf("token %q is not in the Yoyo contract", name)
	}
	return nil
}

// ValidateValue 拒绝空值、超过 240 字节、非 ASCII，以及含 url(、expression、@import 或 javascript 的文本。数字种类直接拒绝覆盖。
func ValidateValue(kind Kind, raw string) error {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fmt.Errorf("empty value")
	}
	if len(v) > 240 {
		return fmt.Errorf("value too long")
	}
	if !safeRunes(v) {
		return fmt.Errorf("value contains disallowed characters")
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "url(") || strings.Contains(lower, "expression") || strings.Contains(lower, "@import") || strings.Contains(lower, "javascript") {
		return fmt.Errorf("value must not load resources or execute")
	}
	switch kind {
	case KindColor:
		return validateColor(v)
	case KindDimension:
		if !reDim.MatchString(v) {
			return fmt.Errorf("invalid dimension %q", v)
		}
		return nil
	case KindShadow:
		return validateShadow(v)
	case KindNumber:
		return fmt.Errorf("number tokens are not overridable")
	default:
		return fmt.Errorf("unknown kind")
	}
}

// ValidateMap 对映射里的每个名字和值调用 ValidateName、ValidateValue。nil 映射返回错误。
func ValidateMap(m TokenMap) error {
	if m == nil {
		return fmt.Errorf("missing token map")
	}
	for name, val := range m {
		if err := ValidateName(name); err != nil {
			return err
		}
		spec, _ := Lookup(name)
		if err := ValidateValue(spec.Kind, val); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// validateColor 接受 transparent、white、black、十六进制、var()、oklch()、rgb/hsl 和通过 nilIfSafeColorMix 的 color-mix。
func validateColor(v string) error {
	switch {
	case v == "transparent" || v == "white" || v == "black":
		return nil
	case reHex.MatchString(v), reVar.MatchString(v), reOklch.MatchString(v), reRGB.MatchString(v):
		return nil
	case reColorMix.MatchString(v):
		inner := v[strings.Index(v, ",")+1:]
		inner = strings.TrimSuffix(strings.TrimSpace(inner), ")")
		return nilIfSafeColorMix(inner)
	default:
		return fmt.Errorf("invalid color %q", v)
	}
}

// nilIfSafeColorMix 在 color-mix 的内部出现分号、花括号或尖括号时返回错误。
func nilIfSafeColorMix(inner string) error {
	// color-mix 的参数是颜色，后面可以跟百分比。
	if strings.ContainsAny(inner, ";{}<>") {
		return fmt.Errorf("invalid color-mix")
	}
	return nil
}

// validateShadow 在阴影值含分号、花括号或尖括号时返回错误。
func validateShadow(v string) error {
	if strings.ContainsAny(v, ";{}<>") {
		return fmt.Errorf("invalid shadow")
	}
	return nil
}

// safeRunes 只允许 ASCII 字母、数字和 # % . , ( ) 空格 / - _ + *。
func safeRunes(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '#', '%', '.', ',', '(', ')', ' ', '/', '-', '_', '+', '*':
			continue
		default:
			return false
		}
	}
	return true
}

// StripEvidence 返回去掉证据色键之后的副本。
func StripEvidence(m TokenMap) TokenMap {
	out := make(TokenMap, len(m))
	for k, v := range m {
		if IsEvidence(k) {
			continue
		}
		out[k] = v
	}
	return out
}
