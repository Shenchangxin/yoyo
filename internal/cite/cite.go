// Package cite 给报告附上带来源摘录和哈希的引用，并检查报告是否提到这些来源。
// HashExcerpt 取 SHA-256 的前 8 个字节。Validate 要求来源非空、URL 以前缀 http 开头，并且带哈希。
package cite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Source 是一条引用：地址、摘录和哈希。
type Source struct {
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
	Hash    string `json:"hash"`
}

// HashExcerpt 把 URL、一个换行和摘录做 SHA-256，十六进制只编码摘要的前 8 个字节。
// url 与一个换行、excerpt 拼成字节交给 sha256.Sum256，并写入返回值的 URL。
// excerpt 接在 url 和一个换行后面交给 sha256.Sum256，并写入返回值的 Excerpt。
func HashExcerpt(url, excerpt string) Source {
	sum := sha256.Sum256([]byte(url + "\n" + excerpt))
	return Source{URL: url, Excerpt: excerpt, Hash: hex.EncodeToString(sum[:8])}
}

// Write 把来源写成缩进两个空格的 JSON，文件权限 0o644。
// path 交给 os.WriteFile，权限是 0o644。
// srcs 交给 json.MarshalIndent，前缀是空串，缩进是两个空格。
func Write(path string, srcs []Source) error {
	b, err := json.MarshalIndent(srcs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Validate 读 citesPath 的 JSON 并对照 report。
// 来源为空、URL 为空或前缀不是 http、哈希为空时返回错误。
// 摘录非空时，报告转成小写后要包含摘录前 40 个字符的小写，或者包含原文 URL。摘录为空则不做这步。
func Validate(report, citesPath string) error {
	b, err := os.ReadFile(citesPath)
	if err != nil {
		return err
	}
	var srcs []Source
	if err := json.Unmarshal(b, &srcs); err != nil {
		return err
	}
	if len(srcs) == 0 {
		return fmt.Errorf("cite: empty sources")
	}
	low := strings.ToLower(report)
	for _, s := range srcs {
		if s.URL == "" || !strings.HasPrefix(s.URL, "http") {
			return fmt.Errorf("cite: bad url")
		}
		if s.Hash == "" {
			return fmt.Errorf("cite: missing hash for %s", s.URL)
		}
		if s.Excerpt != "" && !strings.Contains(low, strings.ToLower(s.Excerpt[:min(40, len(s.Excerpt))])) && !strings.Contains(low, s.URL) {
			return fmt.Errorf("cite: report does not mention %s", s.URL)
		}
	}
	return nil
}

// min 返回 a 和 b 里较小的那个。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
