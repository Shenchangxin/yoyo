package runtime

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Attachment struct {
	Path    string
	Name    string
	MIME    string
	DataB64 string
}

func ExpandAttachments(workspace string, atts []Attachment, budget int) string {
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	for i, att := range atts {
		if used >= budget {
			b.WriteString("…[attachments truncated]\n")
			break
		}
		chunk := attachmentChunk(workspace, att.Path, att.Name, att.MIME, att.DataB64, i)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[attachments truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String())
}

func attachmentChunk(workspace, path, name, mime, b64 string, i int) string {
	label := name
	if label == "" {
		label = filepath.Base(path)
	}
	if label == "" || label == "." {
		label = fmt.Sprintf("attachment-%d", i+1)
	}
	if path != "" {
		p, err := jailPath(workspace, path)
		if err != nil {
			return fmt.Sprintf("## attachment:%s\nERROR: %s\n", label, err)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Sprintf("## attachment:%s\nERROR: %s\n", label, err)
		}
		return formatAttachment(label, mime, raw)
	}
	if b64 == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Sprintf("## attachment:%s\nERROR: invalid base64\n", label)
	}
	if workspace != "" && looksBinary(raw) {
		dir := filepath.Join(workspace, ".yoyo", "uploads")
		_ = os.MkdirAll(dir, 0o755)
		dest := filepath.Join(dir, sanitizeName(label))
		if err := os.WriteFile(dest, raw, 0o644); err == nil {
			rel, _ := filepath.Rel(workspace, dest)
			return fmt.Sprintf("## attachment:%s\nsaved %s (%d bytes, %s)\n", label, filepath.ToSlash(rel), len(raw), mime)
		}
	}
	return formatAttachment(label, mime, raw)
}

func formatAttachment(label, mime string, raw []byte) string {
	if looksBinary(raw) {
		return fmt.Sprintf("## attachment:%s\nbinary %s (%d bytes)\n", label, mime, len(raw))
	}
	capped, trunc := capText(string(raw), 800)
	out := fmt.Sprintf("## attachment:%s\n%s", label, capped)
	if trunc {
		out += "\n…[truncated]"
	}
	return out + "\n"
}

func looksBinary(b []byte) bool {
	if !utf8.Valid(b) {
		return true
	}
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

func sanitizeName(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	if name == "" {
		return "file"
	}
	return name
}
