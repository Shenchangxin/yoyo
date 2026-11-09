// ring.go 拥有 agent stderr 的环形缓冲：只留最后 64 KB，取出时先打码。
// client.go 把它接到进程的 stderr；buddy 的日志尾巴和 acpworker 的崩溃原因读它。

package acp

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// Ring 是一个只留最后 size 字节的写入端。
type Ring struct {
	mu   sync.Mutex
	size int
	buf  []byte
}

// NewRing 返回容量为 size 的缓冲。size 小于 1 时当作 1。
func NewRing(size int) *Ring {
	if size < 1 {
		size = 1
	}
	return &Ring{size: size}
}

// Write 追加 p，超过容量时丢掉最旧的部分。永远不报错。
func (r *Ring) Write(p []byte) (int, error) {
	if r == nil {
		return len(p), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if over := len(r.buf) - r.size; over > 0 {
		r.buf = append(r.buf[:0], r.buf[over:]...)
	}
	return len(p), nil
}

// Text 返回全部内容，已打码。
func (r *Ring) Text() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	raw := string(r.buf)
	r.mu.Unlock()
	return diaglog.Redact(validFrom(raw))
}

// Tail 返回最后约 n 字节，打码后把连续空白收成一个空格，用在一行错误说明里。
func (r *Ring) Tail(n int) string {
	text := r.Text()
	if n > 0 && len(text) > n {
		text = validFrom(text[len(text)-n:])
	}
	return strings.Join(strings.Fields(text), " ")
}

// validFrom 去掉开头被截断的半个 UTF-8 字符。
func validFrom(s string) string {
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r != utf8.RuneError || size > 1 {
			return s
		}
		s = s[1:]
	}
	return s
}
