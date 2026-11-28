// Package lease 是同一个 yoyo home 上跨进程的互斥。
// Try 抢一份长期租约：谁拿着，谁跑后台（送达循环、简报时刻、定时例行、个人任务、IM 网关、编排轮次）。
// Lock 在改一份共享文件时短暂上锁，让桌面和 yoyo daemon 先重读磁盘再改，不互相盖掉对方写下的。
// 锁由操作系统持有：Windows 用 LockFileEx，其余系统用 flock。进程退出或崩溃时锁自动放开，所以不靠 pid 文件猜对方死活。
// 同一个进程里两次打开同一把锁也互斥，测试用它模拟两个进程。
package lease

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// ErrHeld 是别的进程拿着这份租约。
var ErrHeld = errors.New("lease: held by another process")

// Holder 是拿着租约的进程。写在锁文件旁边的 .json 里，给另一个进程的界面看。Since 是 RFC3339。
// Draining 为真时这个进程已经决定让出，正在等手上在跑的回合跑完，跑完就放开租约。
type Holder struct {
	Role     string `json:"role"`
	PID      int    `json:"pid"`
	Since    string `json:"since"`
	Draining bool   `json:"draining,omitempty"`
}

// Lease 是拿到手的一份租约。Release 之前一直有效。
type Lease struct {
	mu   sync.Mutex
	f    *os.File
	path string
}

// Update 重写持有人记录，租约本身不动。领头的进程进入收尾、或者收尾取消时用它改 Draining。
// 已经放开的租约什么都不写，返回 nil：放开时记录已经删掉了，再写会让别人以为还有人拿着。
func (l *Lease) Update(h Holder) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	return writeHolder(l.path+".json", h)
}

// Try 不等待地抢 path 这把锁。别人拿着时返回 ErrHeld。拿到之后把 h 写进 path.json。
func Try(path string, h Holder) (*Lease, error) {
	f, err := openLock(path)
	if err != nil {
		return nil, err
	}
	if err := tryLock(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	_ = writeHolder(path+".json", h)
	return &Lease{f: f, path: path}, nil
}

// Release 放开租约并删掉持有人记录。重复调用无害。
func (l *Lease) Release() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return
	}
	_ = os.Remove(l.path + ".json")
	_ = unlock(l.f)
	_ = l.f.Close()
	l.f = nil
}

// ReadHolder 读出 path 现在的持有人。没人拿着、或者记录是崩溃的进程留下的时 ok 为假。
// 判断有没有人拿着要短暂试一下锁，所以只给界面偶尔读，不放在热路径上。
func ReadHolder(path string) (Holder, bool) {
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		return Holder{}, false
	}
	var h Holder
	if json.Unmarshal(raw, &h) != nil {
		return Holder{}, false
	}
	f, err := openLock(path)
	if err != nil {
		return h, true
	}
	defer f.Close()
	if err := tryLock(f); err == nil {
		_ = unlock(f)
		return Holder{}, false
	}
	return h, true
}

// Lock 等到拿到 path 这把锁，返回放开它的函数。用来包住一次读改写。打不开锁文件时返回错误和一个什么都不做的函数。
func Lock(path string) (func(), error) {
	f, err := openLock(path)
	if err != nil {
		return func() {}, err
	}
	if err := lockWait(f); err != nil {
		_ = f.Close()
		return func() {}, err
	}
	return func() {
		_ = unlock(f)
		_ = f.Close()
	}, nil
}

func openLock(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
}

func writeHolder(path string, h Holder) error {
	raw, err := json.Marshal(h)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
