// effects.go 是先记下、再执行的副作用队列。
// 同一个 kind 和 key 还没成功时再次入队，会换成新载荷并保持原来的顺序。

package superagent

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// ErrLater 表示这次先不做，不计失败次数。槽位满或暂停时用它。
var ErrLater = errors.New("later")

// Effect 是队列里的一行。Key 在同一个 Kind 里唯一。
type Effect struct {
	Kind     string
	Key      string
	Payload  any
	Attempts int
	Due      time.Time
	LastErr  string
	done     bool
	running  bool
}

// Handler 执行一条副作用。payload 是入队时的载荷。
type Handler func(payload any) error

// Queue 按入队顺序执行到期的副作用。同一类同时跑的次数由 Handle 的 limit 限制。
// ordered 为真的种类，前面一条没成功时后面的先等着。
type Queue struct {
	mu       sync.Mutex
	rows     []*Effect
	handlers map[string]Handler
	limits   map[string]int
	ordered  map[string]bool
}

// NewQueue 返回空队列。
func NewQueue() *Queue {
	return &Queue{handlers: map[string]Handler{}, limits: map[string]int{}, ordered: map[string]bool{}}
}

// Handle 登记 kind 的处理函数 fn。limit 小于 1 时当作 1，是这一类在一次 Pump 里同时执行的上限。
func (q *Queue) Handle(kind string, limit int, fn Handler) {
	q.handle(kind, limit, fn, false)
}

// HandleOrdered 和 Handle 一样，但同一类必须按入队顺序成功。前面一条没完成时，后面的不动。
func (q *Queue) HandleOrdered(kind string, limit int, fn Handler) {
	q.handle(kind, limit, fn, true)
}

// handle 登记处理函数。ordered 为真时这一类保持顺序。
func (q *Queue) handle(kind string, limit int, fn Handler, ordered bool) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.handlers == nil {
		q.handlers = map[string]Handler{}
	}
	if q.limits == nil {
		q.limits = map[string]int{}
	}
	if q.ordered == nil {
		q.ordered = map[string]bool{}
	}
	if limit < 1 {
		limit = 1
	}
	q.handlers[kind] = fn
	q.limits[kind] = limit
	if ordered {
		q.ordered[kind] = true
		if limit > 1 {
			q.limits[kind] = 1
		}
	}
}

// Enqueue 放入一条。kind 与 key 标识这一行，payload 是载荷。
// 相同 kind 和 key 还没成功时替换 Payload，并把 Attempts 清零、Due 改成现在，原来的队列位置保持不变。
func (q *Queue) Enqueue(kind, key string, payload any) {
	if q == nil || kind == "" || key == "" {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueueLocked(kind, key, payload)
}

// enqueueLocked 放入一条。调用时要已持有锁。
func (q *Queue) enqueueLocked(kind, key string, payload any) {
	now := time.Now()
	for _, row := range q.rows {
		if row.Kind == kind && row.Key == key && !row.done {
			row.Payload = payload
			row.Attempts = 0
			row.LastErr = ""
			row.Due = now
			return
		}
	}
	q.rows = append(q.rows, &Effect{Kind: kind, Key: key, Payload: payload, Due: now})
}

// effectJob 是一次泵里已经占住、准备在锁外执行的一行。
type effectJob struct {
	row     *Effect
	payload any
	fn      Handler
}

// Pump 执行 now 之前到期的行。同一种类不超过 Handle 登记的上限。
// 要保持顺序的种类，遇到还没成功的一条就停住这一类后面的行。处理函数在锁外调用。
func (q *Queue) Pump(now time.Time) {
	if q == nil {
		return
	}
	q.mu.Lock()
	ran := map[string]int{}
	held := map[string]bool{}
	var jobs []effectJob
	for _, row := range q.rows {
		if row == nil || row.done {
			continue
		}
		if row.running {
			if q.ordered[row.Kind] {
				held[row.Kind] = true
			}
			continue
		}
		if q.ordered[row.Kind] && held[row.Kind] {
			continue
		}
		if row.Due.After(now) {
			if q.ordered[row.Kind] {
				held[row.Kind] = true
			}
			continue
		}
		fn := q.handlers[row.Kind]
		if fn == nil {
			continue
		}
		limit := q.limits[row.Kind]
		if limit < 1 {
			limit = 1
		}
		if ran[row.Kind] >= limit {
			if q.ordered[row.Kind] {
				held[row.Kind] = true
			}
			continue
		}
		ran[row.Kind]++
		row.running = true
		if q.ordered[row.Kind] {
			held[row.Kind] = true
		}
		jobs = append(jobs, effectJob{row: row, payload: row.Payload, fn: fn})
	}
	q.mu.Unlock()
	for _, item := range jobs {
		q.finish(item.row, item.payload, item.fn(item.payload), now)
	}
}

// finish 写回一行的结果。载荷在执行期间被替换时，不把这次结果当成新载荷的结果。
func (q *Queue) finish(row *Effect, payload any, err error, now time.Time) {
	if q == nil || row == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	row.running = false
	if row.done {
		return
	}
	if row.Payload != payload {
		row.Due = now
		return
	}
	if err == nil {
		row.done = true
		row.LastErr = ""
		return
	}
	if errors.Is(err, ErrLater) {
		row.Due = now.Add(2 * time.Second)
		return
	}
	row.Attempts++
	row.LastErr = err.Error()
	shift := row.Attempts
	if shift > 5 {
		shift = 5
	}
	delay := time.Duration(1<<shift) * time.Second
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}
	row.Due = now.Add(delay)
}

// Hurry 让 kind 里键以 prefix 开头、还没成功也没在跑的行马上到期。返回改了几行。
func (q *Queue) Hurry(kind, prefix string) int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	n := 0
	for _, row := range q.rows {
		if row == nil || row.done || row.running || row.Kind != kind || !strings.HasPrefix(row.Key, prefix) {
			continue
		}
		row.Due = now
		n++
	}
	return n
}

// Has 在 kind 已经登记了处理函数时为真。
func (q *Queue) Has(kind string) bool {
	if q == nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.handlers[kind] != nil
}

// Failing 返回连续失败至少三次、还没完成的行。顺序与队列一致。
func (q *Queue) Failing() []savedEffect {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []savedEffect
	for _, row := range q.rows {
		if row == nil || row.done || row.Attempts < 3 {
			continue
		}
		payload, _ := row.Payload.(string)
		out = append(out, savedEffect{Kind: row.Kind, Key: row.Key, Payload: payload, Attempts: row.Attempts})
	}
	return out
}

// savedEffect 是落盘的一行。Payload 只保留字符串。处理函数不在快照里。
type savedEffect struct {
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	Payload  string `json:"payload,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

// ExportPending 返回还没成功、载荷是字符串的行。顺序与队列一致。
func (q *Queue) ExportPending() []savedEffect {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []savedEffect
	for _, row := range q.rows {
		if row == nil || row.done {
			continue
		}
		payload, ok := row.Payload.(string)
		if !ok {
			continue
		}
		out = append(out, savedEffect{Kind: row.Kind, Key: row.Key, Payload: payload, Attempts: row.Attempts})
	}
	return out
}

// Restore 把 rows 放回队列。相同 kind 和 key 只保留先到的那条，失败次数沿用快照。
func (q *Queue) Restore(rows []savedEffect) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range rows {
		if row.Kind == "" || row.Key == "" {
			continue
		}
		q.enqueueLocked(row.Kind, row.Key, row.Payload)
		for _, got := range q.rows {
			if got.Kind == row.Kind && got.Key == row.Key && !got.done {
				got.Attempts = row.Attempts
				break
			}
		}
	}
}

// Pending 返回还没成功的行数。
func (q *Queue) Pending() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, row := range q.rows {
		if !row.done {
			n++
		}
	}
	return n
}

// Healthy 在没有行连续失败三次或以上时为真。
func (q *Queue) Healthy() bool {
	if q == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range q.rows {
		if row != nil && !row.done && row.Attempts >= 3 {
			return false
		}
	}
	return true
}
