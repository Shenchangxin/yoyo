// model.go 放领域类型。规则在 Engine 里，不在这些结构的方法里。

package superagent

import "time"

// Trigger 是一次不可变事件。ID 相同的第二次接收会被当成重复。
type Trigger struct {
	ID           string
	Source       string
	Threads      []string
	Primary      string
	Human        bool
	Addressed    bool
	Sender       string
	Text         string
	Channel      string
	Ref          string
	Reached      []string
	Handbacks    []string
	Requeued     bool
	DecisionID   string
	Queued       bool
	Filtered     string
	StrengthHold string
	// At 是这条触发到达的时刻。旧快照没有这一项时为零，不计入最近一天。
	At time.Time `json:"At,omitempty"`
	// Due 是日历上的开始时间。空着表示没有到期点。
	Due string `json:"Due,omitempty"`
	// Agents 是人在这条消息里 @ 过、确认存在的 agent 编号，按出现顺序。路由只能交给它们或 yoyo。
	Agents []string `json:"Agents,omitempty"`
}

// Thread 按时间收触发。Subs 是订阅它的工单。Queued 是还没评估的触发。
// Evaluating 非空表示这一批评触发正在评估。Expired 是连续过期后重新排队的次数。
type Thread struct {
	Key        string
	Subs       []string
	Context    []string
	Queued     []string
	Evaluating *Evaluation
	Expired    int
	// Watch 是只为了上下文跟随的工单。它们能收到消息，但不能恢复已结束的工单，也不拥有人的后续。
	Watch []string
}

// Evaluation 是一条线程上的一轮评估。Deadline 从交给编排台的时刻起算。
// Delivered 为零表示还没交到评估者，过期逻辑不会把它判成超时。PausedAt 非零表示暂停开始的时刻。
type Evaluation struct {
	ID            string
	TriggerIDs    []string
	Primary       string
	Strength      string
	Deadline      time.Time
	Delivered     time.Time
	PausedAt      time.Time
	PolicyVersion int
}

// Mandate 是接受时授予的范围。Version 是当时的策略版本。Strength 为 explicit 或 inferred。
type Mandate struct {
	Version  int
	Strength string
	Sender   string
	Rules    []string
}

// Report 是派工报给工单的状态。State 取 progress、waiting、ready、blocked、failed、dropped。
type Report struct {
	State    string
	Summary  string
	Question string
	Seq      int
	// At 是这份报告记下的时刻。旧快照没有这一项时为零。
	At time.Time `json:"At,omitempty"`
	// Need 是人要先做的事。login 表示 agent 要先在 Buddy 登录。
	Need string `json:"Need,omitempty"`
}

// Assignment 把工单绑到一次执行。State 取 pending、running、ended。
type Assignment struct {
	ID      string
	State   string
	Reports []Report
	// Agent 是这次派工交给的 agent。空串表示沿用工单的 Agent。
	Agent string `json:"Agent,omitempty"`
	// From 是改派前做这张工单的 agent。非空表示这次派工是交接，工人日志从这里另起一段。
	From string `json:"From,omitempty"`
}

// Usage 是外部 agent 报的上下文用量和花费。Used 和 Size 是 token 数，Cost 按 Currency 计。
type Usage struct {
	Used     int     `json:"Used,omitempty"`
	Size     int     `json:"Size,omitempty"`
	Cost     float64 `json:"Cost,omitempty"`
	Currency string  `json:"Currency,omitempty"`
}

// Input 是投递给工单的上下文。Human 为真表示人写的。
type Input struct {
	ID        string
	TriggerID string
	Human     bool
	Text      string
	Seq       int
}

// Close 是一条线程的关闭事实。At 用来比较先后。Outcome 为 merged 或 closed。
type Close struct {
	At      string
	Outcome string
}

// Follow 是工单跟着的线程。Own 为真表示工单自己打开的。Closed 非空表示这条线程已经关闭。
type Follow struct {
	Thread string
	Own    bool
	Closed *Close
}

// End 是工单结束记录。By 为 executor 或 human。Outcome 为 completed、dropped、stopped、merged、closed。
// At 是结束时刻。旧快照没有这一项时为零，订阅线程不再为了它多读两周。
type End struct {
	By      string
	Outcome string
	Reason  string
	Seq     int
	At      time.Time
}

// Ticket 是一份已授权的工作。Ended 为 nil 表示还开着。
type Ticket struct {
	ID          string
	Mandate     Mandate
	Origin      string
	Assignments []Assignment
	Inputs      []Input
	Follows     []Follow
	Attention   string
	DeliveryErr string
	Undelivered []string
	Ended       *End
	// Directory 是这张工单的工作目录。Name 是短名，准备命令和重试沿用它。
	Directory string
	Name      string
	// Resume 为真时，下一次启动只接上已有日志，不再把工单说明重发一遍。
	Resume bool
	// Opened 是接单的时刻。旧快照没有这一项时为零，快照里用起点触发的到达时刻补上。
	Opened time.Time `json:"Opened,omitempty"`
	// Agent 是做这张工单的 agent。空串和 yoyo 都是 yoyo 自带工人。
	Agent string `json:"Agent,omitempty"`
	// AgentPinned 为真表示人用 @ 点名了 Agent，离线时不擅自换人。
	AgentPinned bool `json:"AgentPinned,omitempty"`
	// Pointer 是外部 agent 的会话编号，重启或重连后用它续接。PointerOf 记它属于哪个 agent，换了人就不再用。
	Pointer   string `json:"Pointer,omitempty"`
	PointerOf string `json:"PointerOf,omitempty"`
	// Usage 是外部 agent 最近一次报的用量。
	Usage *Usage `json:"Usage,omitempty"`
}

// Decision 是一次评估结果。Kind 为 accept、stop、ignore、unavailable。
type Decision struct {
	ID        string
	TriggerID string
	Thread    string
	Kind      string
	Reason    string
	TicketID  string
	Dismissed bool
	At        time.Time
}

// Outbound 是准备发回渠道的一条消息。
type Outbound struct {
	ID       string
	TicketID string
	Kind     string
	Text     string
	Status   string
	Source   string
	Ref      string
	Key      string
	// Attempts 是发送失败的次数。同一对话里前面一条失败三次后，后面的才继续发。
	Attempts int
}

// Error 是人能造成的拒绝。Code 是稳定短码。
type Error struct {
	Code string
}

// Error 返回 Code。
func (e Error) Error() string { return e.Code }

// errCode 用 code 造一个 Error。code 会原样成为这条错误的文本。
func errCode(code string) error { return Error{Code: code} }
