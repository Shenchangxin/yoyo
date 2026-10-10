package runtime

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
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

// ImageParts turns image attachments into native multimodal parts so the
// model sees pixels instead of "binary mime (N bytes)".
func ImageParts(workspace string, atts []Attachment) []ContentPart {
	var out []ContentPart
	for i, att := range atts {
		part, ok := imagePart(workspace, att, i)
		if !ok {
			continue
		}
		out = append(out, part)
	}
	return out
}

func imagePart(workspace string, att Attachment, i int) (ContentPart, bool) {
	if !isImageAtt(att) {
		return ContentPart{}, false
	}
	raw, mime, err := attachmentBytes(workspace, att)
	if err != nil || len(raw) == 0 {
		return ContentPart{}, false
	}
	if len(raw) > 2<<20 {
		raw = raw[:2<<20]
	}
	if mime == "" {
		mime = "image/png"
	}
	url := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
	label := att.Name
	if label == "" {
		label = filepath.Base(att.Path)
	}
	if label == "" || label == "." {
		label = fmt.Sprintf("attachment-%d", i+1)
	}
	return ContentPart{Type: "image_url", ImageURL: url, MIME: mime, Text: label}, true
}

func isImageAtt(att Attachment) bool {
	mime := strings.ToLower(strings.TrimSpace(att.MIME))
	if strings.HasPrefix(mime, "image/") {
		return true
	}
	name := strings.ToLower(att.Name + " " + att.Path)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func attachmentBytes(workspace string, att Attachment) ([]byte, string, error) {
	mime := strings.TrimSpace(att.MIME)
	if att.Path != "" {
		p, err := jailPath(workspace, att.Path)
		if err != nil {
			return nil, mime, err
		}
		raw, err := os.ReadFile(p)
		return raw, mime, err
	}
	if att.DataB64 == "" {
		return nil, mime, fmt.Errorf("empty")
	}
	raw, err := base64.StdEncoding.DecodeString(att.DataB64)
	return raw, mime, err
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
			if isImageMIME(mime) || looksImageName(label) {
				return fmt.Sprintf("## attachment:%s\nsaved %s (%d bytes, %s); image attached as a multimodal part\n", label, filepath.ToSlash(rel), len(raw), mime)
			}
			return fmt.Sprintf("## attachment:%s\nsaved %s (%d bytes, %s)\n", label, filepath.ToSlash(rel), len(raw), mime)
		}
	}
	return formatAttachment(label, mime, raw)
}

func formatAttachment(label, mime string, raw []byte) string {
	if isImageMIME(mime) || looksImageName(label) {
		return fmt.Sprintf("## attachment:%s\nimage attached as a multimodal part (%d bytes, %s)\n", label, len(raw), mime)
	}
	if looksBinary(raw) {
		return fmt.Sprintf("## attachment:%s\nbinary %s (%d bytes)\n", label, mime, len(raw))
	}
	capped, trunc := capText(string(raw), 800)
	out := fmt.Sprintf("## attachment:%s\n%s", label, capped)
	if trunc {
		out += "\n…[truncated; use read_file for more]"
	}
	return out + "\n"
}

const maxUploadBytes = 64 << 20

// MaterializeAttachments copies operator-picked files into the workspace jail
// so later read_file / @file mentions succeed. Native pickers often return
// absolute paths outside the project; File-API uploads arrive as data_b64
// with no path at all. Both used to vanish from the transcript and the model.
func MaterializeAttachments(workspace string, atts []Attachment) ([]Attachment, error) {
	if len(atts) == 0 {
		return atts, nil
	}
	out := make([]Attachment, 0, len(atts))
	for i, att := range atts {
		next, err := materializeOne(workspace, att, i)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
	}
	return out, nil
}

// StampFileMentions appends @file:<rel> tokens for non-image attachments so
// the user bubble peels a chip and ExpandMentions can preview the file.
func StampFileMentions(message string, atts []Attachment) string {
	seen := map[string]bool{}
	for _, m := range ParseMentions(message) {
		if m.Kind == "file" {
			seen["@file:"+filepath.ToSlash(m.Ref)] = true
		}
	}
	var extra []string
	for _, att := range atts {
		if isImageAtt(att) {
			continue
		}
		rel := strings.TrimSpace(att.Path)
		if rel == "" {
			continue
		}
		token := "@file:" + filepath.ToSlash(rel)
		if seen[token] {
			continue
		}
		seen[token] = true
		extra = append(extra, token)
	}
	if len(extra) == 0 {
		return message
	}
	head := strings.Join(extra, " ")
	msg := strings.TrimSpace(message)
	if msg == "" {
		return head
	}
	return msg + "\n" + head
}

func materializeOne(workspace string, att Attachment, i int) (Attachment, error) {
	label := att.Name
	if label == "" {
		label = filepath.Base(att.Path)
	}
	if label == "" || label == "." {
		label = fmt.Sprintf("attachment-%d", i+1)
	}
	att.Name = label
	if strings.TrimSpace(workspace) == "" {
		return att, nil
	}
	workspace = capability.CanonicalizeToolPath(workspace)

	if strings.TrimSpace(att.Path) != "" {
		src := capability.CanonicalizeToolPath(att.Path)
		if p, err := jailPath(workspace, src); err == nil {
			rel, err := filepath.Rel(workspace, p)
			if err != nil {
				return Attachment{}, fmt.Errorf("attach %s: %w", label, err)
			}
			att.Path = filepath.ToSlash(rel)
			att.DataB64 = ""
			return att, nil
		}
		if !filepath.IsAbs(src) {
			src = filepath.Join(workspace, src)
		}
		src = filepath.Clean(src)
		info, err := os.Stat(src)
		if err != nil {
			return Attachment{}, fmt.Errorf("attach %s: %w", label, err)
		}
		if info.IsDir() {
			return Attachment{}, fmt.Errorf("attach %s: is a directory", label)
		}
		if info.Size() > maxUploadBytes {
			return Attachment{}, fmt.Errorf("attach %s: larger than 64MB", label)
		}
		raw, err := os.ReadFile(src)
		if err != nil {
			return Attachment{}, fmt.Errorf("attach %s: %w", label, err)
		}
		rel, err := writeUpload(workspace, label, raw)
		if err != nil {
			return Attachment{}, err
		}
		att.Path = rel
		att.DataB64 = ""
		return att, nil
	}

	if att.DataB64 == "" {
		return Attachment{}, fmt.Errorf("attach %s: empty", label)
	}
	raw, err := base64.StdEncoding.DecodeString(att.DataB64)
	if err != nil {
		return Attachment{}, fmt.Errorf("attach %s: invalid base64", label)
	}
	if len(raw) > maxUploadBytes {
		return Attachment{}, fmt.Errorf("attach %s: larger than 64MB", label)
	}
	rel, err := writeUpload(workspace, label, raw)
	if err != nil {
		return Attachment{}, err
	}
	att.Path = rel
	att.DataB64 = ""
	return att, nil
}

func writeUpload(workspace, label string, raw []byte) (string, error) {
	dir := filepath.Join(workspace, ".yoyo", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("attach %s: %w", label, err)
	}
	dest := uniqueUploadPath(dir, label)
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		return "", fmt.Errorf("attach %s: %w", label, err)
	}
	rel, err := filepath.Rel(workspace, dest)
	if err != nil {
		return "", fmt.Errorf("attach %s: %w", label, err)
	}
	return filepath.ToSlash(rel), nil
}

func uniqueUploadPath(dir, name string) string {
	base := sanitizeName(name)
	dest := filepath.Join(dir, base)
	if _, err := os.Stat(dest); err != nil {
		return dest
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		stem = "file"
	}
	for i := 2; i < 10000; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, os.Getpid(), ext))
}

func isImageMIME(mime string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
}

func looksImageName(name string) bool {
	n := strings.ToLower(name)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if strings.HasSuffix(n, ext) {
			return true
		}
	}
	return false
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
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return "file"
	}
	return out
}
