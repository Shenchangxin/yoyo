// 本文件维护可重建的搜索边车 _search.json。JSONL 仍是正文来源。

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// Index 是会话搜索边车。文档按 id 存在，匹配不区分大小写。
type Index struct {
	// mu 保护 docs。
	mu sync.Mutex
	// path 是 dir/_search.json。
	path string
	// docs 的值是已经转成小写的检索文本。
	docs map[string]string
}

// OpenIndex 打开 dir/_search.json。文件能读就解进 docs，解失败则保留空 map。文件不存在也返回空索引。
func OpenIndex(dir string) *Index {
	idx := &Index{path: filepath.Join(dir, "_search.json"), docs: map[string]string{}}
	if b, err := os.ReadFile(idx.path); err == nil {
		_ = json.Unmarshal(b, &idx.docs)
	}
	return idx
}

// Put 把 blob 转成小写后写入该 id。x 为 nil 时直接返回。
func (x *Index) Put(id, blob string) {
	if x == nil {
		return
	}
	x.mu.Lock()
	x.docs[id] = strings.ToLower(blob)
	x.mu.Unlock()
}

// Delete 删掉该 id。x 为 nil 时直接返回。
func (x *Index) Delete(id string) {
	if x == nil {
		return
	}
	x.mu.Lock()
	delete(x.docs, id)
	x.mu.Unlock()
}

// Match 返回命中的会话 id。q 去掉空白后为空时返回全部 id，顺序不定。
// 否则按 tokenize 拆词，每个词都要出现在小写文本里。x 为 nil 时返回 nil。
func (x *Index) Match(q string) []string {
	if x == nil {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
	x.mu.Lock()
	defer x.mu.Unlock()
	if q == "" {
		out := make([]string, 0, len(x.docs))
		for id := range x.docs {
			out = append(out, id)
		}
		return out
	}
	terms := tokenize(q)
	var out []string
	for id, blob := range x.docs {
		ok := true
		for _, t := range terms {
			if !strings.Contains(blob, t) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, id)
		}
	}
	return out
}

// Flush 把 docs 写成 path，权限 0o644。x 为 nil 时返回 nil。
func (x *Index) Flush() error {
	if x == nil {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	b, err := json.Marshal(x.docs)
	if err != nil {
		return err
	}
	return os.WriteFile(x.path, b, 0o644)
}

// tokenize 按字母和数字切分，其他字符当分隔。结果已是小写。
// s 转成小写后逐个字符扫描，字母和数字留下，其他字符切开。
func tokenize(s string) []string {
	var b strings.Builder
	var out []string
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
