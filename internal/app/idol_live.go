// 偶像回合的进行中状态：一轮在后台跑，界面用 turn_next 长轮询拿写完的段，通话可以一边生成一边念。
// 也管她交给助手的事：后台等那个会话跑完，再让她用自己的话转述，打字和通话都能拿到同一条转述。
// 这些都走 IdolCall 的 op，Wails 绑定、/api/idol 和 RPC 三条路一样能用，不依赖事件总线。

package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
)

const (
	// idolTurnTimeout 是一轮最多跑多久。
	idolTurnTimeout = 150 * time.Second
	// idolWaitCap 是一次长轮询最多等多久。
	idolWaitCap = 25 * time.Second
	// idolLiveKeep 是跑完的回合在表里留多久，供迟到的长轮询取结果。
	idolLiveKeep = 2 * time.Minute
	// idolWorkLimit 是等助手把事办完最多等多久。
	idolWorkLimit = 45 * time.Minute
)

// idolLiveTurn 是一轮正在跑或刚跑完的回合。wake 在每次有新进展时关闭并换新，等待的人醒来再看。
type idolLiveTurn struct {
	id       string
	idolID   string
	mu       sync.Mutex
	wake     chan struct{}
	segs     []idol.Segment
	emotion  string
	photo    bool
	done     bool
	result   *idol.TurnResult
	errText  string
	stop     chan struct{}
	stopOnce sync.Once
	cancel   context.CancelFunc
	finished time.Time
	created  time.Time
}

// update 在锁内改状态并叫醒等待的人。
func (t *idolLiveTurn) update(fn func()) {
	t.mu.Lock()
	fn()
	close(t.wake)
	t.wake = make(chan struct{})
	t.mu.Unlock()
}

// idolTurnView 是 turn_next 返回给界面的样子：after 之后新写完的段、她的情绪、是否在拍照，跑完时带上结果或错误。
type idolTurnView struct {
	ID       string           `json:"id"`
	Segments []idol.Segment   `json:"segments"`
	Emotion  string           `json:"emotion,omitempty"`
	Photo    bool             `json:"photo,omitempty"`
	Done     bool             `json:"done"`
	Result   *idol.TurnResult `json:"result,omitempty"`
	Error    string           `json:"error,omitempty"`
}

// wait 等到 after 之后有新段、回合跑完或 timeout 到期，返回那时的样子。
func (t *idolLiveTurn) wait(after int, timeout time.Duration) idolTurnView {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		t.mu.Lock()
		if len(t.segs) > after || t.done {
			v := t.viewLocked(after)
			t.mu.Unlock()
			return v
		}
		ch := t.wake
		t.mu.Unlock()
		select {
		case <-ch:
		case <-timer.C:
			t.mu.Lock()
			v := t.viewLocked(after)
			t.mu.Unlock()
			return v
		}
	}
}

// viewLocked 在锁内拼出 after 之后的样子。
func (t *idolLiveTurn) viewLocked(after int) idolTurnView {
	if after < 0 {
		after = 0
	}
	v := idolTurnView{ID: t.id, Segments: []idol.Segment{}, Emotion: t.emotion, Photo: t.photo, Done: t.done, Result: t.result, Error: t.errText}
	if after < len(t.segs) {
		v.Segments = append(v.Segments, t.segs[after:]...)
	}
	return v
}

// idolLiveTable 是所有进行中和刚跑完的回合。一个进程只有一个 App，所以放在包里。
var idolLiveTable = struct {
	mu    sync.Mutex
	turns map[string]*idolLiveTurn
}{turns: map[string]*idolLiveTurn{}}

// idolLive 按 id 找一轮。
func idolLive(id string) *idolLiveTurn {
	idolLiveTable.mu.Lock()
	defer idolLiveTable.mu.Unlock()
	return idolLiveTable.turns[id]
}

// cancelIdolTurns 取消还在跑的回合。idolID 为空时取消全部，否则只取消这位偶像的。
func cancelIdolTurns(idolID string) {
	idolLiveTable.mu.Lock()
	defer idolLiveTable.mu.Unlock()
	for _, t := range idolLiveTable.turns {
		if idolID != "" && t.idolID != idolID {
			continue
		}
		t.mu.Lock()
		done := t.done
		t.mu.Unlock()
		if !done {
			t.cancel()
		}
	}
}

const (
	// idolTalkingWindow 是最近开过一轮之后多久内算用户还在和她说话。
	idolTalkingWindow = 10 * time.Minute
	// idolCallMax 是一通电话最多算多久；挂断没报上来（比如窗口崩了）时，过了这么久就不再当她在通话。
	idolCallMax = 3 * time.Hour
)

// idolActivity 记着每位偶像最近一次开回合的时间和正在进行的通话：用户正在和她打字或打电话时，她不先开口。
var idolActivity = struct {
	mu    sync.Mutex
	turns map[string]time.Time
	calls map[string]time.Time
}{turns: map[string]time.Time{}, calls: map[string]time.Time{}}

// noteIdolTurn 记下这位偶像刚开了一轮。
func noteIdolTurn(idolID string) {
	idolActivity.mu.Lock()
	idolActivity.turns[idolID] = time.Now()
	idolActivity.mu.Unlock()
}

// noteIdolCall 记下这位偶像的通话接通（on 为真）或挂断。
func noteIdolCall(idolID string, on bool) {
	idolActivity.mu.Lock()
	defer idolActivity.mu.Unlock()
	if on {
		idolActivity.calls[idolID] = time.Now()
		return
	}
	delete(idolActivity.calls, idolID)
}

// idolTalking 判断用户此刻是否正在和她说话：通话还没挂断，或者最近 idolTalkingWindow 内开过一轮。
func idolTalking(idolID string, now time.Time) bool {
	idolActivity.mu.Lock()
	defer idolActivity.mu.Unlock()
	if at, ok := idolActivity.calls[idolID]; ok && now.Sub(at) < idolCallMax {
		return true
	}
	at, ok := idolActivity.turns[idolID]
	return ok && now.Sub(at) < idolTalkingWindow
}

// openIdolTurn 在后台开一轮。同一位偶像还有没跑完的回合时先取消它：新的一句接替旧的。暂停时拒绝。
func (a *App) openIdolTurn(in idol.TurnInput) (*idolLiveTurn, error) {
	if a.IsPaused() {
		return nil, errors.New("paused")
	}
	cancelIdolTurns(in.IdolID)
	noteIdolTurn(in.IdolID)
	ctx, cancel := context.WithTimeout(context.Background(), idolTurnTimeout)
	t := &idolLiveTurn{
		id: newID(), idolID: in.IdolID, wake: make(chan struct{}), stop: make(chan struct{}),
		cancel: cancel, created: time.Now(),
	}
	now := time.Now()
	idolLiveTable.mu.Lock()
	for id, old := range idolLiveTable.turns {
		old.mu.Lock()
		stale := (old.done && now.Sub(old.finished) > idolLiveKeep) || now.Sub(old.created) > idolTurnTimeout+idolLiveKeep
		old.mu.Unlock()
		if stale {
			delete(idolLiveTable.turns, id)
		}
	}
	idolLiveTable.turns[t.id] = t
	idolLiveTable.mu.Unlock()
	go a.runIdolTurn(ctx, t, in)
	return t, nil
}

// runIdolTurn 跑一轮，把评判、写完的段和拍照交给 t；跑完时她要办事就交给助手，并开始等结果。
func (a *App) runIdolTurn(ctx context.Context, t *idolLiveTurn, in idol.TurnInput) {
	defer t.cancel()
	in.Stop = t.stop
	in.OnJudged = func(e string) { t.update(func() { t.emotion = e }) }
	in.OnSegment = func(_ int, s idol.Segment) { t.update(func() { t.segs = append(t.segs, s) }) }
	in.OnPhoto = func() { t.update(func() { t.photo = true }) }
	res, err := a.idolEngine().Turn(ctx, in)
	if err == nil && res.Work {
		sid, werr := a.startIdolWork(in.IdolID, res.WorkInstruction)
		res.WorkSession = sid
		if werr != nil {
			res.WorkError = werr.Error()
		}
		_ = a.Idols.AttachWork(res.RecordID, sid, res.WorkError)
		if werr == nil && sid != "" {
			a.watchIdolWork(in.IdolID, sid, res.RecordID)
		}
	}
	t.update(func() {
		t.done = true
		t.finished = time.Now()
		switch {
		case err == nil:
			t.result = &res
		case errors.Is(err, idol.ErrStopped):
			t.errText = "stopped"
		case errors.Is(err, context.Canceled):
			t.errText = "canceled"
		case errors.Is(err, context.DeadlineExceeded):
			t.errText = "timed out"
		default:
			t.errText = err.Error()
		}
	})
}

// stopIdolTurn 叫停一轮：不再往下生成，已经写完的段照常记下。
func stopIdolTurn(id string) {
	if t := idolLive(id); t != nil {
		t.stopOnce.Do(func() { close(t.stop) })
	}
}

// idolWaitFor 把界面要求的等待时间夹在 0 到 idolWaitCap 之间，没给时用 idolWaitCap。
func idolWaitFor(ms int) time.Duration {
	if ms <= 0 {
		return idolWaitCap
	}
	d := time.Duration(ms) * time.Millisecond
	if d > idolWaitCap {
		return idolWaitCap
	}
	return d
}

// idolWorkWatch 是一件交给助手的事：done 关闭时 result 或 errText 有值。
type idolWorkWatch struct {
	done    chan struct{}
	result  *idol.TurnResult
	errText string
}

// idolWorkTable 按发起那一轮的记录 id 存着她交出去的事，再按会话记下最近一件，给旧界面只带会话 id 的轮询用。
var idolWorkTable = struct {
	mu      sync.Mutex
	records map[string]*idolWorkWatch
	latest  map[string]*idolWorkWatch
}{records: map[string]*idolWorkWatch{}, latest: map[string]*idolWorkWatch{}}

// idolWork 按记录 id 找一件事，找不到再按会话找最近一件。
func idolWork(recordID, session string) *idolWorkWatch {
	idolWorkTable.mu.Lock()
	defer idolWorkTable.mu.Unlock()
	if w := idolWorkTable.records[recordID]; recordID != "" && w != nil {
		return w
	}
	if session != "" {
		return idolWorkTable.latest[session]
	}
	return nil
}

// watchIdolWork 在后台等会话 sid 跑完，再让她转述结果，写成一条 work_said。会话连续两次查到没在跑才算完，
// 免得排队的下一件刚好在两件之间的空当里被当成办完。返回这件事，同一条记录重复调用时返回已有的那件。
func (a *App) watchIdolWork(idolID, sid, recordID string) *idolWorkWatch {
	idolWorkTable.mu.Lock()
	if w := idolWorkTable.records[recordID]; recordID != "" && w != nil {
		idolWorkTable.mu.Unlock()
		return w
	}
	w := &idolWorkWatch{done: make(chan struct{})}
	if recordID != "" {
		idolWorkTable.records[recordID] = w
	}
	idolWorkTable.latest[sid] = w
	idolWorkTable.mu.Unlock()
	go func() {
		defer close(w.done)
		deadline := time.Now().Add(idolWorkLimit)
		idle := 0
		for idle < 2 && time.Now().Before(deadline) {
			time.Sleep(time.Second)
			if a.Running(sid) {
				idle = 0
			} else {
				idle++
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := a.idolEngine().Narrate(ctx, idolID, a.lastAssistant(sid), time.Now())
		if err != nil {
			w.errText = err.Error()
			return
		}
		w.result = &res
		a.Touch("idol", idolID)
	}()
	return w
}

// waitIdolPhoto 等一条记录的照片拍完或失败，最多 timeout。返回照片 id 和状态。
func (a *App) waitIdolPhoto(recordID string, timeout time.Duration) (string, string) {
	deadline := time.Now().Add(timeout)
	for {
		photo, state, ok := a.Idols.PhotoState(recordID)
		if !ok {
			return "", "missing"
		}
		if state != "pending" || time.Now().After(deadline) {
			return photo, state
		}
		time.Sleep(300 * time.Millisecond)
	}
}
