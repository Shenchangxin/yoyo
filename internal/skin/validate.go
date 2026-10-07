package skin

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	reHex       = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	reVar       = regexp.MustCompile(`^var\(--[a-z0-9-]+\)$`)
	reDim       = regexp.MustCompile(`^(?:0|[0-9]*\.?[0-9]+)(?:px|rem|em|%)$`)
	reOklch     = regexp.MustCompile(`^oklch\(\s*[0-9.%\s./+-]+\s*\)$`)
	reRGB       = regexp.MustCompile(`^(?:rgb|hsl)a?\(\s*[0-9.%\s,./+-]+\s*\)$`)
	reColorMix  = regexp.MustCompile(`^color-mix\(\s*in\s+srgb\s*,.+\)$`)
	reTokenName = regexp.MustCompile(`^--[a-z0-9-]+$`)
	reSafeIdent = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

func ValidateName(name string) error {
	if !reTokenName.MatchString(name) {
		return fmt.Errorf("unknown token %q", name)
	}
	if _, ok := Lookup(name); !ok {
		return fmt.Errorf("token %q is not in the Yoyo contract", name)
	}
	return nil
}

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

func nilIfSafeColorMix(inner string) error {
	// color-mix args: <color> <p%>?, <color> <p%>?
	if strings.ContainsAny(inner, ";{}<>") {
		return fmt.Errorf("invalid color-mix")
	}
	return nil
}

func validateShadow(v string) error {
	if strings.ContainsAny(v, ";{}<>") {
		return fmt.Errorf("invalid shadow")
	}
	return nil
}

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
