// 对象库的哈希、写入和读取。不拥有指针名字，也不定义载荷字段。快照、内容差异和包外的写入方调用 Store。

package artifact

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeebo/blake3"
)

// Store 是内容寻址对象库，按哈希把字节放在 Root 下。
type Store struct {
	Root string
}

// NewStore 只记下根目录，不创建它。
func NewStore(root string) *Store {
	return &Store{Root: root}
}

// HashBytes 返回裸字节的 BLAKE3-256 十六进制。Put 和 PutRaw 用它，PutFile 用同一个算法的流式哈希。
func HashBytes(b []byte) string {
	sum := blake3.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// pathFor 用哈希前两位做子目录。短于 4 个字符时直接放在根下。
func (s *Store) pathFor(hash string) string {
	if len(hash) < 4 {
		return filepath.Join(s.Root, hash)
	}
	return filepath.Join(s.Root, hash[:2], hash)
}

// Path 返回对象在盘上的路径，并先去掉哈希两端的空白。文件可以还不存在。
func (s *Store) Path(hash string) string {
	return s.pathFor(strings.TrimSpace(hash))
}

// Put 哈希的是带 Kind、ID、CreatedAt 和载荷的信封。CreatedAt 取这次写入的 UTC，所以同一段文案写两次是两个对象。同哈希的路径会整份重写。
func (s *Store) Put(kind Kind, id string, payload any) (string, error) {
	env := Envelope{
		Kind:      kind,
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Payload:   payload,
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	h := HashBytes(raw)
	p := s.pathFor(h)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		return "", err
	}
	return h, nil
}

// PutRaw 哈希裸字节。目标文件已在就直接返回，不重写。
func (s *Store) PutRaw(b []byte) (string, error) {
	h := HashBytes(b)
	p := s.pathFor(h)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		return h, nil
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", err
	}
	return h, nil
}

// PutFile 哈希文件的裸字节并拷进库。目标已在就直接返回，不重写；独占创建若撞上已有文件，同样返回该哈希。拷贝或关闭失败会删掉未写完的目标。
func (s *Store) PutFile(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	sum := h.Sum(nil)
	hash := hex.EncodeToString(sum)
	dst := s.pathFor(hash)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(dst); err == nil {
		return hash, nil
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return hash, nil
		}
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	return hash, nil
}

// GetRaw 读出该哈希的原始字节。读失败时错误里带上哈希。两端空白会先去掉。
func (s *Store) GetRaw(hash string) ([]byte, error) {
	hash = strings.TrimSpace(hash)
	p := s.pathFor(hash)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("cas: missing %s: %w", hash, err)
	}
	return b, nil
}

// GetEnvelope 读出原始字节并按信封解码。解码失败时仍把原始字节一并返回。
func (s *Store) GetEnvelope(hash string) (Envelope, []byte, error) {
	b, err := s.GetRaw(hash)
	if err != nil {
		return Envelope{}, nil, err
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return Envelope{}, b, err
	}
	return env, b, nil
}

// Decode 从信封的 payload 解出 T。读取或解码失败时返回 T 的零值。
func Decode[T any](s *Store, hash string) (T, Envelope, error) {
	var zero T
	env, raw, err := s.GetEnvelope(hash)
	if err != nil {
		return zero, env, err
	}
	var wrap struct {
		Payload T `json:"payload"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return zero, env, err
	}
	return wrap.Payload, env, nil
}

// Exists 只看这个哈希的路径在不在，不校验内容。两端空白会先去掉。
func (s *Store) Exists(hash string) bool {
	_, err := os.Stat(s.pathFor(strings.TrimSpace(hash)))
	return err == nil
}
