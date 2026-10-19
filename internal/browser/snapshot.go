package browser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	SnapshotMinimal     = "minimal"
	SnapshotInteractive = "interactive"
	SnapshotFull        = "full"
	SnapshotStructured  = "structured"
)

// NormalizeSnapshotLevel picks a DOM tier. Empty level defaults to
// interactive for remote http(s) and full for workspace HTML, where
// browser_verify still needs page text.
func NormalizeSnapshotLevel(level string, pageURL string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case SnapshotMinimal, SnapshotInteractive, SnapshotFull, SnapshotStructured:
		return strings.ToLower(strings.TrimSpace(level))
	}
	if isRemoteHTTP(pageURL) {
		return SnapshotInteractive
	}
	return SnapshotFull
}

func isRemoteHTTP(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host != "127.0.0.1" && host != "localhost" && host != "::1"
}

// FormatSnapshot renders a Snapshot at one of the four observation levels.
func FormatSnapshot(snap Snapshot, level string, live bool) string {
	level = NormalizeSnapshotLevel(level, snap.URL)
	acts := snap.Elements
	if len(acts) == 0 && strings.TrimSpace(snap.Actions) != "" {
		// Actions were formatted already; keep them for interactive/full.
	}
	var b strings.Builder
	if !live {
		b.WriteString("(isolated Chrome is not running; this is the last snapshot, not the live page)\n")
	}
	switch level {
	case SnapshotMinimal:
		fmt.Fprintf(&b, "Page: %s | %q\nInteractive elements: %d", snap.URL, snap.Title, countActions(snap))
	case SnapshotInteractive:
		fmt.Fprintf(&b, "Page: %s | %q\n", snap.URL, snap.Title)
		if strings.TrimSpace(snap.Actions) != "" {
			b.WriteString("visible actions (prefer ref= on browser_click):\n")
			b.WriteString(snap.Actions)
		} else {
			b.WriteString("no visible interactive elements on this page")
		}
	case SnapshotStructured:
		b.WriteString(formatStructuredSnapshot(snap))
	default:
		body := snap.Text
		if strings.TrimSpace(snap.Actions) != "" {
			if strings.TrimSpace(body) != "" {
				body += "\n\n"
			}
			body += "visible actions (prefer ref= on browser_click):\n" + snap.Actions
		}
		b.WriteString(body)
	}
	if strings.TrimSpace(snap.Errors) != "" {
		b.WriteString("\n\n")
		b.WriteString(snap.Errors)
	}
	if snap.Profile != "" {
		fmt.Fprintf(&b, "\nprofile=%s", snap.Profile)
	}
	return strings.TrimSpace(b.String())
}

func countActions(snap Snapshot) int {
	if n := len(snap.Elements); n > 0 {
		return n
	}
	if strings.TrimSpace(snap.Actions) == "" {
		return 0
	}
	n := 0
	for _, line := range strings.Split(snap.Actions, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

func formatStructuredSnapshot(snap Snapshot) string {
	els := snap.Elements
	if len(els) == 0 {
		els = nil
	}
	payload := map[string]any{
		"url":      snap.URL,
		"title":    snap.Title,
		"elements": els,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return snap.Actions
	}
	return string(raw)
}

func (h *Host) snapshotWithElements() Snapshot {
	snap := h.snapshotLocked()
	snap.Elements = h.refs.all()
	if name := h.ProfileName(); name != "" {
		snap.Profile = name
	}
	return snap
}
