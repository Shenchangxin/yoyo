package runtime

import (
	"path/filepath"
	"regexp"
	"strings"
)

// htmlCloseTag matches the document's closing tag in any letter case.
var htmlCloseTag = regexp.MustCompile(`(?i)</html\s*>`)

// isHTMLDocPath reports whether rel is an HTML document.
func isHTMLDocPath(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".html", ".htm":
		return true
	default:
		return false
	}
}

// htmlTailAfterClose returns the non-space text after the last </html>, or ""
// when there is none. Text there sits outside the document: browsers render
// it after the page, and the next part appended to the file lands after it.
func htmlTailAfterClose(doc string) string {
	locs := htmlCloseTag.FindAllStringIndex(doc, -1)
	if len(locs) == 0 {
		return ""
	}
	return strings.TrimSpace(doc[locs[len(locs)-1][1]:])
}

// htmlTailSnippet shows the start of a stray tail on one line for a message.
func htmlTailSnippet(tail string) string {
	r := []rune(strings.Join(strings.Fields(tail), " "))
	if len(r) > 80 {
		return string(r[:80]) + "..."
	}
	return string(r)
}

// Chunked writes that rebuild an HTML page have repeatedly ended the document
// early: a part that starts with </script></body></html> leaves the next part
// as text after the page. That silently breaks the page, and the agent's own
// later checks passed on the wreck. Appends are refused when they would leave
// text after </html>; overwrites are allowed and only warned about.
func htmlAppendRefusal(rel, want string) string {
	if !isHTMLDocPath(rel) {
		return ""
	}
	tail := htmlTailAfterClose(want)
	if tail == "" {
		return ""
	}
	return "refusing to append to " + rel + ": the result would put text after </html> (\"" + htmlTailSnippet(tail) + "\"), which is outside the document. " +
		"Put the new code before the closing </script> with str_replace, or rewrite the whole file with write_file mode=overwrite."
}

// htmlOverwriteWarning returns a note for a written HTML file whose content
// has text after </html>, or "" when the document is closed cleanly.
func htmlOverwriteWarning(rel, want string) string {
	if !isHTMLDocPath(rel) {
		return ""
	}
	tail := htmlTailAfterClose(want)
	if tail == "" {
		return ""
	}
	return "warning: " + rel + " has text after </html> (\"" + htmlTailSnippet(tail) + "\"). Browsers render it outside the page; move it inside the document."
}
