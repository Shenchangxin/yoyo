// 把操作者附上的文件收进工作区，并展开成提示文本或多模态图片段。
// 路径是否在工作区内交给 mentions.go 的 jailPath，不调用 WorkspaceTools.resolve。工作区外的文件复制到 .yoyo/uploads。

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

// Attachment 是一条操作者附件。Path 和 DataB64 二选一；物化之后只留工作区内的相对路径。
type Attachment struct {
	Path    string
	Name    string
	MIME    string
	DataB64 string
}

// controlAttachment 识别名为 __request_id__ 的控制附件，它不是文件。
// att 的 Name 去掉空白后等于 __request_id__ 时返回 true。
func controlAttachment(att Attachment) bool {
	return strings.TrimSpace(att.Name) == "__request_id__"
}

// ExpandAttachments 把附件收成提示文本。budget 小于等于 0 时用 2400 字节。超出预算时停下并写上截断标记。控制附件跳过。
// workspace 传给 attachmentChunk。
// atts 逐条处理，控制附件跳过。
func ExpandAttachments(workspace string, atts []Attachment, budget int) string {
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	for i, att := range atts {
		if controlAttachment(att) {
			continue
		}
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

// ImageParts 把图片附件收成原生多模态段，让模型看到像素，而不是 “binary mime (N bytes)”。
// workspace 传给 imagePart。
// atts 逐条处理，控制附件跳过。
func ImageParts(workspace string, atts []Attachment) []ContentPart {
	var out []ContentPart
	for i, att := range atts {
		if controlAttachment(att) {
			continue
		}
		part, ok := imagePart(workspace, att, i)
		if !ok {
			continue
		}
		out = append(out, part)
	}
	return out
}

// imagePart 把一张图收成 data URL。超过 2 MiB 的字节被截断。MIME 为空时写成 image/png。
// workspace 传给 attachmentBytes。
// att 先交给 isImageAtt，通过后连同 workspace 交给 attachmentBytes，再用 Name 和 Path 做标签。
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

// isImageAtt 在 MIME 以 image/ 开头，或名字以 png、jpg、jpeg、gif、webp 结尾时返回真。
// att 读取 MIME、Name 和 Path。
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

// attachmentBytes 有路径时经 jailPath 读取，否则解码 DataB64。
// workspace 在 att.Path 非空时传给 jailPath。
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

// attachmentChunk 把一条附件收成一段文本。路径走 jailPath。无路径的二进制在有工作区时写到 .yoyo/uploads。图片只留多模态说明，不倾倒字节。
// workspace 在 path 非空时传给 jailPath；path 为空且字节像二进制时作为 .yoyo/uploads 的根。
// path 非空时经 jailPath 读文件再交给 formatAttachment，为空则改走 b64。
// name 作为标签，为空时改用 path 的文件名。
// mime 传给 formatAttachment，保存文件时也写入说明。
// b64 在 path 为空时解码，空串则返回空串。
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

// formatAttachment 格式化一段附件。文本超过 800 字符时截断并提示改用 read_file。
// label 写入标题，并传给 looksImageName。
// mime 传给 isImageMIME，图片或二进制说明里也会写上。
// raw 交给 looksBinary；文本时截到 800 字符。
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

// maxUploadBytes 是单份附件的上限，64 MiB。
const maxUploadBytes = 64 << 20

// MaterializeAttachments 把操作者选中的文件复制进工作区，使之后的 read_file 和 @file 能成功。系统选择器常返回项目外的绝对路径；文件接口的上传只有 data_b64，没有路径。这两种以前都会从记录和模型视野里消失。
// workspace 传给 materializeOne。
// atts 长度为 0 时原样返回，否则逐条处理并跳过控制附件。
func MaterializeAttachments(workspace string, atts []Attachment) ([]Attachment, error) {
	if len(atts) == 0 {
		return atts, nil
	}
	out := make([]Attachment, 0, len(atts))
	for i, att := range atts {
		if controlAttachment(att) {
			continue
		}
		next, err := materializeOne(workspace, att, i)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
	}
	return out, nil
}

// StampFileMentions 为非图片附件追加 @file:<相对路径>，让用户气泡剥出芯片，并让 ExpandMentions 能预览文件。已有的 @file 和图片不重复添加。
// message 先交给 ParseMentions 收集已有的 @file；没有新标记时原样返回，去掉空白后为空则只返回新标记。
// atts 逐条取 Path，跳过控制附件、图片和去掉空白后为空的路径。
func StampFileMentions(message string, atts []Attachment) string {
	seen := map[string]bool{}
	for _, m := range ParseMentions(message) {
		if m.Kind == "file" {
			seen["@file:"+filepath.ToSlash(m.Ref)] = true
		}
	}
	var extra []string
	for _, att := range atts {
		if controlAttachment(att) || isImageAtt(att) {
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

// materializeOne 处理一条附件。已在 jailPath 内的路径改成相对路径并清掉 DataB64。工作区外的普通文件复制到 .yoyo/uploads。目录拒绝。超过 maxUploadBytes 拒绝。工作区为空时原样返回。
// workspace 去掉空白后为空时原样返回 att，否则规范化后作为 jailPath 和 writeUpload 的根。
// att 用 Name 和 Path 做标签；Path 去掉空白后非空时尝试收成工作区内相对路径，否则解码 DataB64。
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

// writeUpload 把字节写到 .yoyo/uploads，并返回相对工作区的斜杠路径。
// workspace 与 .yoyo/uploads 拼成目录，并用来计算相对路径。
// label 传给 uniqueUploadPath，也写进错误信息。
// raw 写入目标文件。
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

// uniqueUploadPath 在同名文件已存在时追加 -2、-3，直到 9999。仍冲突则用进程号。
// dir 与净化后的名字拼成路径。
// name 先交给 sanitizeName。
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

// isImageMIME 判断 MIME 是否以 image/ 开头。
// mime 去掉空白并转小写后看是否以 image/ 开头。
func isImageMIME(mime string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
}

// looksImageName 按扩展名判断是不是 png、jpg、jpeg、gif 或 webp。
// name 转小写后看是否以 .png、.jpg、.jpeg、.gif 或 .webp 结尾。
func looksImageName(name string) bool {
	n := strings.ToLower(name)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if strings.HasSuffix(n, ext) {
			return true
		}
	}
	return false
}

// looksBinary 在字节不是合法 UTF-8 或含 NUL 时返回真。
// b 不是合法 UTF-8 或含 0 字节时返回 true。
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

// sanitizeName 只保留字母、数字、点、减号和下划线，其余换成下划线。结果为空时用 file。
// name 先取文件名，再按上述规则过滤。
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
