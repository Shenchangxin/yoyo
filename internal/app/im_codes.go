// im_codes.go 把 IM 主通道的短编号（/答 abc、/批准 k2m 里的 abc、k2m）和引用编号（平台消息编号 → 短编号，见 im_quote.go）落到 im/codes.json。
// 以前两张表只在内存里：重启之后连 /答 abc 都认不出；daemon 和桌面换手之后，新领头的进程也认不出旧进程发出去的编号。
//
// 做法和 9.2 里其他共享文件一样：每次发号、查号、删号、记引用、找引用，都先拿文件锁（codes.json.lock）、重读磁盘，
// 改完写回（先写 .tmp 再改名）。这些动作只在往主通道发消息、收到命令时发生，次数很少，所以不在内存里缓存文件状态，每次都读。
// 读到的过期项（超过 imCodeTTL）顺手丢掉。锁的顺序固定是先 imh.mu、再文件锁，两个进程按同一顺序拿，不会互相等死。
// 不拥有：发号和认号的规则（im_home.go 的 imIssue / imLookup，im_quote.go 的 quotedCode）。

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/lease"
)

// imCodesFile 是 im/codes.json 的样子。键都是短编号或「账号\n平台编号」，值里带发出时刻，过期按 imCodeTTL 算。
type imCodesFile struct {
	Codes     map[string]imCode   `json:"codes,omitempty"`
	LastSpeak string              `json:"last_speak,omitempty"`
	Quotes    map[string]imQuoted `json:"quotes,omitempty"`
}

// imCodesPath 是编号文件的位置，和主通道绑定（im/home.json）放在一起。没有 home 时为空，这时只用内存（测试和 Open 之前）。
func (a *App) imCodesPath() string {
	if a == nil || a.Home == nil || strings.TrimSpace(a.Home.Root) == "" {
		return ""
	}
	return filepath.Join(a.Home.Root, "im", "codes.json")
}

// imCodesEdit 在文件锁里重读编号、执行 fn、fn 返回真时写回。调用方持有 imh.mu。
// fn 直接读改 a.imh.codes / a.imh.lastSpeak / a.imh.quotes；只读的查询返回假，不写盘。
// 没有 home 时直接执行 fn，只改内存。锁或读写失败时也照常执行 fn（退回只在内存里生效），不让发消息因为磁盘问题失败。
func (a *App) imCodesEdit(fn func() bool) {
	path := a.imCodesPath()
	if path == "" {
		fn()
		return
	}
	unlock, err := lease.Lock(path + ".lock")
	defer unlock()
	if err == nil {
		a.imCodesLoadLocked(path)
	}
	if fn() && err == nil {
		a.imCodesSaveLocked(path)
	}
}

// imCodesLoadLocked 用磁盘上的那份替换内存里的两张表，并丢掉过期项。文件还不存在时保留内存里的（第一次发号之前）。
// 文件坏了（解析失败）时也保留内存里的，下一次写回会把它盖成好的。调用方持有 imh.mu 和文件锁。
func (a *App) imCodesLoadLocked(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var f imCodesFile
	if json.Unmarshal(raw, &f) != nil {
		return
	}
	now := time.Now()
	codes := map[string]imCode{}
	for k, v := range f.Codes {
		if now.Sub(v.At) <= imCodeTTL {
			codes[k] = v
		}
	}
	quotes := map[string]imQuoted{}
	for k, v := range f.Quotes {
		if now.Sub(v.At) <= imCodeTTL {
			quotes[k] = v
		}
	}
	a.imh.codes, a.imh.quotes = codes, quotes
	// 最近一条短讯的号过期了就不再当「最近」：/有用 不带号时不该认到三天前那条。
	if _, ok := codes[f.LastSpeak]; ok {
		a.imh.lastSpeak = f.LastSpeak
	} else {
		a.imh.lastSpeak = ""
	}
}

// imCodesSaveLocked 把内存里的两张表写回磁盘。写失败时不报错：这一次的编号仍在内存里有效，只是重启后认不出。
// 调用方持有 imh.mu 和文件锁。
func (a *App) imCodesSaveLocked(path string) {
	raw, err := json.Marshal(imCodesFile{Codes: a.imh.codes, LastSpeak: a.imh.lastSpeak, Quotes: a.imh.quotes})
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// imCodesResetLocked 清空两张表并写回空文件：解绑或绑到新的对话时，旧编号不能再用。调用方持有 imh.mu。
// 写成空文件而不是删掉，这样另一个进程下一次重读时也会清空自己内存里的那份。
func (a *App) imCodesResetLocked() {
	a.imCodesEdit(func() bool {
		a.imh.codes, a.imh.quotes, a.imh.lastSpeak = nil, nil, ""
		return true
	})
}
