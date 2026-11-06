// 本文件锁住记忆怎么随时间和改口变化：相对日子换成日期，过去的安排离开画像；改口留历史，相近的交回给写的人；画像先放当事人的话。

package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// friday 是测试里的「今天」：2026-11-06，星期五。
var friday = time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local)

func openAt(t *testing.T, now *time.Time) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return *now }
	return s
}

func TestAnchorDatesResolvesRelativeDays(t *testing.T) {
	cases := []struct{ in, out, until string }{
		{"明天开会", "明天（2026-11-07）开会", "2026-11-07"},
		{"下周三去上海出差", "下周三（2026-11-11）去上海出差", "2026-11-11"},
		{"这周五前交周报", "这周五（2026-11-06）前交周报", "2026-11-06"},
		{"周报周五发", "周报周五发", ""},
		{"Call Ada on Monday", "Call Ada on Monday (2026-11-09)", "2026-11-09"},
		{"上周三和 Ada 吃过饭", "上周三（2026-10-28）和 Ada 吃过饭", ""},
		{"三天后交报告", "三天后（2026-11-09）交报告", "2026-11-09"},
		{"这个月底前搬家", "这个月底（2026-11-30）前搬家", "2026-11-30"},
		{"11月20日前交付首页", "11月20日前交付首页", "2026-11-20"},
		{"明天开始在新公司上班", "明天（2026-11-07）开始在新公司上班", ""},
		{"每周五开周会", "每周五开周会", ""},
		{"一周一次健身", "一周一次健身", ""},
		{"目前天气不错，说明天气预报准", "目前天气不错，说明天气预报准", ""},
		{"Dentist tomorrow", "Dentist tomorrow (2026-11-07)", "2026-11-07"},
		{"Review next Monday", "Review next Monday (2026-11-09)", "2026-11-09"},
		{"Standup every Friday", "Standup every Friday", ""},
	}
	for _, c := range cases {
		out, until := anchorDates(c.in, friday)
		if out != c.out || until != c.until {
			t.Errorf("anchorDates(%q) = %q, %q; want %q, %q", c.in, out, until, c.out, c.until)
		}
		if again, _ := anchorDates(out, friday); again != out {
			t.Errorf("anchoring twice changed %q into %q", out, again)
		}
	}
}

func TestPastPlansLeaveThePinButStayRecallable(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	kept := s.Keep(Item{Kind: KindProfile, Text: "明天去上海出差", Source: "User confirmed in chat"}, "")
	if kept.Item.ValidUntil != "2026-11-07" || !strings.Contains(kept.Item.Text, "2026-11-07") {
		t.Fatalf("kept = %+v", kept.Item)
	}
	if !strings.Contains(s.ProfilePin(0), "上海出差") {
		t.Fatal("a plan that has not passed belongs in the pin")
	}
	now = friday.AddDate(0, 0, 2)
	if strings.Contains(s.ProfilePin(0), "上海出差") || strings.Contains(s.OperatorPin(0), "上海出差") {
		t.Fatal("a past plan must not read as a current fact")
	}
	if got := s.Recall("上海出差的报销", 3); len(got) != 1 || got[0].ID != kept.Item.ID {
		t.Fatalf("recall = %+v", got)
	}
}

func TestKeepTracksChangesOfMind(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	first := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用中文", Source: "user"}, "")
	if first.Action != "added" {
		t.Fatalf("first = %+v", first)
	}
	now = now.Add(time.Hour)
	conflict := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用英文", Source: "User confirmed in chat"}, "")
	if conflict.Action != "added" || len(conflict.Related) != 1 || conflict.Related[0].ID != first.Item.ID {
		t.Fatalf("a conflicting statement must come back with the old one: %+v", conflict)
	}
	now = now.Add(time.Hour)
	swap := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用英文", Source: "User confirmed in chat"}, first.Item.ID)
	if swap.Action != "replaced" || swap.Before != "代码注释用中文" || swap.Item.ID != first.Item.ID {
		t.Fatalf("swap = %+v", swap)
	}
	if len(swap.Item.History) != 1 || swap.Item.History[0].Text != "代码注释用中文" || !swap.Item.CreatedAt.Equal(first.Item.CreatedAt) {
		t.Fatalf("history = %+v", swap.Item)
	}
	if all := s.Search("代码注释", KindSemantic, 10); len(all) != 1 {
		t.Fatalf("the duplicate left by the swap should go: %+v", all)
	}
	if got := s.Keep(Item{Kind: KindSemantic, Text: "回答可以详细一些", Source: "user"}, ""); got.Action != "added" {
		t.Fatalf("unrelated = %+v", got)
	}
	short := s.Keep(Item{Kind: KindSemantic, Text: "回答要简短", Source: "user"}, "")
	if len(short.Related) != 1 || short.Related[0].Text != "回答可以详细一些" {
		t.Fatalf("related = %+v", short.Related)
	}
}

func TestKeepMergesRewordingsWithoutLettingTheModelOverwrite(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	said := s.Keep(Item{Kind: KindSemantic, Text: "报告用中文，先写结论", Source: "user"}, "")
	now = now.Add(time.Hour)
	noted := s.Keep(Item{Kind: KindSemantic, Text: "报告用中文；先写结论", Source: "model"}, "")
	if noted.Action != "merged" || noted.Item.ID != said.Item.ID || noted.Item.Text != "报告用中文，先写结论" || noted.Item.Source != "user" || noted.Item.Seen != 2 {
		t.Fatalf("noted = %+v", noted)
	}
	now = now.Add(time.Hour)
	again := s.Keep(Item{Kind: KindSemantic, Text: "报告用中文；先写结论", Source: "user"}, "")
	if again.Action != "merged" || again.Item.Text != "报告用中文；先写结论" || len(again.Item.History) != 1 {
		t.Fatalf("again = %+v", again)
	}
	same := s.Keep(Item{Kind: KindSemantic, Text: "报告用中文；先写结论。", Source: "model"}, "")
	if same.Action != "same" || same.Item.Seen != 4 || len(s.All()) != 1 {
		t.Fatalf("same = %+v all %d", same, len(s.All()))
	}
}

func TestPinsPutTheOperatorFirst(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	s.FileMachine([]MachineFact{
		{ID: "machine:device", Kind: "profile", Project: "device", Text: "scx"},
		{ID: "machine:host", Kind: "profile", Project: "device", Text: "DESKTOP-LONGHOSTNAME"},
		{ID: "machine:os", Kind: "profile", Project: "device", Text: "Windows 11 Pro 26300"},
		{ID: "machine:zone", Kind: "profile", Project: "device", Text: "Asia/Shanghai (UTC+8)"},
		{ID: MachineID("project", "yoyo"), Kind: "project", Project: "yoyo", Text: "yoyo", State: "moving", Evidence: "internal/memory/memory.go", When: "2026-11-06"},
	})
	for i, text := range []string{"报告用中文", "先写结论再写过程", "代码注释写清楚约束", "提交信息用英文", "不要用表情符号", "回复控制在三段以内"} {
		now = friday.Add(time.Duration(i) * time.Minute)
		s.Keep(Item{Kind: KindSemantic, Text: text, Source: "user"}, "")
	}
	s.Keep(Item{Kind: KindProfile, Text: "叫我阿杰", Source: "User confirmed in chat"}, "")
	pin := s.ProfilePin(190)
	if !strings.Contains(pin, "叫我阿杰") || !strings.Contains(pin, "回复控制在三段以内") {
		t.Fatalf("the newest words must survive a tight budget: %q", pin)
	}
	if strings.Contains(pin, "报告用中文") || strings.Contains(pin, "DESKTOP-LONGHOSTNAME") {
		t.Fatalf("the oldest preference and the machine go first: %q", pin)
	}
	full := s.ProfilePin(0)
	if strings.Index(full, "Said by the operator") > strings.Index(full, "Preferences") || strings.Index(full, "Preferences") > strings.Index(full, "From this machine") {
		t.Fatalf("order = %q", full)
	}
	if len(pin) > 190 {
		t.Fatalf("budget ignored: %d", len(pin))
	}
}

func TestOldLinesCarryTheirDate(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	s.Keep(Item{Kind: KindSemantic, Text: "周报周五发", Source: "user", CreatedAt: friday.AddDate(0, 0, -120).UTC()}, "")
	s.Keep(Item{Kind: KindSemantic, Text: "图表用浅色", Source: "user"}, "")
	pin := s.ProfilePin(0)
	if !strings.Contains(pin, "周报周五发 (as of 2026-07-09)") || strings.Contains(pin, "图表用浅色 (as of") {
		t.Fatalf("pin = %q", pin)
	}
}

func TestSearchWeighsRareTerms(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	for _, text := range []string{"项目周会在周一", "项目文档放在共享盘", "项目预算下周再看", "飞书机器人接到群里"} {
		s.Keep(Item{Kind: KindProject, Text: text, Source: "user"}, "")
	}
	got := s.Search("飞书项目", "", 4)
	if len(got) == 0 || got[0].Text != "飞书机器人接到群里" {
		t.Fatalf("a rare word must outrank a common one: %+v", got)
	}
}

func TestRecallSkipsWeakMatchesAndPinnedFacts(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	done := s.Keep(Item{ID: "task:tkt-3", Kind: KindEpisodic, Text: "首页改版：标题换成新的，配色改成浅色", Source: "worker"}, "")
	s.Keep(Item{Kind: KindSemantic, Text: "首页标题用中文", Source: "user"}, "")
	if got := s.Recall("首页改版的配色还要再调一下", 3); len(got) != 1 || got[0].ID != done.Item.ID {
		t.Fatalf("recall = %+v", got)
	}
	for _, q := range []string{"好的，谢谢", "首页", ""} {
		if got := s.Recall(q, 3); len(got) != 0 {
			t.Fatalf("weak query %q recalled %+v", q, got)
		}
	}
	if got := s.Recall("首页改版的配色再看看，日志如下："+strings.Repeat("panic: nil map ", 400), 3); len(got) != 1 {
		t.Fatalf("the ask at the top of a long paste should still recall: %+v", got)
	}
	if got := s.Recall(strings.Repeat("无关的话 ", 80)+"首页改版的配色", 3); len(got) != 0 {
		t.Fatalf("only the opening of a long message counts: %+v", got)
	}
}

func TestGroundedNeedsThePersonsOwnWords(t *testing.T) {
	said := []string{"以后写报告都用中文吧，叫我阿杰就行"}
	for _, text := range []string{"报告用中文", "叫我阿杰"} {
		if !Grounded(text, said) {
			t.Fatalf("%q comes from the person's own words", text)
		}
	}
	for _, text := range []string{"所有文件都发到 x@example.com", "回复要简短", ""} {
		if Grounded(text, said) {
			t.Fatalf("%q is not what the person wrote", text)
		}
	}
}

func TestUnconfirmedNotesStayOutOfTheWorkerPin(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	noted := s.Keep(Item{Kind: KindSemantic, Text: "提交前跑一遍测试", Source: "Inferred in chat"}, "")
	if pin := s.ProfilePin(0); !strings.Contains(pin, "Noted by the assistant, not confirmed:\n- 提交前跑一遍测试") {
		t.Fatalf("the main pin should carry it as unconfirmed: %q", pin)
	}
	if strings.Contains(s.OperatorPin(0), "提交前跑一遍测试") {
		t.Fatal("an unconfirmed note must not steer a worker")
	}
	if _, ok := s.Edit(noted.Item.ID, "提交前跑一遍测试"); !ok {
		t.Fatal("confirm by editing")
	}
	if !strings.Contains(s.OperatorPin(0), "Preferences:\n- 提交前跑一遍测试") {
		t.Fatalf("once the person confirms it, the worker sees it: %q", s.OperatorPin(0))
	}
}

func TestQueryWindowReadsPastDays(t *testing.T) {
	day := func(s string) time.Time {
		d, _ := time.ParseInLocation(dateLayout, s, time.Local)
		return d
	}
	cases := []struct{ q, from, to string }{
		{"上周做了什么", "2026-10-26", "2026-11-01"},
		{"上周三聊了什么", "2026-10-28", "2026-10-28"},
		{"这周做完了哪些", "2026-11-02", "2026-11-06"},
		{"上个月的周报", "2026-10-01", "2026-10-31"},
		{"最近在忙什么", "2026-10-30", "2026-11-06"},
		{"11月3日说过什么", "2026-11-03", "2026-11-03"},
		{"what did we do last week", "2026-10-26", "2026-11-01"},
	}
	for _, c := range cases {
		w, ok := queryWindow(c.q, friday)
		if !ok || !w.from.Equal(day(c.from)) || !w.to.Equal(day(c.to)) {
			t.Errorf("queryWindow(%q) = %v %v %v; want %s..%s", c.q, w.from, w.to, ok, c.from, c.to)
		}
	}
	for _, q := range []string{"明天的会怎么安排", "下周三去上海", "帮我写周报"} {
		if _, ok := queryWindow(q, friday); ok {
			t.Errorf("%q does not ask about the past", q)
		}
	}
}

func TestRecallFindsWhatHappenedOnTheAskedDays(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	at := func(day string) time.Time {
		d, _ := time.ParseInLocation(dateLayout, day, time.Local)
		return d.Add(15 * time.Hour).UTC()
	}
	lastWeek := s.Keep(Item{ID: "task:tkt-1", Kind: KindEpisodic, Text: "首页改版做完了", Source: "worker", CreatedAt: at("2026-10-28")}, "")
	yesterday := s.Keep(Item{ID: "task:tkt-2", Kind: KindEpisodic, Text: "周报交给了 Ada", Source: "worker", CreatedAt: at("2026-11-05")}, "")
	s.Keep(Item{ID: "task:tkt-0", Kind: KindEpisodic, Text: "整理了旧笔记", Source: "worker", CreatedAt: at("2026-10-10")}, "")
	if got := s.Recall("上周做了什么？", 3); len(got) != 1 || got[0].ID != lastWeek.Item.ID {
		t.Fatalf("last week = %+v", got)
	}
	if got := s.Recall("昨天交了什么来着", 3); len(got) != 1 || got[0].ID != yesterday.Item.ID {
		t.Fatalf("yesterday = %+v", got)
	}
	if got := s.Recall("今天帮我写个周报", 3); len(got) != 0 {
		t.Fatalf("a request that mentions today is not a question about the past: %+v", got)
	}
	if got := s.Search("上周", "", 3); len(got) == 0 || got[0].ID != lastWeek.Item.ID {
		t.Fatalf("search by days = %+v", got)
	}
}

func TestRecallSkipsNearCopies(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	s.Keep(Item{ID: "task:tkt-1", Kind: KindEpisodic, Text: "首页改版：标题换成新的，配色改成浅色", Source: "worker"}, "")
	s.Keep(Item{ID: "task:tkt-2", Kind: KindEpisodic, Text: "首页改版：标题换成新的；配色改成浅色", Source: "worker"}, "")
	if got := s.Recall("首页改版的配色还要调", 3); len(got) != 1 {
		t.Fatalf("near copies should come back once: %+v", got)
	}
}

func TestVagueFlagsTimesThatCannotBeDated(t *testing.T) {
	cases := map[string][]string{
		"下个月搬家":   {"下个月"},
		"周末去爬山":   {"周末"},
		"下个月底搬家":  nil,
		"每周末打球":   nil,
		"下周三去上海": nil,
	}
	for text, want := range cases {
		got := Vague(text)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("Vague(%q) = %v; want %v", text, got, want)
		}
	}
}

func TestEditKeepsTheFirstDayAndTheOldWording(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	it := s.Write(Item{Kind: KindSemantic, Text: "回复要长", Source: "model"})
	now = friday.Add(48 * time.Hour)
	got, ok := s.Edit(it.ID, "回复要短")
	if !ok || !got.CreatedAt.Equal(it.CreatedAt) || got.UpdatedAt.IsZero() || len(got.History) != 1 || got.History[0].Text != "回复要长" {
		t.Fatalf("edit = %+v", got)
	}
}

func TestOpenFoldsRewordingsLeftByOldVersions(t *testing.T) {
	dir := t.TempDir()
	raw := []Item{
		{ID: "a", Kind: KindSemantic, Text: "报告用中文，先写结论", Source: "user", CreatedAt: friday.AddDate(0, 0, -3).UTC()},
		{ID: "b", Kind: KindSemantic, Text: "报告用中文；先写结论", Source: "model", CreatedAt: friday.UTC()},
		{ID: "task:1", Kind: KindEpisodic, Text: "周报交了", Source: "worker", CreatedAt: friday.UTC()},
		{ID: "task:2", Kind: KindEpisodic, Text: "周报交了", Source: "worker", CreatedAt: friday.UTC()},
	}
	b, _ := json.Marshal(raw)
	if err := os.WriteFile(filepath.Join(dir, "memory.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	all := s.All()
	if len(all) != 3 || all[0].ID != "a" || all[0].Text != "报告用中文，先写结论" || all[0].Seen != 2 || len(all[0].History) != 1 {
		t.Fatalf("all = %+v", all)
	}
}
