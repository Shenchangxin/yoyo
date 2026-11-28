// finish.go 是「最后一下交给你」：付款、下单、预订、提交申请这类只能人来按的最后一步，工人不按。
// 工人把能准备的都准备好，在 orchestrator_report 里带上 finish：要人做的那一下、要打开的页面、价格和截止。
// 带了 finish 的报告记成 blocked，Need 是 finish，工单停下等人；人办完或说不用了，回一句话给工单，工人再收尾。

package superagent

import "strings"

// Finish 是交给人的最后一步。What 必填；URL 只收 http 和 https。
type Finish struct {
	What     string `json:"what"`
	URL      string `json:"url,omitempty"`
	Price    string `json:"price,omitempty"`
	Deadline string `json:"deadline,omitempty"`
}

// clean 去掉空白、截短，丢掉不是网页的链接。没有 What 时返回 nil。
func (f *Finish) clean() *Finish {
	if f == nil {
		return nil
	}
	out := Finish{
		What:     clipRunes(strings.TrimSpace(f.What), 200),
		URL:      strings.TrimSpace(f.URL),
		Price:    clipRunes(strings.TrimSpace(f.Price), 60),
		Deadline: clipRunes(strings.TrimSpace(f.Deadline), 60),
	}
	if out.What == "" {
		return nil
	}
	low := strings.ToLower(out.URL)
	if !(strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "http://")) || len(out.URL) > 2000 || strings.ContainsAny(out.URL, " \r\n") {
		out.URL = ""
	}
	return &out
}

// TicketFinish 返回 id 最近一份报告交给人的最后一步。最近那份没有时为 nil。
func (rt *Runtime) TicketFinish(id string) *Finish {
	if rt == nil || rt.eng == nil {
		return nil
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return nil
	}
	asg := latestAssignment(&t)
	if asg == nil || len(asg.Reports) == 0 {
		return nil
	}
	f := asg.Reports[len(asg.Reports)-1].Finish
	if f == nil {
		return nil
	}
	copied := *f
	return &copied
}
