// pool.go 拥有外部 agent 进程的总上限：同时活着的进程最多 maxLive 个，超了就停掉最久没用的空闲工人（会话留着，下次续接），
// 以及应用退出时的 CloseAll。
// 不拥有单个 agent 的并行上限（执行器按 Buddy 设置管），也不拥有工人对象的生命周期（工单结束时执行器关）。

package acpworker

import (
	"sort"
	"sync"
)

// maxLive 是同时活着的外部 agent 进程上限。
const maxLive = 6

// pool 记着打开着的工人和其中连着进程的那些。不要在持有工人锁时调用它的方法。
type pool struct {
	mu      sync.Mutex
	workers map[*Worker]bool
	live    map[*Worker]bool
	max     int
}

// processes 是全进程共用的池。
var processes = newPool(maxLive)

// newPool 建一个上限为 max 的池。
func newPool(max int) *pool {
	return &pool{workers: map[*Worker]bool{}, live: map[*Worker]bool{}, max: max}
}

// track 记下打开了 w。
func (p *pool) track(w *Worker) {
	p.mu.Lock()
	p.workers[w] = true
	p.mu.Unlock()
}

// forget 在 w 关掉后把它整个忘掉。
func (p *pool) forget(w *Worker) {
	p.mu.Lock()
	delete(p.workers, w)
	delete(p.live, w)
	p.mu.Unlock()
}

// add 记下 w 连上了进程。超过上限时在后台停掉最久没用的空闲工人，忙着的不动。
func (p *pool) add(w *Worker) {
	p.mu.Lock()
	p.live[w] = true
	victims := p.overLocked(w)
	p.mu.Unlock()
	for _, v := range victims {
		go v.suspend()
	}
}

// overLocked 挑出超出上限的那几个空闲工人，最久没用的在前。keep 不算在内。
func (p *pool) overLocked(keep *Worker) []*Worker {
	over := len(p.live) - p.max
	if over <= 0 {
		return nil
	}
	var idle []*Worker
	for w := range p.live {
		if w != keep && !w.busy.Load() {
			idle = append(idle, w)
		}
	}
	sort.Slice(idle, func(i, j int) bool { return idle[i].usedAt.Load() < idle[j].usedAt.Load() })
	if len(idle) > over {
		idle = idle[:over]
	}
	return idle
}

// remove 记下 w 的进程停了。
func (p *pool) remove(w *Worker) {
	p.mu.Lock()
	delete(p.live, w)
	p.mu.Unlock()
}

// size 返回活着的进程数。
func (p *pool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.live)
}

// closeAll 让此刻打开着的工人都停下、不再连接，并停掉它们的进程，不关会话。之后新开的工人不受影响。
func (p *pool) closeAll() {
	p.mu.Lock()
	ws := make([]*Worker, 0, len(p.workers))
	for w := range p.workers {
		ws = append(ws, w)
	}
	p.live = map[*Worker]bool{}
	p.mu.Unlock()
	var wg sync.WaitGroup
	for _, w := range ws {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.halt()
		}()
	}
	wg.Wait()
}

// CloseAll 在应用退出时调用：停掉所有外部 agent 进程，这些工人之后不再连接。会话不关，下次启动时续接。
func CloseAll() { processes.closeAll() }
