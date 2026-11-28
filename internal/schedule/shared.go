package schedule

import "github.com/Shenchangxin/yoyo/internal/lease"

// begin 拿到 jobs.json 的跨进程锁并重读文件，返回放开锁的函数。调用方持有 s.mu。
// 桌面和 yoyo daemon 同时开着一个 home 时，一边新建或停掉的例行，拿着后台租约的那边下一拍就按新的到期；
// 谁改都先看到对方写下的，不互相盖掉。
func (s *Service) begin() func() {
	unlock, err := lease.Lock(s.path + ".lock")
	_ = s.load()
	if err != nil {
		return func() {}
	}
	return unlock
}
