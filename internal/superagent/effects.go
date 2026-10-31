// effects.go 是先记下、再执行的副作用队列。
// 同一个 kind 和 key 还没成功时再次入队，会换成新载荷并保持原来的顺序。

package superagent

import (
	"errors"
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
}

// Handler 执行一条副作用。payload 是入队时的载荷。
type Handler func(payload any) error

// Queue 按入队顺序执行到期的副作用。同一类同时跑的次数由 Handle 的 limit 限制。
type Queue struct {
	rows     []*Effect
	handlers map[string]Handler
	limits   map[string]int
}

// NewQueue 返回空队列。
func NewQueue() *Queue {
	return &Queue{handlers: map[string]Handler{}, limits: map[string]int{}}
}

// Handle 登记 kind 的处理函数 fn。limit 小于 1 时当作 1，是这一类在一次 Pump 里同时执行的上限。
func (q *Queue) Handle(kind string, limit int, fn Handler) {
	if limit < 1 {
		limit = 1
	}
	q.handlers[kind] = fn
	q.limits[kind] = limit
}

// Enqueue 放入一条。kind 与 key 标识这一行，payload 是载荷。
// 相同 kind 和 key 还没成功时替换 Payload，并把 Attempts 清零、Due 改成现在，原来的队列位置保持不变。
func (q *Queue) Enqueue(kind, key string, payload any) {
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

// Pump 执行 now 之前到期的行。同一种类不超过 Handle 登记的上限。
func (q *Queue) Pump(now time.Time) {
	ran := map[string]int{}
	for _, row := range q.rows {
		if row.done || row.Due.After(now) {
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
			continue
		}
		ran[row.Kind]++
		err := fn(row.Payload)
		if err == nil {
			row.done = true
			row.LastErr = ""
			continue
		}
		if errors.Is(err, ErrLater) {
			row.Due = now.Add(5 * time.Second)
			continue
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
}

// Pending 返回还没成功的行数。
func (q *Queue) Pending() int {
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
	for _, row := range q.rows {
		if row.Attempts >= 3 {
			return false
		}
	}
	return true
}
