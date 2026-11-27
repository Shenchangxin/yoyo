// Package pages 定义笔记的数据形状（Page、Meta、Review）和读写面 Catalog。
// 存储由 learn 实现；本包只留下双方共用的规范化、截断、审阅哈希，以及迁移时读取旧版 Markdown 页面的 ReadFile。
package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultSpace 是空间名为空时使用的目录名，值为 home。这是存储根下的子目录，不是主目录本身。
	DefaultSpace = "home"
	// MaxTitle 是标题的符文上限，值为 160。
	MaxTitle = 160
	// MaxBody 是正文的符文上限，值为 100000。
	MaxBody = 100_000
)

// Page 是一篇 Markdown。Content 不进 YAML 头，写在正文区。
type Page struct {
	ID            string    `json:"id" yaml:"id"`
	SpaceID       string    `json:"space_id" yaml:"space_id"`
	ParentID      string    `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Title         string    `json:"title" yaml:"title"`
	Content       string    `json:"content" yaml:"-"`
	Revision      int       `json:"revision" yaml:"revision"`
	SourceSession string    `json:"source_session,omitempty" yaml:"source_session,omitempty"`
	CreatedAt     time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" yaml:"updated_at"`
}

// Meta 是索引和列表用的摘要，Preview 来自正文。
type Meta struct {
	ID            string    `json:"id"`
	SpaceID       string    `json:"space_id"`
	ParentID      string    `json:"parent_id,omitempty"`
	Title         string    `json:"title"`
	Revision      int       `json:"revision"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
	SourceSession string    `json:"source_session,omitempty"`
	Preview       string    `json:"preview,omitempty"`
}

// ConflictError 表示保存时的期望修订号和磁盘上的不一致。Page 是磁盘上的当前页。
type ConflictError struct {
	Page Page
}

// Error 返回 page revision conflict。
func (e *ConflictError) Error() string { return "page revision conflict" }

// frontmatter 是 Markdown 文件 YAML 头里的字段，不含正文。
type frontmatter struct {
	ID            string    `yaml:"id"`
	SpaceID       string    `yaml:"space_id"`
	ParentID      string    `yaml:"parent_id,omitempty"`
	Title         string    `yaml:"title"`
	Revision      int       `yaml:"revision"`
	SourceSession string    `yaml:"source_session,omitempty"`
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
}

// readPageFile 读取路径并解码成 Page。
// path 交给 os.ReadFile，读到的字节传给 decodePage。
func readPageFile(path string) (Page, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}
	return decodePage(b)
}

// decodePage 要求以 --- 开头的 YAML 头。空间为空时用 DefaultSpace。修订号小于 1 时写成 1。
// raw 转成字符串。不是 --- 加换行开头时返回 missing frontmatter。没有结束标记或 YAML 解析失败时返回错误。
func decodePage(raw []byte) (Page, error) {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return Page{}, fmt.Errorf("missing frontmatter")
	}
	rest := text[3:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Page{}, fmt.Errorf("unterminated frontmatter")
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return Page{}, err
	}
	body := rest[end+4:]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimSuffix(body, "\n")
	body = strings.TrimSuffix(body, "\r")
	if fm.SpaceID == "" {
		fm.SpaceID = DefaultSpace
	}
	if fm.Revision < 1 {
		fm.Revision = 1
	}
	return Page{
		ID: fm.ID, SpaceID: fm.SpaceID, ParentID: fm.ParentID, Title: fm.Title,
		Content: body, Revision: fm.Revision, SourceSession: fm.SourceSession,
		CreatedAt: fm.CreatedAt, UpdatedAt: fm.UpdatedAt,
	}, nil
}

// previewOf 把空白折叠成单行，超过 160 个符文则截断。
// content 折叠空白。rune 数大于 160 时截到 160。
func previewOf(content string) string {
	s := strings.Join(strings.Fields(content), " ")
	n := utf8.RuneCountInString(s)
	if n <= 160 {
		return s
	}
	return string([]rune(s)[:160])
}

// normalizeSpace 清掉非法字符。结果为空时返回 DefaultSpace。
// id 以 allowDot 为 false 传给 sanitizeSegment。结果为空时返回 DefaultSpace。
func normalizeSpace(id string) string {
	id = sanitizeSegment(id, false)
	if id == "" {
		return DefaultSpace
	}
	return id
}

// normalizePageID 清掉非法字符，并删掉剩余的 ..。
// id 以 allowDot 为 true 传给 sanitizeSegment，再删掉剩余的 ..。
func normalizePageID(id string) string {
	id = sanitizeSegment(id, true)
	for strings.Contains(id, "..") {
		id = strings.ReplaceAll(id, "..", "")
	}
	return id
}

// sanitizeSegment 只保留字母、数字、减号和下划线。allowDot 为真时还保留点。
// id 去掉首尾空白后逐个 rune 过滤。
func sanitizeSegment(id string, allowDot bool) string {
	id = strings.TrimSpace(id)
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case allowDot && r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// clipTitle 去掉空白。空标题变成 Untitled。超过 MaxTitle 个符文则截断。
// s 去掉首尾空白。为空时写成 Untitled。rune 数大于 MaxTitle 时截到 MaxTitle。
func clipTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "Untitled"
	}
	if utf8.RuneCountInString(s) > MaxTitle {
		s = string([]rune(s)[:MaxTitle])
	}
	return s
}

// clipBody 在正文超过 MaxBody 个符文时截断。
// s 的 rune 数大于 MaxBody 时截到 MaxBody。
func clipBody(s string) string {
	if utf8.RuneCountInString(s) > MaxBody {
		s = string([]rune(s)[:MaxBody])
	}
	return s
}

// hashDraft 对空间、页面 id、标题、正文和修订号做 SHA-256，返回十六进制。
// title 按 spaceID、pageID、title、content、rev 的顺序用换行拼进待哈希文本。
// content 按同样顺序拼进待哈希文本。
// pageID 按同样顺序拼进待哈希文本。
// spaceID 放在待哈希文本的第一段。
// rev 以十进制写在最后一段。
func hashDraft(title, content, pageID, spaceID string, rev int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%d", spaceID, pageID, title, content, rev)))
	return hex.EncodeToString(sum[:])
}
