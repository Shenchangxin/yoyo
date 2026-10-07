package runtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// parseToolArgs recovers a tool argument object from model JSON.
// Models often emit literal newlines (and occasionally a truncated tail)
// inside write_file/apply_patch payloads. Swallowing Unmarshal errors made
// those look like "empty path" / "empty patch".
func parseToolArgs(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	if args, ok := unmarshalObject(raw); ok {
		return args
	}
	repaired := repairJSONStrings(raw)
	if repaired != raw {
		if args, ok := unmarshalObject(repaired); ok {
			return args
		}
		raw = repaired
	}
	closed := closeTruncatedJSON(raw)
	if closed != raw {
		if args, ok := unmarshalObject(closed); ok {
			return args
		}
	}
	if salvaged := salvageJSONObject(raw); len(salvaged) > 0 {
		return unwrapToolArgs(salvaged)
	}
	return map[string]any{}
}

// unwrapToolArgs lifts a model-nested {"arguments": {path, content}} object
// (or a JSON string of that object) so write_file does not see an empty path.
func unwrapToolArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return args
	}
	if hasToolArg(args) {
		return args
	}
	switch inner := args["arguments"].(type) {
	case map[string]any:
		if len(inner) > 0 {
			return unwrapToolArgs(inner)
		}
	case string:
		if parsed := parseToolArgs(inner); len(parsed) > 0 {
			return unwrapToolArgs(parsed)
		}
	}
	return args
}

func hasToolArg(args map[string]any) bool {
	for _, key := range []string{"path", "command", "prompt", "patch", "pattern", "url", "name", "query"} {
		if strings.TrimSpace(str(args[key])) != "" {
			return true
		}
	}
	return false
}

func unmarshalObject(raw string) (map[string]any, bool) {
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil || args == nil {
		return nil, false
	}
	return args, true
}

// repairJSONStrings escapes raw control characters that appear inside JSON
// strings. It does not invent quotes or keys.
func repairJSONStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 32)
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !inString {
			b.WriteByte(c)
			if c == '"' {
				inString = true
			}
			continue
		}
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			b.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			b.WriteByte(c)
			inString = false
			continue
		}
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func closeTruncatedJSON(s string) string {
	inString := false
	escaped := false
	var stack []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, c)
		case '}':
			if n := len(stack); n > 0 && stack[n-1] == '{' {
				stack = stack[:n-1]
			}
		case ']':
			if n := len(stack); n > 0 && stack[n-1] == '[' {
				stack = stack[:n-1]
			}
		}
	}
	var b strings.Builder
	b.WriteString(s)
	if escaped {
		b.WriteByte('n')
	}
	if inString {
		b.WriteByte('"')
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			b.WriteByte('}')
		} else {
			b.WriteByte(']')
		}
	}
	return b.String()
}

func salvageJSONObject(raw string) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"path", "content", "patch", "command", "old_str", "new_str"} {
		if v := salvageJSONStringField(raw, key); v != "" {
			out[key] = v
		}
	}
	return out
}

func salvageJSONStringField(raw, key string) string {
	needle := `"` + key + `"`
	i := strings.Index(raw, needle)
	if i < 0 {
		return ""
	}
	rest := strings.TrimLeft(raw[i+len(needle):], " \t\r\n")
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	inner := rest[1:]
	var b strings.Builder
	escaped := false
	for j := 0; j < len(inner); j++ {
		c := inner[j]
		if escaped {
			b.WriteByte('\\')
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			tail := strings.TrimLeft(inner[j+1:], " \t\r\n")
			if tail == "" || strings.HasPrefix(tail, ",") || strings.HasPrefix(tail, "}") || strings.HasPrefix(tail, "]") {
				return unescapeJSONString(b.String())
			}
		}
		b.WriteByte(c)
	}
	if escaped {
		b.WriteByte('\\')
	}
	if b.Len() == 0 {
		return ""
	}
	return unescapeJSONString(b.String())
}

func unescapeJSONString(s string) string {
	if u, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return u
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case '"', '\\', '/':
			b.WriteByte(s[i+1])
			i++
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func missingWriteArg(kind, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || json.Valid([]byte(raw)) {
		return fmt.Errorf("empty %s", kind)
	}
	return fmt.Errorf("invalid JSON arguments; %s could not be recovered (literal newlines are OK; unclosed quotes or a truncated payload are not). Do not retry the same blob or python -c the body. A leftover file at that path is not this turn's write", kind)
}
