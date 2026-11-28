package personal

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Shenchangxin/yoyo/internal/lease"
)

// storeStamp 是 workspace.json 上次读到或写下时的修改时间和长度。
type storeStamp struct {
	mod  time.Time
	size int64
}

func storeStampOf(path string) storeStamp {
	st, err := os.Stat(path)
	if err != nil {
		return storeStamp{}
	}
	return storeStamp{mod: st.ModTime(), size: st.Size()}
}

// begin 拿到 workspace.json 的跨进程锁；文件被别的进程改过时先重读。返回放开锁的函数。调用方持有 s.mu。
// 桌面和 yoyo daemon 同时开着时，只有领头的进程跑个人任务；另一个在界面上新建、批准、取消的东西，领头的下一拍就看得到，谁写都不盖掉对方。
func (s *Store) begin() func() {
	unlock, _ := lease.Lock(s.path + ".lock")
	if st := storeStampOf(s.path); !st.mod.IsZero() && st != s.stamp {
		s.reloadLocked(st)
	}
	return unlock
}

func (s *Store) reloadLocked(st storeStamp) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var d doc
	if json.Unmarshal(b, &d) != nil {
		return
	}
	if d.SamplePages == nil {
		d.SamplePages = map[string]string{}
	}
	if d.LastIdeasAt.IsZero() {
		d.LastIdeasAt = s.doc.LastIdeasAt
	}
	s.doc = d
	s.stamp = st
}
