// Copyright 2026 Hypit.AI
// Translated from @hypit/source/src/header.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package source

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

type SourceHeader struct {
	// Using is the logical Frontend request. A trusted Host resolves it to one exact implementation.
	Using string
	Start int
	End   int
}

type SourceHeaderError struct {
	Code       string
	Message    string
	SourceName string
	Offset     int
	Line       int
	Column     int
	Name       string
}

func (e *SourceHeaderError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func fail(sourceName, source, code, message string, offset int) {
	panic(newSourceHeaderError(code, message, sourceName, source, offset))
}

func newSourceHeaderError(code, message, sourceName, source string, offset int) *SourceHeaderError {
	units := utf16Units(source)
	end := offset
	if end < 0 {
		end = 0
	}
	if end > len(units) {
		end = len(units)
	}
	line, column := lineAndColumn(units[:end])
	return &SourceHeaderError{
		Code:       code,
		Message:    fmt.Sprintf("%s:%d:%d: %s", sourceName, line, column, message),
		SourceName: sourceName,
		Offset:     offset,
		Line:       line,
		Column:     column,
		Name:       "SourceHeaderError",
	}
}

// ParseSourceHeader parses the single irreducible SVML bootstrap directive.
//
// The grammar deliberately has no comments, aliases, interpolation or defaults:
//
//	[UTF-8 BOM] <?svml using="logical-frontend-request"?>
func ParseSourceHeader(sourceName, text string) SourceHeader {
	units := utf16Units(text)
	start := 0
	if len(units) > 0 && units[0] == 0xfeff {
		start = 1
	}
	if !hasPrefixAt(units, start, "<?svml") {
		fail(sourceName, text, "SOURCE_HEADER_MISSING", `Source must begin with <?svml using="..."?>`, start)
	}
	closing := indexFrom(units, start+len("<?svml"), "?>")
	if closing < 0 {
		fail(sourceName, text, "SOURCE_HEADER_UNCLOSED", "Source Header is not closed", start)
	}
	raw := utf16Decode(units[start : closing+2])
	using, ok := matchUsing(raw)
	if !ok {
		fail(sourceName, text, "SOURCE_HEADER_INVALID", `Source Header must use exactly <?svml using="logical-frontend-request"?>`, start)
	}
	if !frontendRequest(using) {
		fail(sourceName, text, "SOURCE_HEADER_FRONTEND", "Source Header Frontend request must not be empty or padded", start)
	}
	end := closing + 2
	after := units[end:]
	next := indexNonSpace(after)
	if next >= 0 && hasPrefixAt(after, next, "<?svml") {
		fail(sourceName, text, "SOURCE_HEADER_DUPLICATE", "Source declares more than one Source Header", end+next)
	}
	return SourceHeader{Using: using, Start: start, End: end}
}

// MaskSourceHeader preserves every original offset while making the Header ordinary whitespace to body Frontends.
func MaskSourceHeader(text string, header SourceHeader) string {
	units := utf16Units(text)
	prefix := append([]uint16(nil), sliceUnits(units, 0, header.End, true)...)
	// Source ranges use UTF-16 offsets: replace each code unit, not each code point.
	for i, unit := range prefix {
		if unit != '\r' && unit != '\n' {
			prefix[i] = ' '
		}
	}
	tail := sliceUnits(units, header.End, 0, false)
	return utf16Decode(append(prefix, tail...))
}

type SelfDescribedTextSource struct {
	ID   string
	Name string
	Text string
}

// ResolveSelfDescribedTextSource is the optional adapter for Hypit's self-described UTF-8 author-file convention.
func ResolveSelfDescribedTextSource(input SelfDescribedTextSource) ResolvedSource {
	header := ParseSourceHeader(input.Name, input.Text)
	return ResolvedSource{
		Frontend: header.Using,
		Unit: SourceUnit{
			ID:    input.ID,
			Name:  input.Name,
			Bytes: []byte(MaskSourceHeader(input.Text, header)),
		},
	}
}

func utf16Units(text string) []uint16 {
	return utf16.Encode([]rune(text))
}

func utf16Decode(units []uint16) string {
	return string(utf16.Decode(units))
}

func hasPrefixAt(units []uint16, from int, prefix string) bool {
	if from < 0 {
		from = 0
	}
	if from > len(units) {
		return false
	}
	needle := utf16Units(prefix)
	if from+len(needle) > len(units) {
		return false
	}
	return slices.Equal(units[from:from+len(needle)], needle)
}

func indexFrom(units []uint16, from int, needle string) int {
	if from < 0 {
		from = 0
	}
	target := utf16Units(needle)
	if len(target) == 0 {
		if from > len(units) {
			return len(units)
		}
		return from
	}
	if from > len(units) {
		return -1
	}
	for i := from; i+len(target) <= len(units); i++ {
		if slices.Equal(units[i:i+len(target)], target) {
			return i
		}
	}
	return -1
}

func indexNonSpace(units []uint16) int {
	for i, unit := range units {
		if !jsWhitespace(rune(unit)) {
			return i
		}
	}
	return -1
}

func sliceUnits(units []uint16, start, end int, hasEnd bool) []uint16 {
	length := len(units)
	if start < 0 {
		start += length
		if start < 0 {
			start = 0
		}
	} else if start > length {
		start = length
	}
	if !hasEnd {
		end = length
	} else if end < 0 {
		end += length
		if end < 0 {
			end = 0
		}
	} else if end > length {
		end = length
	}
	if end < start {
		end = start
	}
	return units[start:end]
}

func lineAndColumn(before []uint16) (int, int) {
	line := 1
	start := 0
	for i, unit := range before {
		if unit == '\n' {
			line++
			start = i + 1
		}
	}
	return line, len(utf16.Decode(before[start:])) + 1
}

func matchUsing(raw string) (string, bool) {
	rest, ok := strings.CutPrefix(raw, "<?svml")
	if !ok {
		return "", false
	}
	rest, ok = cutASCIISpace(rest, true)
	if !ok {
		return "", false
	}
	rest, ok = strings.CutPrefix(rest, "using=")
	if !ok {
		return "", false
	}
	if rest == "" || (rest[0] != '\'' && rest[0] != '"') {
		return "", false
	}
	quote := rest[0]
	rest = rest[1:]
	end := 0
	for end < len(rest) {
		r, size := utf8.DecodeRuneInString(rest[end:])
		if r == '\'' || r == '"' || r == '\r' || r == '\n' {
			break
		}
		end += size
	}
	if end == 0 {
		return "", false
	}
	using := rest[:end]
	rest = rest[end:]
	if !strings.HasPrefix(rest, string(quote)) {
		return "", false
	}
	rest = rest[1:]
	rest, _ = cutASCIISpace(rest, false)
	if rest != "?>" {
		return "", false
	}
	return using, true
}

func cutASCIISpace(text string, required bool) (string, bool) {
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	if required && i == 0 {
		return "", false
	}
	return text[i:], true
}

func frontendRequest(using string) bool {
	if using == "" {
		return false
	}
	for _, r := range using {
		if r == '<' || r == '>' || r == '&' || r == '\'' || r == '"' || jsWhitespace(r) {
			return false
		}
	}
	return true
}

func jsWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', '\u2028', '\u2029', '\ufeff':
		return true
	default:
		return unicode.Is(unicode.Zs, r)
	}
}
