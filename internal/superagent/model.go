// model.go 放领域类型。规则在 Engine 里，不在这些结构的方法里。

package superagent

// Trigger 是一次不可变事件。ID 相同的第二次接收会被当成重复。
type Trigger struct {
	ID         string
	Source     string
	Threads    []string
	Primary    string
	Human      bool
	Addressed  bool
	Sender     string
	Text       string
	Reached    []string
	Handbacks  []string
	Requeued   bool
	DecisionID string
	Queued     bool
	Filtered   string
}

// Thread 按时间收触发。Subs 是订阅它的工单。Queued 是还没评估的触发。
type Thread struct {
	Key     string
	Subs    []string
	Context []string
	Queued  []string
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
}

// Assignment 把工单绑到一次执行。State 取 pending、running、ended。
type Assignment struct {
	ID      string
	State   string
	Reports []Report
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
type End struct {
	By      string
	Outcome string
	Reason  string
	Seq     int
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
}

// Decision 是一次评估结果。Kind 为 accept、stop、ignore、unavailable。
type Decision struct {
	ID        string
	TriggerID string
	Thread    string
	Kind      string
	Reason    string
	TicketID  string
}

// Outbound 是准备发回渠道的一条消息。
type Outbound struct {
	ID       string
	TicketID string
	Kind     string
	Text     string
	Status   string
}

// Error 是人能造成的拒绝。Code 是稳定短码。
type Error struct {
	Code string
}

// Error 返回 Code。
func (e Error) Error() string { return e.Code }

// errCode 用 code 造一个 Error。code 会原样成为这条错误的文本。
func errCode(code string) error { return Error{Code: code} }
