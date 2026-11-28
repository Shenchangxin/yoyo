package superagent

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/agentloop"
)

// TestReportFinishHandsTheLastStepToThePerson 锁住：带 finish 的报告即使写的是 ready 也记成 blocked，Need 是 finish，
// 工单停下等人；问题自动写成「最后一步交给你」；不是网页的链接丢掉。
func TestReportFinishHandsTheLastStepToThePerson(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	res, err := (&Executor{Engine: eng}).reportToolWith(id, true).Run(context.Background(), agentloop.ToolCall{
		Arguments: `{"state":"ready","summary":"比过三家，选了这一家","finish":"付款下单","finish_url":"https://shop.example/cart/42","finish_price":"¥199","finish_deadline":"今晚 24 点前"}`,
	})
	if err != nil || res.IsError || res.Text != "recorded blocked" {
		t.Fatalf("%+v %v", res, err)
	}
	rt := &Runtime{eng: eng}
	f := rt.TicketFinish(id)
	if f == nil || f.What != "付款下单" || f.URL != "https://shop.example/cart/42" || f.Price != "¥199" || f.Deadline != "今晚 24 点前" {
		t.Fatalf("finish = %+v", f)
	}
	if rt.TicketNeed(id) != "finish" {
		t.Fatalf("need = %q", rt.TicketNeed(id))
	}
	tk, _ := eng.Ticket(id)
	if tk.Ended != nil || tk.Attention != "最后一步交给你：付款下单" {
		t.Fatalf("ticket = ended %v attention %q", tk.Ended, tk.Attention)
	}

	other := acceptOn(t, eng, "m2", "room2")
	if _, err := (&Executor{Engine: eng}).reportToolWith(other, false).Run(context.Background(), agentloop.ToolCall{
		Arguments: `{"state":"blocked","summary":"表填好了","finish":"提交申请","finish_url":"javascript:alert(1)"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if f := rt.TicketFinish(other); f == nil || f.URL != "" {
		t.Fatalf("a non-web link must be dropped: %+v", f)
	}
	if f := (&Finish{What: "  "}).clean(); f != nil {
		t.Fatalf("empty what = %+v", f)
	}
}
