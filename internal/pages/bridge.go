// 把笔记库的规范化、截断和审阅哈希交给 learn。文件库自己的读写仍在 pages.go。

package pages

// Catalog 是笔记的读写面。文件库和 learn 都实现它，桌面应用只依赖这一面。
type Catalog interface {
	List(spaceID string) []Meta
	Search(q, spaceID string) []Meta
	Get(spaceID, id string) (Page, error)
	Delete(spaceID, id string) error
	Create(p Page) (Page, error)
	Save(p Page, expectedRevision int) (Page, error)
	Propose(r Review) (Review, error)
	Decide(id, hash string, approve bool) (Page, Review, error)
	Pending() []Review
}

// NormalizeSpace 清掉非法字符。结果为空时返回 DefaultSpace。
// id 传给 normalizeSpace。
func NormalizeSpace(id string) string { return normalizeSpace(id) }

// NormalizePageID 清掉非法字符，并删掉剩余的 ..。
// id 传给 normalizePageID。
func NormalizePageID(id string) string { return normalizePageID(id) }

// ClipTitle 把标题截到 MaxTitle 个符文。
// s 传给 clipTitle。
func ClipTitle(s string) string { return clipTitle(s) }

// ClipBody 把正文截到 MaxBody 个符文。
// s 传给 clipBody。
func ClipBody(s string) string { return clipBody(s) }

// Preview 把正文折成最多 160 个符文的单行。
// content 传给 previewOf。
func Preview(content string) string { return previewOf(content) }

// HashDraft 对空间、页面 id、标题、正文和修订号做 SHA-256。learn 的批准卡片用同一公式。
// title、content、pageID、spaceID、rev 按这个顺序传给 hashDraft。
func HashDraft(title, content, pageID, spaceID string, rev int) string {
	return hashDraft(title, content, pageID, spaceID, rev)
}

// ReviewID 用哈希前 12 位生成 rev- 前缀。
// hash 传给 reviewID。
func ReviewID(hash string) string { return reviewID(hash) }

// ReadFile 读取一篇已经写在磁盘上的 Markdown。迁移时用它把旧页面抄进学习库。
// path 传给 readPageFile。
func ReadFile(path string) (Page, error) { return readPageFile(path) }
