// Copyright 2026 Hypit.AI
// Translated from @hypit/source/test/text.test.ts (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package source

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestSourceHeaderExplicitlySelectsOneFrontendAndMaskingPreservesOffsets(t *testing.T) {
	text := "<?svml using=\"@hypit/markup@1\"?>\n\n<svml/>\n"
	header := ParseSourceHeader("main.svml", text)
	if header.Using != "@hypit/markup@1" {
		t.Fatalf("using %q", header.Using)
	}
	masked := MaskSourceHeader(text, header)
	if utf16Len(masked) != utf16Len(text) {
		t.Fatalf("length %d != %d", utf16Len(masked), utf16Len(text))
	}
	if utf16Index(masked, "<svml/>") != utf16Index(text, "<svml/>") {
		t.Fatalf("index %d != %d", utf16Index(masked, "<svml/>"), utf16Index(text, "<svml/>"))
	}
	if strings.TrimSpace(utf16Slice(masked, 0, header.End)) != "" {
		t.Fatalf("prefix %q", utf16Slice(masked, 0, header.End))
	}
}

func TestHeaderMaskingPreservesUTF16OffsetsForANonBMPFrontendRequest(t *testing.T) {
	text := "<?svml using=\"project.𠮷@1\"?>\r\n<body/>\r\n"
	header := ParseSourceHeader("project.svml", text)
	masked := MaskSourceHeader(text, header)
	if utf16Len(masked) != utf16Len(text) {
		t.Fatalf("length %d != %d", utf16Len(masked), utf16Len(text))
	}
	if utf16Slice(masked, header.End, utf16Len(masked)) != "\r\n<body/>\r\n" {
		t.Fatalf("tail %q", utf16Slice(masked, header.End, utf16Len(masked)))
	}
	if utf16Index(masked, "<body/>") != utf16Index(text, "<body/>") {
		t.Fatalf("index %d != %d", utf16Index(masked, "<body/>"), utf16Index(text, "<body/>"))
	}
}

func TestSourceHeaderAcceptsAUTF8BOMButNoImplicitOrDuplicateFrontend(t *testing.T) {
	header := ParseSourceHeader("bom.svs", "\ufeff<?svml using=\"@hypit/recipe@1\"?>\n<sheet/>")
	if header.Using != "@hypit/recipe@1" {
		t.Fatalf("using %q", header.Using)
	}

	missing := recoverHeader(t, func() { ParseSourceHeader("missing.svml", "<svml/>") })
	if missing.Code != "SOURCE_HEADER_MISSING" {
		t.Fatal(missing.Code)
	}

	duplicate := recoverHeader(t, func() {
		ParseSourceHeader("duplicate.svml", "<?svml using=\"a@1\"?>\n<?svml using=\"b@1\"?>")
	})
	if duplicate.Code != "SOURCE_HEADER_DUPLICATE" || !strings.HasPrefix(duplicate.Error(), "duplicate.svml:2:1:") {
		t.Fatalf("%s %s", duplicate.Code, duplicate.Error())
	}

	padded := recoverHeader(t, func() { ParseSourceHeader("padded.svml", "<?svml using=\" a@1\"?>\n<body/>") })
	if padded.Code != "SOURCE_HEADER_FRONTEND" {
		t.Fatal(padded.Code)
	}
}

func TestSourceHeaderErrorStrings(t *testing.T) {
	cases := []struct {
		name, text, code, message string
	}{
		{"missing.svml", "<svml/>", "SOURCE_HEADER_MISSING", "missing.svml:1:1: Source must begin with <?svml using=\"...\"?>"},
		{"empty.svml", "", "SOURCE_HEADER_MISSING", "empty.svml:1:1: Source must begin with <?svml using=\"...\"?>"},
		{"bom.svml", "\ufeff<svml/>", "SOURCE_HEADER_MISSING", "bom.svml:1:2: Source must begin with <?svml using=\"...\"?>"},
		{"open.svml", "<?svml using=\"a@1\"", "SOURCE_HEADER_UNCLOSED", "open.svml:1:1: Source Header is not closed"},
		{"bad.svml", "<?svml using=a@1?>", "SOURCE_HEADER_INVALID", "bad.svml:1:1: Source Header must use exactly <?svml using=\"logical-frontend-request\"?>"},
		{"empty-using.svml", "<?svml using=\"\"?>", "SOURCE_HEADER_INVALID", "empty-using.svml:1:1: Source Header must use exactly <?svml using=\"logical-frontend-request\"?>"},
		{"vt.svml", "<?svml\vusing=\"a@1\"?>", "SOURCE_HEADER_INVALID", "vt.svml:1:1: Source Header must use exactly <?svml using=\"logical-frontend-request\"?>"},
		{"padded.svml", "<?svml using=\" a@1\"?>", "SOURCE_HEADER_FRONTEND", "padded.svml:1:1: Source Header Frontend request must not be empty or padded"},
		{"dup.svml", "<?svml using=\"a@1\"?>\n<?svml using=\"b@1\"?>", "SOURCE_HEADER_DUPLICATE", "dup.svml:2:1: Source declares more than one Source Header"},
		{"crlf.svml", "<?svml using=\"a@1\"?>\r\n<?svml using=\"b@1\"?>", "SOURCE_HEADER_DUPLICATE", "crlf.svml:2:1: Source declares more than one Source Header"},
		{"pad.svml", "<?svml using=\"a@1\"?> \n \t<?svml using=\"b@1\"?>", "SOURCE_HEADER_DUPLICATE", "pad.svml:2:3: Source declares more than one Source Header"},
	}
	for _, tc := range cases {
		err := recoverHeader(t, func() { ParseSourceHeader(tc.name, tc.text) })
		if err.Code != tc.code || err.Error() != tc.message || err.Name != "SourceHeaderError" || err.SourceName != tc.name {
			t.Fatalf("%s: got %s %q name %s source %s", tc.name, err.Code, err.Error(), err.Name, err.SourceName)
		}
	}
	astral := newSourceHeaderError("X", "broken", "f.svml", "𠮷\n", 2)
	if astral.Line != 1 || astral.Column != 2 || astral.Offset != 2 || astral.Error() != "f.svml:1:2: broken" {
		t.Fatalf("%+v", astral)
	}
}

func TestSourceFrontendPackageABI(t *testing.T) {
	if SourceFrontendPackageABI != "hypit.source-frontend@1" {
		t.Fatal(SourceFrontendPackageABI)
	}
}

func TestParseSourceHeaderAcceptsTheGrammarEdges(t *testing.T) {
	single := ParseSourceHeader("sq.svml", "<?svml using='pkg@1'?>\n<body/>")
	if single.Using != "pkg@1" || single.Start != 0 || single.End != 22 {
		t.Fatalf("%+v", single)
	}
	tabbed := ParseSourceHeader("tab.svml", "<?svml\tusing=\"a@1\"  ?>")
	if tabbed.Using != "a@1" {
		t.Fatal(tabbed.Using)
	}
	kept := ParseSourceHeader("body.svml", "<?svml using=\"a@1\"?>\n<svml><?svml using=\"b@1\"?>")
	if kept.Using != "a@1" || kept.End != 20 {
		t.Fatalf("%+v", kept)
	}
	nel := ParseSourceHeader("nel.svml", "<?svml using=\"a\u0085b@1\"?>")
	if nel.Using != "a\u0085b@1" {
		t.Fatalf("%q", nel.Using)
	}
	bom := ParseSourceHeader("bom.svs", "\ufeff<?svml using=\"@hypit/recipe@1\"?>\n<sheet/>")
	if bom.Start != 1 {
		t.Fatal(bom.Start)
	}
}

func TestFrontendWhitespaceMatchesECMAScript(t *testing.T) {
	invalidInside := []rune{'\n', '\r'}
	rejected := []rune{
		'\t', '\v', '\f', ' ', '\u00a0', '\u1680',
		'\u2000', '\u2001', '\u2002', '\u2003', '\u2004', '\u2005', '\u2006', '\u2007', '\u2008', '\u2009', '\u200a',
		'\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff',
		'<', '>', '&',
	}
	for _, r := range invalidInside {
		err := recoverHeader(t, func() {
			ParseSourceHeader("ws.svml", "<?svml using=\"a"+string(r)+"b\"?>")
		})
		if err.Code != "SOURCE_HEADER_INVALID" {
			t.Fatalf("U+%04X %s", r, err.Code)
		}
	}
	for _, r := range rejected {
		err := recoverHeader(t, func() {
			ParseSourceHeader("ws.svml", "<?svml using=\"a"+string(r)+"b\"?>")
		})
		if err.Code != "SOURCE_HEADER_FRONTEND" {
			t.Fatalf("U+%04X %s %s", r, err.Code, err.Error())
		}
	}
}

func TestMaskSourceHeaderReplacesCodeUnitsAndKeepsBreaks(t *testing.T) {
	if got := MaskSourceHeader("ab\ncd", SourceHeader{End: 5}); got != "  \n  " {
		t.Fatalf("%q", got)
	}
	if got := MaskSourceHeader("𠮷", SourceHeader{End: 2}); got != "  " {
		t.Fatalf("%q (%d bytes)", got, len(got))
	}
	text := "<?svml using=\"a@1\"?>\u00a0𠮷"
	header := ParseSourceHeader("astral.svml", text)
	masked := MaskSourceHeader(text, header)
	if utf16Len(masked) != utf16Len(text) || utf16Slice(masked, header.End, utf16Len(masked)) != "\u00a0𠮷" {
		t.Fatalf("%q end %d", masked, header.End)
	}
}

func TestDecodeSourceTextStripsOneBOMAndRejectsInvalidUTF8(t *testing.T) {
	if got := DecodeSourceText(SourceUnit{Bytes: []byte{0xEF, 0xBB, 0xBF, 'a'}}); got != "a" {
		t.Fatalf("%q", got)
	}
	if got := DecodeSourceText(SourceUnit{Bytes: []byte("a\uFEFF")}); got != "a\uFEFF" {
		t.Fatalf("%q", got)
	}
	if got := DecodeSourceText(SourceUnit{}); got != "" {
		t.Fatalf("%q", got)
	}
	for _, bytes := range [][]byte{{0xFF}, {0xEF, 0xBB, 0xBF, 0xFF}, {0xED, 0xA0, 0x80}} {
		message := recoverText(t, func() { DecodeSourceText(SourceUnit{Bytes: bytes}) })
		if message != "The encoded data was not valid for encoding utf-8" {
			t.Fatalf("%v: %s", bytes, message)
		}
	}
}

func TestResolveSelfDescribedTextSourceEncodesTheMaskedHeader(t *testing.T) {
	text := "<?svml using=\"project.𠮷@1\"?>\r\n<body/>\r\n"
	resolved := ResolveSelfDescribedTextSource(SelfDescribedTextSource{ID: "main", Name: "project.svml", Text: text})
	if resolved.Frontend != "project.𠮷@1" || resolved.Unit.ID != "main" || resolved.Unit.Name != "project.svml" {
		t.Fatalf("%+v", resolved)
	}
	masked := MaskSourceHeader(text, ParseSourceHeader("project.svml", text))
	if string(resolved.Unit.Bytes) != masked {
		t.Fatalf("bytes %q want %q", resolved.Unit.Bytes, masked)
	}
	if DecodeSourceText(resolved.Unit) != masked {
		t.Fatal(DecodeSourceText(resolved.Unit))
	}
	recoverHeader(t, func() {
		ResolveSelfDescribedTextSource(SelfDescribedTextSource{Name: "missing.svml", Text: "<svml/>"})
	})
}

func recoverHeader(t *testing.T, fn func()) *SourceHeaderError {
	t.Helper()
	var got *SourceHeaderError
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(*SourceHeaderError)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			got = err
		}()
		fn()
	}()
	if got == nil {
		t.Fatal("expected SourceHeaderError")
	}
	return got
}

func recoverText(t *testing.T, fn func()) string {
	t.Helper()
	var message string
	func() {
		defer func() {
			rec := recover()
			err, ok := rec.(error)
			if !ok {
				t.Fatalf("panic %#v", rec)
			}
			message = err.Error()
		}()
		fn()
	}()
	if message == "" {
		t.Fatal("expected a thrown error")
	}
	return message
}

func utf16Len(text string) int {
	return len(utf16.Encode([]rune(text)))
}

func utf16Index(text, needle string) int {
	units := utf16.Encode([]rune(text))
	target := utf16.Encode([]rune(needle))
	for i := 0; i+len(target) <= len(units); i++ {
		if slices.Equal(units[i:i+len(target)], target) {
			return i
		}
	}
	return -1
}

func utf16Slice(text string, start, end int) string {
	units := utf16.Encode([]rune(text))
	return string(utf16.Decode(units[start:end]))
}
