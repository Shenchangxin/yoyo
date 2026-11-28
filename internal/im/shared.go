// shared.go 让桌面和 yoyo daemon 共用一份 im.json：改之前看文件是不是被另一个进程换过，换过就先重读。
// 跑着网关的那个进程每两秒调一次 Sync：另一个进程在设置里改了账号或规则，它重读并按新的设置重启适配器。

package im

import (
	"os"
	"time"
)

// imStamp 是 im.json 的修改时间和长度。
type imStamp struct {
	mod  time.Time
	size int64
}

func imStampOf(path string) imStamp {
	st, err := os.Stat(path)
	if err != nil {
		return imStamp{}
	}
	return imStamp{mod: st.ModTime(), size: st.Size()}
}

// refreshLocked 在 im.json 被别的进程改过时重读。网关正在跑时记下还要重启，留给下一次 Sync。调用方持有 rt.mu。
func (rt *Runtime) refreshLocked() {
	st := imStampOf(rt.path)
	if st.mod.IsZero() || st == rt.stamp {
		return
	}
	if rt.load() == nil && rt.root != nil {
		rt.pending = true
	}
}

// Sync 在 im.json 被别的进程改过时重读；网关正在跑时按新的账号和规则重启。没有变化时什么都不做。
func (rt *Runtime) Sync() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.refreshLocked()
	restart := rt.pending && rt.root != nil
	rt.pending = false
	rt.mu.Unlock()
	if restart {
		rt.reload()
	}
}
