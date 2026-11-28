package superagent

// Discard 关掉数据库，不落盘。给只读一眼的进程用：桌面和 yoyo daemon 同时开着时，不领头的那个读完编排台就用它关掉，
// 免得把自己内存里旧的快照写回去，盖掉领头进程刚写下的工单。没有 Start 过的运行时才该用它。
func (rt *Runtime) Discard() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	db := rt.db
	rt.db = nil
	rt.mu.Unlock()
	if db != nil {
		_ = db.Close()
	}
}
