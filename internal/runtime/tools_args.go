// 从模型输出的工具参数里恢复 JSON。字面换行会修好，不算截断。靠补括号或捞字段才恢复的载荷，写文件时拒绝落盘。单次正文上限由 writeFilePartMax（32 * 1024 字节）执行，错误文本里写成大约 32 KB。

package runtime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseToolArgs 丢掉截断标记，只返回参数对象。模型常在 write_file 或 apply_patch 里写出字面换行，偶尔还有截断的尾巴。吞掉 Unmarshal 错误会让它们看起来像空路径或空补丁。
func parseToolArgs(raw string) map[string]any {
	args, _ := parseToolArgsStatus(raw)
	return args
}

// toolArgsTruncated 在载荷不完整、只能靠补上括号或捞出字符串字段才恢复时返回真。JSON 字符串里的字面换行会被修好，不算截断。
func toolArgsTruncated(raw string) bool {
	_, truncated := parseToolArgsStatus(raw)
	return truncated
}

// parseToolArgsStatus 先去掉围栏。原文能解析则截断为假。修好字符串里的控制字符后能解析也算未截断。补括号或捞字段才成功则截断为真。都失败则返回空对象且截断为假。
func parseToolArgsStatus(raw string) (map[string]any, bool) {
	raw = stripJSONFence(strings.TrimSpace(raw))
	if raw == "" {
		return map[string]any{}, false
	}
	if args, ok := unmarshalObject(raw); ok {
		return args, false
	}
	repaired := repairJSONStrings(raw)
	if args, ok := unmarshalObject(repaired); ok {
		return args, false
	}
	if repaired != raw {
		raw = repaired
	}
	closed := closeTruncatedJSON(raw)
	if closed != raw {
		if args, ok := unmarshalObject(closed); ok {
			return args, true
		}
	}
	if salvaged := salvageJSONObject(raw); len(salvaged) > 0 {
		return unwrapToolArgs(salvaged), true
	}
	return map[string]any{}, false
}

// unwrapToolArgs 把嵌套的 arguments 对象或它的 JSON 字符串抬上来，避免 write_file 看到空路径。参数里已有 path、command、prompt、patch、pattern、url、name、query 之一则不再抬。
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

// hasToolArg 在 path、command、prompt、patch、pattern、url、name、query 任一去掉空白后非空时返回真。
func hasToolArg(args map[string]any) bool {
	for _, key := range []string{"path", "command", "prompt", "patch", "pattern", "url", "name", "query"} {
		if strings.TrimSpace(str(args[key])) != "" {
			return true
		}
	}
	return false
}

// unmarshalObject 只接受 JSON 对象。失败或得到 nil 时返回假。
func unmarshalObject(raw string) (map[string]any, bool) {
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil || args == nil {
		return nil, false
	}
	return args, true
}

// repairJSONStrings 把 JSON 字符串内部的原始换行、回车、制表符和其他小于 0x20 的字节转义。不补引号，也不发明键。
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

// closeTruncatedJSON 按括号栈补上未闭合的引号、对象和数组。字符串末尾若停在反斜杠，会先补一个 n。
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

// salvageJSONObject 从原文里捞 path、content、patch、command、old_str、new_str 这几个字符串字段。
func salvageJSONObject(raw string) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"path", "content", "patch", "command", "old_str", "new_str"} {
		if v := salvageJSONStringField(raw, key); v != "" {
			out[key] = v
		}
	}
	return out
}

// salvageJSONStringField 从第一个同名键后面的 JSON 字符串取值。结尾没有闭合引号时，只要已经有内容，也返回反转义后的前缀。
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

// unescapeJSONString 先尝试 strconv.Unquote。失败则只展开 \\n、\\r、\\t 和引号、反斜杠、斜杠。
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

// stripJSONFence 去掉首尾的 ``` 围栏。语言标记 json 会被丢掉。
func stripJSONFence(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if n := len(s); n >= 4 && (strings.HasPrefix(s, "json") || strings.HasPrefix(s, "JSON")) {
		s = strings.TrimSpace(s[4:])
	} else {
		if i := strings.IndexAny(s, "\r\n"); i >= 0 {
			s = strings.TrimSpace(s[i+1:])
		}
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// truncatedWriteErr 说明这是截断的 JSON，什么都没写。错误文本要求分段写，每段大约 32 KB。能捞到 path 时附上已恢复的字节数，以及最后一个完整的 function 标记。
func truncatedWriteErr(raw string) error {
	salvaged := salvageJSONObject(raw)
	path := str(salvaged["path"])
	content := str(salvaged["content"])
	msg := "truncated JSON arguments; refusing a partial write. Nothing was written. Do not resend the same blob: write the file in parts — the first part with write_file mode=create (or overwrite), each following part with mode=append, at most about 32 KB per call"
	if path != "" {
		msg += fmt.Sprintf(" (path=%s recovered=%d bytes", path, len(content))
		if mark := lastCompleteAnchor(content); mark != "" {
			msg += "; last complete marker: " + mark
		}
		msg += ")"
	}
	return fmt.Errorf("%s", msg)
}

// lastCompleteAnchor 返回正文里最后一个 function 名称。没有则返回空串。
func lastCompleteAnchor(s string) string {
	re := regexp.MustCompile(`(?m)^function\s+([A-Za-z_$][\w$]*)`)
	matches := re.FindAllStringSubmatch(s, -1)
	if n := len(matches); n > 0 {
		return "function " + matches[n-1][1]
	}
	return ""
}

// missingWriteArg 在原文为空或已是合法 JSON 时返回 empty。否则说明该字段无法恢复，并指出字面换行可以，未闭合引号或截断不行。
func missingWriteArg(kind, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || json.Valid([]byte(raw)) {
		return fmt.Errorf("empty %s", kind)
	}
	return fmt.Errorf("invalid JSON arguments; %s could not be recovered (literal newlines are OK; unclosed quotes or a truncated payload are not). Do not retry the same blob or python -c the body. A leftover file at that path is not this turn's write", kind)
}
