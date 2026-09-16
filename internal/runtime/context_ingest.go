package runtime

import "strings"

const ingestPreviewRunes = 2_000

// ingestToolResult spills full bytes first, then returns a head+tail preview.
func ingestToolResult(spill *Spill, id, name, content string) string {
	if strings.TrimSpace(content) == "" {
		return emptyToolAnchor(name)
	}
	if spill != nil {
		id = spill.Put(id, content)
	}
	capped, trunc := capText(content, ingestPreviewRunes)
	if !trunc {
		return content
	}
	return stubTool(id, name, len(content)) + "\n" + capped
}
