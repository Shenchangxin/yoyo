package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// pageErrorCap bounds how many distinct page errors are kept between two
// browser calls. The cap protects memory; repeats only bump a counter.
const pageErrorCap = 20

// pageErrors collects the JavaScript errors a page raises between browser
// calls. Before this existed, Runtime.exceptionThrown and console.error were
// read off the socket and dropped, so a page whose handlers threw on every
// click looked identical to one whose clicks simply did nothing.
type pageErrors struct {
	mu      sync.Mutex
	order   []string
	count   map[string]int
	dropped int // distinct errors past the cap; reported as a count
}

// add records one CDP event. Events that are not page errors are ignored.
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

// take returns the collected errors as a report and clears the log. It
// returns "" when nothing was raised.
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

// pageErrorFromEvent turns one CDP event into a one-line report, or "" when
// the event is not an error.
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

// firstLine keeps the first line of a stack-carrying message. The stack is
// long and the agent needs the error itself, not the frames.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// shortLocation names the file and line of an error: " (index.html:1029)".
// CDP reports line numbers from zero.
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

// takePageErrors drains the errors raised by the live page. It returns ""
// when Chrome is not running.
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

// PageErrors drains the page errors raised since the last browser call. It is
// for a caller that reports a page without going through Click, Type or
// Refresh, such as opening a local file.
func (h *Host) PageErrors() string { return h.takePageErrors() }

// pageErrorsNote is takePageErrors shaped for appending to a tool result: a
// leading newline and the report, or "" when there is nothing to report.
func (h *Host) pageErrorsNote() string {
	if s := h.takePageErrors(); s != "" {
		return "\n" + s
	}
	return ""
}
