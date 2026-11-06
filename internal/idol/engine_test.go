// 本文件锁住引擎和存储的整轮行为：流式吐段、被打断时只记写完的段、照片在回复之后挂上、告别和挂断、
// 提示的角色交替和礼物旁白、写时复制、坏文件恢复、删除偶像连带清理、图片压缩、先开口和回应礼物。由 go test 调用。

package idol

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// scriptBrain 按 kind 返回脚本里的文字。chat 时逐字流式吐出；holdAfter 非空时吐完这段就等到 ctx 取消。
type scriptBrain struct {
	mu         sync.Mutex
	judge      []string
	chat       string
	holdAfter  string
	reach      string
	title      string
	titleCalls int
	paint      []byte
	prompts    [][]Msg
}

func (b *scriptBrain) Chat(ctx context.Context, kind, _ string, msgs []Msg) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch kind {
	case "judge":
		if len(b.judge) == 0 {
			return `{"level":2}`, nil
		}
		v := b.judge[0]
		b.judge = b.judge[1:]
		return v, nil
	case "reach":
		b.prompts = append(b.prompts, msgs)
		return b.reach, nil
	case "memory":
		return `{"memories":[]}`, nil
	case "title":
		b.titleCalls++
		return b.title, nil
	}
	b.prompts = append(b.prompts, msgs)
	return b.chat, nil
}

func (b *scriptBrain) ChatStream(ctx context.Context, kind, key string, msgs []Msg, onDelta func(string)) (string, error) {
	if kind != "chat" {
		return b.Chat(ctx, kind, key, msgs)
	}
	b.mu.Lock()
	b.prompts = append(b.prompts, msgs)
	text, hold := b.chat, b.holdAfter
	b.mu.Unlock()
	for _, r := range text {
		onDelta(string(r))
	}
	if hold != "" {
		for _, r := range hold {
			onDelta(string(r))
		}
		<-ctx.Done()
		return text + hold, ctx.Err()
	}
	return text, nil
}

func (b *scriptBrain) Paint(context.Context, string) ([]byte, error) {
	if b.paint == nil {
		return nil, errf("no image")
	}
	return b.paint, nil
}

// newEngine 开一个临时库，建一位说中文的偶像，返回引擎和她的 id。
func newEngine(t *testing.T, brain Brain) (*Engine, string) {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	saved, err := st.SaveIdol(Idol{Name: "小灯", SpokenLang: "zh", SubtitleLang: "zh", Sheet: "短句。"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return &Engine{Store: st, Brain: brain}, saved.ID
}

// TestTurnStreamsSegments 锁住流式回合依次吐出写完的段，评判之后先报情绪，记下的段和吐出的一致。
func TestTurnStreamsSegments(t *testing.T) {
	brain := &scriptBrain{judge: []string{`{"level":2,"emotion":"shy"}`}, chat: "<zh>[shyly] 嗯……</zh><zh>好呀。</zh>"}
	eng, id := newEngine(t, brain)
	var segs []Segment
	mood := ""
	res, err := eng.Turn(context.Background(), TurnInput{
		IdolID: id, Text: "周末一起看电影吗", Call: true, CallID: "call-1",
		OnJudged:  func(e string) { mood = e },
		OnSegment: func(i int, s Segment) { segs = append(segs, s) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if mood != "shy" || len(segs) != 2 || res.Emotion != "shy" || res.Segments[1].Spoken != "好呀。" {
		t.Fatalf("mood %q segs %+v res %+v", mood, segs, res)
	}
	rec := eng.Store.View().History(id).Current.Records[0]
	if rec.Call != "call-1" || len(rec.Segments) != 2 {
		t.Fatalf("record %+v", rec)
	}
}

// TestTurnStopKeepsCompleteSegments 锁住被叫停时只记下已经写完的段；一段都没写完时什么也不记。
func TestTurnStopKeepsCompleteSegments(t *testing.T) {
	brain := &scriptBrain{chat: "<zh>第一句。</zh>", holdAfter: "<zh>第二"}
	eng, id := newEngine(t, brain)
	stop := make(chan struct{})
	res, err := eng.Turn(context.Background(), TurnInput{
		IdolID: id, Text: "讲个故事", Stop: stop,
		OnSegment: func(i int, s Segment) { close(stop) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Stopped || len(res.Segments) != 1 {
		t.Fatalf("%+v", res)
	}
	brain.chat, brain.holdAfter = "", "<zh>还没"
	stop2 := make(chan struct{})
	go func() { time.Sleep(50 * time.Millisecond); close(stop2) }()
	if _, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "再讲一个", Stop: stop2}); err != ErrStopped {
		t.Fatalf("want ErrStopped, got %v", err)
	}
	if n := len(eng.Store.View().History(id).Current.Records); n != 1 {
		t.Fatalf("records %d", n)
	}
}

// TestTurnPhotoAfterReply 锁住她写了照片描述时照片在回复记下之后挂上，压过的图存进 blob；没配生图时提示她发不了。
func TestTurnPhotoAfterReply(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	seed := uint32(7)
	for i := range img.Pix {
		seed = seed*1664525 + 1013904223
		img.Pix[i] = byte(seed >> 24)
		if i%4 == 3 {
			img.Pix[i] = 0xff
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	if buf.Len() <= keepAsIs {
		t.Fatalf("test image too small: %d", buf.Len())
	}
	brain := &scriptBrain{judge: []string{`{"level":3,"photo":true}`}, chat: "<photo>窗边，穿着毛衣</photo><zh>给你。</zh>", paint: buf.Bytes()}
	eng, id := newEngine(t, brain)
	res, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "发张自拍给我"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.PhotoPending {
		t.Fatalf("%+v", res)
	}
	photo, state, ok := eng.Store.PhotoState(res.RecordID)
	if !ok || photo == "" || state != "" {
		t.Fatalf("photo %q state %q", photo, state)
	}
	b, err := eng.Store.ReadBlob(photo)
	if err != nil || len(b) >= buf.Len() {
		t.Fatalf("blob %d of %d, %v", len(b), buf.Len(), err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width > photoMaxSide {
		t.Fatalf("shrunk %+v %v", cfg, err)
	}
	eng.CanPaint = func() bool { return false }
	brain.chat = "<zh>现在不方便。</zh>"
	if _, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "再来一张"}); err != nil {
		t.Fatal(err)
	}
	last := brain.prompts[len(brain.prompts)-1]
	if !strings.Contains(last[len(last)-2].Content, "发不了照片") {
		t.Fatalf("cannot send missing: %q", last[len(last)-2].Content)
	}
}

// TestTurnEndsAndHangsUp 锁住打字时第二次越界后她结束这段对话，之后的话被拒绝；通话里第二次越界后她挂断，对话不结束。
func TestTurnEndsAndHangsUp(t *testing.T) {
	rude := `{"level":0,"out_of_line":true,"emotion":"angry"}`
	brain := &scriptBrain{judge: []string{rude, rude}, chat: "<zh>请不要这样说。</zh>"}
	eng, id := newEngine(t, brain)
	first, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "滚"})
	if err != nil || first.Ended {
		t.Fatalf("first %+v %v", first, err)
	}
	second, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "滚开"})
	if err != nil || !second.Ended || second.EndReason != "out_of_line" {
		t.Fatalf("second %+v %v", second, err)
	}
	prompt := brain.prompts[len(brain.prompts)-1]
	if !strings.Contains(prompt[len(prompt)-2].Content, "不想再和用户说话") {
		t.Fatal("she was not told to part")
	}
	if _, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "对不起"}); err == nil || err.Error() != "ended" {
		t.Fatalf("want ended, got %v", err)
	}

	brain2 := &scriptBrain{judge: []string{rude, rude}, chat: "<zh>我挂了。</zh>"}
	eng2, id2 := newEngine(t, brain2)
	if _, err := eng2.Turn(context.Background(), TurnInput{IdolID: id2, Text: "滚", Call: true, CallID: "c"}); err != nil {
		t.Fatal(err)
	}
	res, err := eng2.Turn(context.Background(), TurnInput{IdolID: id2, Text: "滚开", Call: true, CallID: "c"})
	if err != nil || !res.HungUp || res.Ended {
		t.Fatalf("hang up %+v %v", res, err)
	}
	if eng2.Store.View().History(id2).Current.Ended {
		t.Fatal("a call hang-up does not end the conversation")
	}
}

// TestPromptShapeAndGiftNarration 锁住提示的形状：两条稳定系统消息、一问一答交替、每轮会变的一节在最后一句之前；
// 礼物写成旁白折进下一条用户消息，带名字和她喜欢，并只提醒一次要回应。
func TestPromptShapeAndGiftNarration(t *testing.T) {
	brain := &scriptBrain{chat: "<zh>好。</zh>"}
	eng, id := newEngine(t, brain)
	ctx := context.Background()
	if _, err := eng.Turn(ctx, TurnInput{IdolID: id, Text: "你好"}); err != nil {
		t.Fatal(err)
	}
	_ = eng.Store.Update(func(st *State) error {
		it, _ := st.idol(id)
		it.LikedGifts = []string{"tea"}
		st.putIdol(it)
		st.addStock("tea", 1)
		_, _, _, err := st.Give(id, "tea", time.Now())
		return err
	})
	if _, err := eng.Turn(ctx, TurnInput{IdolID: id, Text: "喜欢吗"}); err != nil {
		t.Fatal(err)
	}
	msgs := brain.prompts[len(brain.prompts)-1]
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.Role
	}
	if got := strings.Join(roles, ","); got != "system,system,user,assistant,system,user" {
		t.Fatalf("roles %s", got)
	}
	last := msgs[len(msgs)-1].Content
	if !strings.Contains(last, "热茶") || !strings.Contains(last, "这是你喜欢的东西") || !strings.Contains(last, respondToGifts) || !strings.HasSuffix(last, "喜欢吗") {
		t.Fatalf("narration %q", last)
	}
	if !strings.Contains(msgs[len(msgs)-2].Content, "# 此刻") || !strings.Contains(msgs[0].Content, "# 回复格式") {
		t.Fatal("sections")
	}
}

// TestTitleOnlyAtThirdTurn 锁住标题只在第 3 轮试一次：那次给了空标题，之后几轮也不再多花调用去试。
func TestTitleOnlyAtThirdTurn(t *testing.T) {
	brain := &scriptBrain{chat: "<zh>嗯。</zh>", title: ""}
	eng, id := newEngine(t, brain)
	for i := 0; i < 5; i++ {
		if _, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "第几句"}); err != nil {
			t.Fatal(err)
		}
	}
	if brain.titleCalls != 1 || eng.Store.View().History(id).Current.Title != "" {
		t.Fatalf("title calls %d", brain.titleCalls)
	}
	brain2 := &scriptBrain{chat: "<zh>嗯。</zh>", title: "「周末计划」"}
	eng2, id2 := newEngine(t, brain2)
	for i := 0; i < 3; i++ {
		if _, err := eng2.Turn(context.Background(), TurnInput{IdolID: id2, Text: "周末去哪"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := eng2.Store.View().History(id2).Current.Title; got != "周末计划" {
		t.Fatalf("title %q", got)
	}
}

// TestUpdateIsCopyOnWrite 锁住回调返回错误时库里的状态不变。
func TestUpdateIsCopyOnWrite(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := len(st.View().Idols)
	_ = st.Update(func(s *State) error {
		s.Idols = nil
		s.Draws = 99
		return errf("nope")
	})
	view := st.View()
	if len(view.Idols) != before || view.Draws == 99 {
		t.Fatalf("mutated on failure: %d idols, %d draws", len(view.Idols), view.Draws)
	}
}

// TestOpenRecoversBrokenState 锁住 state.json 坏了时留底并从临时文件恢复，而不是让偶像模块打不开。
func TestOpenRecoversBrokenState(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json.tmp"), []byte(`{"draws":4,"preset_rev":99,"affinity_rev":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.View().Draws != 4 {
		t.Fatalf("draws %d", st.View().Draws)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "state.json.broken-*"))
	if len(matches) != 1 {
		t.Fatalf("backup %v", matches)
	}
}

// TestDeleteIdolCleansUp 锁住删除偶像时她的对话、记忆、好感一起删掉，桌面改放桌宠，别人的记忆被她取代的那条重新成立。
func TestDeleteIdolCleansUp(t *testing.T) {
	brain := &scriptBrain{chat: "<zh>嗯。</zh>"}
	eng, id := newEngine(t, brain)
	if _, err := eng.Turn(context.Background(), TurnInput{IdolID: id, Text: "你好"}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Store.Pin(id); err != nil {
		t.Fatal(err)
	}
	_ = eng.Store.Update(func(st *State) error {
		st.Memories = append(st.Memories,
			Memory{ID: "old", Text: "用户住在大阪", SourceIdol: petIdolID, SupersededBy: "new"},
			Memory{ID: "new", Text: "用户搬到了东京", SourceIdol: id})
		return nil
	})
	if err := eng.Store.DeleteIdol(id); err != nil {
		t.Fatal(err)
	}
	st := eng.Store.View()
	if _, ok := st.Idol(id); ok || len(st.History(id).Past) != 0 || st.History(id).Current != nil {
		t.Fatal("idol or conversations left")
	}
	if p, ok := st.Pinned(); !ok || p.ID != petIdolID {
		t.Fatalf("pinned %+v", p)
	}
	if len(st.Memories) != 1 || st.Memories[0].ID != "old" || st.Memories[0].SupersededBy != "" {
		t.Fatalf("memories %+v", st.Memories)
	}
	if err := eng.Store.DeleteIdol(id); err == nil {
		t.Fatal("deleting twice should fail")
	}
}

// TestShrinkImage 锁住大图被缩到长边 1024 并重编码，小图原样返回，带透明的存成 PNG。
func TestShrinkImage(t *testing.T) {
	small := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var sb bytes.Buffer
	_ = png.Encode(&sb, small)
	if out, _ := ShrinkImage(sb.Bytes(), 1024); !bytes.Equal(out, sb.Bytes()) {
		t.Fatal("small image changed")
	}
	big := image.NewRGBA(image.Rect(0, 0, 3000, 1500))
	seed := uint32(3)
	for y := 0; y < 1500; y++ {
		for x := 0; x < 3000; x++ {
			seed = seed*1664525 + 1013904223
			big.Set(x, y, color.RGBA{uint8(seed >> 24), uint8(seed >> 16), uint8(x), 128})
		}
	}
	var bb bytes.Buffer
	_ = png.Encode(&bb, big)
	out, err := ShrinkImage(bb.Bytes(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 1024 || cfg.Height != 512 || format != "png" {
		t.Fatalf("%+v %s %v", cfg, format, err)
	}
}

// TestReachAndReact 锁住到时间时她先开口记成 reach，提示里写你们没在说话并带先开口的规则；送礼后她回应一次，记成 reply。
func TestReachAndReact(t *testing.T) {
	brain := &scriptBrain{chat: "<zh>好。</zh>", reach: "<zh>[gently] 上次说的考试怎么样了？</zh>"}
	eng, id := newEngine(t, brain)
	ctx := context.Background()
	talked := time.Now().Add(-50 * time.Hour)
	if _, err := eng.Turn(ctx, TurnInput{IdolID: id, Text: "我后天考试", Now: talked}); err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	if eng.Store.View().User.Night(at) {
		at = at.Add(12 * time.Hour)
	}
	res, err := eng.Reach(ctx, id, at)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Caption, "考试") || res.Emotion != "calm" {
		t.Fatalf("%+v", res)
	}
	prompt := brain.prompts[len(brain.prompts)-1]
	if !strings.Contains(prompt[len(prompt)-2].Content, "# 先开口") || !strings.Contains(prompt[len(prompt)-2].Content, "你们这会儿没在说话") {
		t.Fatal("reach prompt")
	}
	if _, err := eng.Reach(ctx, id, at); err == nil {
		t.Fatal("reaches once until answered")
	}
	if _, err := eng.React(ctx, id, at); err == nil {
		t.Fatal("nothing to answer yet")
	}
	_ = eng.Store.Update(func(st *State) error {
		st.addStock("flower", 1)
		_, _, _, err := st.Give(id, "flower", at)
		return err
	})
	brain.reach = "<zh>花好漂亮，谢谢。</zh>"
	if _, err := eng.React(ctx, id, at); err != nil {
		t.Fatal(err)
	}
	if eng.Store.View().GiftUnanswered(id) {
		t.Fatal("answered")
	}
}
