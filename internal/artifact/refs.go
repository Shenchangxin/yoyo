// 指向哈希的名字。不存放对象字节。检出和演化通过这些名字换快照。

package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// RefActive 是当前快照的指针名，值为 active。
	RefActive = "active"
	// RefCanary 是金丝雀快照的指针名，值为 canary。
	RefCanary = "canary"
	// RefHead 是指针名，值为 HEAD。
	RefHead = "HEAD"
	// RefStaging 是暂存快照的指针名，值为 staging。
	RefStaging = "staging"
)

// Refs 是一组指向哈希的名字。Root 下每个相对路径一个文件。Set 和 Get 在本进程内串行；List 不加锁。
type Refs struct {
	Root string
	mu   sync.Mutex
}

// NewRefs 只记下根目录，不创建它。
func NewRefs(root string) *Refs {
	return &Refs{Root: root}
}

// path 去掉名字开头的斜杠，再按系统分隔符拼到根目录下。
func (r *Refs) path(name string) string {
	name = strings.TrimPrefix(name, "/")
	return filepath.Join(r.Root, filepath.FromSlash(name))
}

// Set 先把哈希写进同路径的临时文件，再改名换成正式指针。哈希会去掉首尾空白，并补一个换行。
func (r *Refs) Set(name, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.path(name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.TrimSpace(hash)+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Get 读出指针里的哈希并去掉首尾空白。文件不存在时返回错误。
func (r *Refs) Get(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, err := os.ReadFile(r.path(name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// GetOrEmpty 在读失败时返回空串。文件里只有空白时也是空串。
func (r *Refs) GetOrEmpty(name string) string {
	h, err := r.Get(name)
	if err != nil {
		return ""
	}
	return h
}

// List 递归走过根目录。目录本身不记入结果，以 .tmp 结尾的文件也跳过。键用斜杠。这次遍历不加锁；根目录不存在时返回错误。
func (r *Refs) List() (map[string]string, error) {
	out := map[string]string{}
	err := filepath.Walk(r.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}
		rel, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = strings.TrimSpace(string(b))
		return nil
	})
	return out, err
}

// Archive 把哈希写到 archive/ 加上哈希前 12 位。空哈希报错。不足或刚好 12 位时，文件名用整段哈希。
func (r *Refs) Archive(hash string) error {
	if hash == "" {
		return fmt.Errorf("refs: empty archive hash")
	}
	short := hash
	if len(short) > 12 {
		short = short[:12]
	}
	return r.Set("archive/"+short, hash)
}

// ModelActive 返回 models/<指纹>/active。指纹里的 /、\、: 会换成下划线。
func ModelActive(fingerprint string) string {
	return "models/" + sanitize(fingerprint) + "/active"
}

// ModelCanary 返回 models/<指纹>/canary。清洗规则与 ModelActive 相同。
func ModelCanary(fingerprint string) string {
	return "models/" + sanitize(fingerprint) + "/canary"
}

// sanitize 把 /、\、: 换成下划线，避免指纹再拆出目录。
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
