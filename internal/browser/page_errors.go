// 本文件收集两次浏览器调用之间页面抛出的 JavaScript 错误。
// Runtime.exceptionThrown 和 console.error 记下来，其余 CDP 事件忽略。同一句重复只加计数。最多留 pageErrorCap 条不同的错误。

package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// pageErrorCap 是两次浏览器调用之间保留的不同错误条数，20。超出的只计个数，用来限制内存。
const pageErrorCap = 20

// pageErrors 收集页面在两次浏览器调用之间抛出的 JavaScript 错误。读循环写入，宿主取走。
type pageErrors struct {
	mu      sync.Mutex
	order   []string
	count   map[string]int
	dropped int // 超过上限的不同错误条数，报告里只写个数
}

// add 记下一条 CDP 事件。不是页面错误的事件忽略。新错误超过 pageErrorCap 时只增加 dropped。
// method 传给 pageErrorFromEvent。
// params 传给 pageErrorFromEvent。返回的文字为空时不加记录。
func (p *pageErrors) add(method string, params json.RawMessage) {
	msg := pageErrorFromEvent(method, params)
	if msg == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.count == nil {
		p.count = map[string]int{}
	}
	if p.count[msg] == 0 {
		if len(p.order) >= pageErrorCap {
			p.dropped++
			return
		}
		p.order = append(p.order, msg)
	}
	p.count[msg]++
}

// take 把已收集的错误写成一段报告并清空。没有错误时返回空字符串。重复的带上次数，被丢掉的条数写在末尾。
func (p *pageErrors) take() string {
	p.mu.Lock()
	order, count, dropped := p.order, p.count, p.dropped
	p.order, p.count, p.dropped = nil, nil, 0
	p.mu.Unlock()
	if len(order) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("page errors raised since the last browser call (JavaScript in the page; check these before judging the page):")
	for _, m := range order {
		if n := count[m]; n > 1 {
			fmt.Fprintf(&b, "\n- %s (x%d)", m, n)
		} else {
			fmt.Fprintf(&b, "\n- %s", m)
		}
	}
	if dropped > 0 {
		fmt.Fprintf(&b, "\n- ... and %d more distinct errors not shown", dropped)
	}
	return b.String()
}

// pageErrorFromEvent 把一条 CDP 事件收成一行。Runtime.exceptionThrown 用异常描述和文件行号；Runtime.consoleAPICalled 只收 type 为 error 的。其他事件返回空字符串。
// method 为 Runtime.exceptionThrown 或 Runtime.consoleAPICalled 时才解析，其他值返回空字符串。
// params 按 method 解成异常详情或控制台参数。Unmarshal 失败时返回空字符串。
func pageErrorFromEvent(method string, params json.RawMessage) string {
	switch method {
	case "Runtime.exceptionThrown":
		var p struct {
			ExceptionDetails struct {
				Text       string `json:"text"`
				LineNumber int    `json:"lineNumber"`
				URL        string `json:"url"`
				Exception  *struct {
					Description string `json:"description"`
				} `json:"exception"`
			} `json:"exceptionDetails"`
		}
		if json.Unmarshal(params, &p) != nil {
			return ""
		}
		d := p.ExceptionDetails
		what := d.Text
		if d.Exception != nil && strings.TrimSpace(d.Exception.Description) != "" {
			what = d.Exception.Description
		}
		return truncRunes(firstLine(what), 300) + shortLocation(d.URL, d.LineNumber)
	case "Runtime.consoleAPICalled":
		var p struct {
			Type string `json:"type"`
			Args []struct {
				Type        string `json:"type"`
				Value       any    `json:"value"`
				Description string `json:"description"`
			} `json:"args"`
		}
		if json.Unmarshal(params, &p) != nil || p.Type != "error" {
			return ""
		}
		parts := make([]string, 0, len(p.Args))
		for _, a := range p.Args {
			parts = append(parts, consoleArgText(a.Value, a.Description, a.Type))
		}
		text := strings.TrimSpace(strings.Join(parts, " "))
		if text == "" {
			text = "(no message)"
		}
		return "console.error: " + truncRunes(firstLine(text), 300)
	}
	return ""
}

// consoleArgText 取控制台参数的字符串。优先 value 里的字符串，其次 description，再其次 fmt 打印 value，最后用类型名。
// typ 在 value 不是字符串、description 为空且 value 为 nil 时作为返回值。
func consoleArgText(value any, description, typ string) string {
	if s, ok := value.(string); ok {
		return s
	}
	if description != "" {
		return description
	}
	if value != nil {
		return fmt.Sprint(value)
	}
	return typ
}

// firstLine 只留第一行。堆栈很长，报告要的是错误本身。
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// shortLocation 写成「 (文件名:行号)」。CDP 的行号从 0 起，这里加 1。地址为空时返回空字符串。问号和井号后面的部分丢掉。
// raw 为空时返回空字符串。否则丢掉 ? 或 # 及其后面的部分，取最后一个 / 后面的名字，名字空则用剩下的整段。
// line0 小于 0 时只写文件名，否则行号写成 line0 加 1。
func shortLocation(raw string, line0 int) string {
	if raw == "" {
		return ""
	}
	u := raw
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	name := u[strings.LastIndex(u, "/")+1:]
	if name == "" {
		name = u
	}
	if line0 < 0 {
		return " (" + name + ")"
	}
	return fmt.Sprintf(" (%s:%d)", name, line0+1)
}

// takePageErrors 取走当前页面积累的错误。Chrome 没在跑时返回空字符串。
func (h *Host) takePageErrors() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	c := h.cdp
	h.mu.Unlock()
	if c == nil {
		return ""
	}
	return c.errs.take()
}

// PageErrors 取走上次浏览器调用以来的页面错误。给不经过 Click、Type、Refresh 的调用方用，例如打开本地文件。
func (h *Host) PageErrors() string { return h.takePageErrors() }

// pageErrorsNote 在报告前加一个换行，方便接在工具结果后面。没有错误时返回空字符串。
func (h *Host) pageErrorsNote() string {
	if s := h.takePageErrors(); s != "" {
		return "\n" + s
	}
	return ""
}
