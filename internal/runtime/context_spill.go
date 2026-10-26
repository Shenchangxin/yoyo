// 工具结果的无损落盘。压缩和投影只留预览，全文在磁盘上。
// recall_context 经 Get 读回后按页返回：offset 和 limit 是从 1 起的行，小于等于 0 时 offset 当作 1、limit 当作 200，取出的页再按 4000 个 rune 做头尾截断。分页实现在 tools.go 的 pageSpill。

package runtime

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zeebo/blake3"
)

// Spill 保存被截掉的工具结果全文。
type Spill struct {
	// Dir 是主目录。BindSpill 把它指到 YOYO_HOME 下的会话 spill。
	Dir string
	// Mirror 非空时再写一份到工作区监狱，供 grep 和 read_file 按同一 id 查看。
	Mirror string
	mu     sync.Mutex
}

// NewSpill 用 dir 做主目录。dir 为空时返回 nil。
func NewSpill(dir string) *Spill {
	if dir == "" {
		return nil
	}
	return &Spill{Dir: dir}
}

// SessionSpillDir 返回 homeRoot/sessions/<sessionID>/spill。任一为空则返回空串。
func SessionSpillDir(homeRoot, sessionID string) string {
	if homeRoot == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(homeRoot, "sessions", sessionID, "spill")
}

// WorkspaceSpillDir 返回工作区 .yoyo/context/<sessionID>/spill。任一为空则返回空串。
// workspace 为空时返回空串，否则与 sessionID 拼成该目录。
func WorkspaceSpillDir(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "context", sessionID, "spill")
}

// BindSpill 把 Dir 指到 YOYO_HOME 下的会话 spill，Mirror 指到工作区监狱。sessionID 为空返回 nil。
// 没有 home 目录时 Dir 改用工作区路径，并且不再镜像。两边解析成同一路径时也不镜像。
// homeRoot 传给 SessionSpillDir。
// workspace 传给 WorkspaceSpillDir。
func BindSpill(homeRoot, workspace, sessionID string) *Spill {
	if sessionID == "" {
		return nil
	}
	dir := SessionSpillDir(homeRoot, sessionID)
	ws := WorkspaceSpillDir(workspace, sessionID)
	if dir == "" {
		dir = ws
		ws = ""
	}
	if dir == "" {
		return nil
	}
	if ws != "" && samePath(dir, ws) {
		ws = ""
	}
	return &Spill{Dir: dir, Mirror: ws}
}

// samePath 用绝对路径比较。解析失败时退回字符串相等。任一侧为空则返回 false。
// b 为空时返回 false，否则与另一侧比绝对路径，解析失败时退回字符串相等。
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, err := filepath.Abs(a)
	if err != nil {
		return a == b
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return a == b
	}
	return aa == bb
}

// Put 把正文写成 id.txt，并在 Mirror 非空时再写一份。id 为空时用正文 blake3 的前 8 字节十六进制。返回清洗后的 id。接收者为 nil、目录为空或正文为空时原样返回 id。
// content 为空时原样返回 id；id 为空时参与 blake3，并作为文件正文写入。
func (s *Spill) Put(id, content string) string {
	if s == nil || s.Dir == "" || content == "" {
		return id
	}
	if id == "" {
		sum := blake3.Sum256([]byte(content))
		id = hex.EncodeToString(sum[:8])
	}
	id = sanitizeID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	writeSpillFile(s.Dir, id, content)
	if s.Mirror != "" {
		writeSpillFile(s.Mirror, id, content)
	}
	return id
}

// writeSpillFile 在 dir 下写入 id.txt。目录创建和写入的错误都忽略。
// content 作为文件正文写入。
func writeSpillFile(dir, id, content string) {
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, id+".txt"), []byte(content), 0o644)
}

// Get 先读 Dir，没有再读 Mirror。接收者或目录为空时返回 os.ErrNotExist。
// id 交给 sanitizeID，再拼成 .txt 文件名。
func (s *Spill) Get(id string) (string, error) {
	if s == nil || s.Dir == "" {
		return "", os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := sanitizeID(id) + ".txt"
	b, err := os.ReadFile(filepath.Join(s.Dir, name))
	if err == nil {
		return string(b), nil
	}
	if s.Mirror != "" {
		if b, err2 := os.ReadFile(filepath.Join(s.Mirror, name)); err2 == nil {
			return string(b), nil
		}
	}
	return "", err
}

// Size 返回 Dir 或 Mirror 里该 id 文件的字节数。都没有则返回 0。
func (s *Spill) Size(id string) int {
	if s == nil || s.Dir == "" {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := sanitizeID(id) + ".txt"
	for _, dir := range []string{s.Dir, s.Mirror} {
		if dir == "" {
			continue
		}
		fi, err := os.Stat(filepath.Join(dir, name))
		if err == nil {
			return int(fi.Size())
		}
	}
	return 0
}

// ListIDs 合并 Dir 和 Mirror 里的 .txt 文件名，去掉后缀并去重。不排序。
func (s *Spill) ListIDs() []string {
	if s == nil || s.Dir == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	var ids []string
	collect := func(dir string) {
		if dir == "" {
			return
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".txt") {
				continue
			}
			id := strings.TrimSuffix(name, ".txt")
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	collect(s.Dir)
	collect(s.Mirror)
	return ids
}

// sanitizeID 只保留字母、数字、连字符和下划线。结果为空时写成 x，长于 64 则截断。
// id 去掉空白后再按上述规则过滤。
func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	var b strings.Builder
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	if len(out) > 64 {
		return out[:64]
	}
	return out
}
