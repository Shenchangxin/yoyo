package runtime

import "strings"

const ingestPreviewRunes = 2_000

// ingestToolResult spills full bytes first, then returns a head+tail preview
// plus the spill id so a trace viewer can recall the lossless artifact.
func ingestToolResult(spill *Spill, id, name, content string) (preview, spillID string) {
	spillID = id
	if strings.TrimSpace(content) == "" {
		return emptyToolAnchor(name), spillID
	}
	if spill != nil {
		spillID = spill.Put(id, content)
	}
	capped, trunc := capText(content, ingestPreviewRunes)
	if !trunc {
		return content, spillID
	}
	return stubTool(spillID, name, len(content)) + "\n" + capped, spillID
}
