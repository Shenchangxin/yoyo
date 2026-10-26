// 任务、目标和提案的类型与状态常量。
// ProposalPageSave 与 ProposalPageEdit 只是种类字符串，Propose 不接受它们。页面库在 pages 包。

package personal

import "time"

// Status 是任务状态字符串。
type Status string

const (
	// StatusQueued 是排队，值为 queued。
	StatusQueued Status = "queued"
	// StatusRunning 是执行中，值为 running。
	StatusRunning Status = "running"
	// StatusWaitingInput 是等待用户输入，值为 waiting_input。
	StatusWaitingInput Status = "waiting_input"
	// StatusWaitingApproval 是等待批准提案，值为 waiting_approval。
	StatusWaitingApproval Status = "waiting_approval"
	// StatusScheduled 是等到 NextRunAt 再跑，值为 scheduled。
	StatusScheduled Status = "scheduled"
	// StatusPaused 是暂停，值为 paused。
	StatusPaused Status = "paused"
	// StatusSucceeded 是成功，值为 succeeded。
	StatusSucceeded Status = "succeeded"
	// StatusFailed 是失败，值为 failed。
	StatusFailed Status = "failed"
	// StatusCancelled 是取消，值为 cancelled。
	StatusCancelled Status = "cancelled"
)

// Kind 是任务种类。
type Kind string

const (
	// KindAgent 是对话里完成的任务，值为 agent。worker 不领取这种任务。
	KindAgent Kind = "agent"
	// KindDocument 是填表并准备回信，值为 document。
	KindDocument Kind = "document"
	// KindMonitor 是按 URL 监视，值为 monitor。
	KindMonitor Kind = "monitor"
	// KindFinance 是汇总交易 CSV，值为 finance。
	KindFinance Kind = "finance"
	// KindPlan 是给目标写里程碑，值为 plan。
	KindPlan Kind = "plan"
)

// ProposalKind 是待审阅动作的种类。
type ProposalKind string

const (
	// ProposalEmailSend 是发信，值为 email.send。
	ProposalEmailSend ProposalKind = "email.send"
	// ProposalCalendarCreate 是创建日程，值为 calendar.create。
	ProposalCalendarCreate ProposalKind = "calendar.create"
	// ProposalCalendarUpdate 是更新日程，值为 calendar.update。
	ProposalCalendarUpdate ProposalKind = "calendar.update"
	// ProposalCalendarDelete 是删除日程，值为 calendar.delete。
	ProposalCalendarDelete ProposalKind = "calendar.delete"
	// ProposalPageSave 是字符串 page.save。Propose 的 switch 不接受它。
	ProposalPageSave ProposalKind = "page.save"
	// ProposalPageEdit 是字符串 page.edit。Propose 的 switch 不接受它。
	ProposalPageEdit ProposalKind = "page.edit"
)

// ProposalStatus 是提案状态。
type ProposalStatus string

const (
	// ProposalAwaiting 是等待审阅，值为 awaiting_review。
	ProposalAwaiting ProposalStatus = "awaiting_review"
	// ProposalExecuting 是正在执行，值为 executing。
	ProposalExecuting ProposalStatus = "executing"
	// ProposalSucceeded 是执行成功，值为 succeeded。
	ProposalSucceeded ProposalStatus = "succeeded"
	// ProposalFailed 是执行失败，值为 failed。
	ProposalFailed ProposalStatus = "failed"
	// ProposalOutcomeUnknown 是结果未知，值为 outcome_unknown。Retry 会拒绝这种关联任务。
	ProposalOutcomeUnknown ProposalStatus = "outcome_unknown"
	// ProposalDenied 是被拒绝，值为 denied。
	ProposalDenied ProposalStatus = "denied"
	// ProposalCancelled 是取消，值为 cancelled。
	ProposalCancelled ProposalStatus = "cancelled"
	// ProposalExpired 是过期，值为 expired。
	ProposalExpired ProposalStatus = "expired"
)

// Evidence 是一条可引用的邮件、网页或用户原文摘录。
type Evidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	URL     string `json:"url,omitempty"`
}

// Step 是任务计划里的一步。
type Step struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Task 是一条个人任务，含种类、状态、输入和租约。
type Task struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Prompt      string         `json:"prompt"`
	Kind        Kind           `json:"kind"`
	Status      Status         `json:"status"`
	GoalID      string         `json:"goal_id,omitempty"`
	MilestoneID string         `json:"milestone_id,omitempty"`
	Plan        []Step         `json:"plan,omitempty"`
	Evidence    []Evidence     `json:"evidence,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
	State       map[string]any `json:"state,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	NextRunAt   *time.Time     `json:"next_run_at,omitempty"`
	LeaseID     string         `json:"lease_id,omitempty"`
	LeaseUntil  *time.Time     `json:"lease_until,omitempty"`
	Attempts    int            `json:"attempts"`
	ActionID    string         `json:"action_id,omitempty"`
	Result      string         `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	Question    string         `json:"question,omitempty"`
	ArtifactIDs []string       `json:"artifact_ids,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
}

// Goal 是一个带里程碑的目标。
type Goal struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Category    string      `json:"category,omitempty"`
	Status      string      `json:"status"`
	Milestones  []Milestone `json:"milestones,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

// Milestone 是目标下的一条里程碑。
type Milestone struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Monitor 是一条 URL 监视及其检查间隔。
type Monitor struct {
	ID              string    `json:"id"`
	TaskID          string    `json:"task_id"`
	Title           string    `json:"title"`
	URL             string    `json:"url"`
	Condition       string    `json:"condition"`
	Value           string    `json:"value,omitempty"`
	IntervalMinutes int       `json:"interval_minutes"`
	Status          string    `json:"status"`
	NextCheckAt     time.Time `json:"next_check_at"`
	LastCheckedAt   time.Time `json:"last_checked_at,omitempty"`
	LastValue       string    `json:"last_value,omitempty"`
	LastHash        string    `json:"last_hash,omitempty"`
	Error           string    `json:"error,omitempty"`
	Checks          int       `json:"checks"`
}

// MonitorPage 是某次监视抓到的行和哈希，存在 workspace.json，不是页面库文件。
type MonitorPage struct {
	ID           string   `json:"id"`
	Hash         string   `json:"hash"`
	TimelessHash string   `json:"timeless_hash,omitempty"`
	Lines        []string `json:"lines,omitempty"`
}

// Idea 是一条待接受或已处理的想法。
type Idea struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Reason    string         `json:"reason"`
	Evidence  []Evidence     `json:"evidence,omitempty"`
	Prompt    string         `json:"prompt"`
	Kind      Kind           `json:"kind"`
	Input     map[string]any `json:"input,omitempty"`
	Status    string         `json:"status"`
	TaskID    string         `json:"task_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Artifact 是任务写下的文件或摘要。
type Artifact struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Summary   string         `json:"summary"`
	Path      string         `json:"path,omitempty"`
	Parent    string         `json:"parent,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Activity 是提案上的一条状态记录。
type Activity struct {
	At     time.Time `json:"at"`
	Status string    `json:"status"`
	Detail string    `json:"detail,omitempty"`
}

// Proposal 是待审阅或已执行的邮件、日历动作。Hash 覆盖种类、数据和账号。
type Proposal struct {
	ID            string         `json:"id"`
	TaskID        string         `json:"task_id,omitempty"`
	Account       string         `json:"account,omitempty"`
	Title         string         `json:"title"`
	Kind          ProposalKind   `json:"kind"`
	Data          map[string]any `json:"data"`
	Status        ProposalStatus `json:"status"`
	Hash          string         `json:"hash"`
	TargetVersion string         `json:"target_version,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Result        string         `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
	Source        string         `json:"source,omitempty"`
	Activity      []Activity     `json:"activity,omitempty"`
}

// Choice 是一次让用户点选的提问。
type Choice struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id,omitempty"`
	Title     string         `json:"title"`
	Options   []ChoiceOption `json:"options"`
	Selected  string         `json:"selected,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ChoiceOption 是一个选项。Source 非空时 PresentChoices 把整组当成比较并最多留 3 项。
type ChoiceOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Source string `json:"source,omitempty"`
}

// MailView 是从连接器读到的一封邮件的视图，不单独落盘。
type MailView struct {
	ID          string
	ThreadID    string
	From        string
	Sender      string
	Subject     string
	Body        string
	Label       string
	Attachments []string
}

// Snapshot 是某一刻的任务、目标、监视、想法、产物、提案、选择和 worker 状态。
type Snapshot struct {
	Tasks     []Task       `json:"tasks"`
	Goals     []Goal       `json:"goals"`
	Monitors  []Monitor    `json:"monitors"`
	Ideas     []Idea       `json:"ideas"`
	Artifacts []Artifact   `json:"artifacts"`
	Proposals []Proposal   `json:"proposals"`
	Choices   []Choice     `json:"choices"`
	Worker    WorkerStatus `json:"worker"`
	LastIdeas time.Time    `json:"last_ideas_at,omitempty"`
}

// WorkerStatus 表示轮询循环是否在跑，以及上次 tick 的时间。
type WorkerStatus struct {
	Running    bool      `json:"running"`
	LastTickAt time.Time `json:"last_tick_at,omitempty"`
}

// doc 是 workspace.json 的整份文档，含监视页快照和样例页。
type doc struct {
	Tasks       []Task            `json:"tasks"`
	Goals       []Goal            `json:"goals"`
	Monitors    []Monitor         `json:"monitors"`
	Ideas       []Idea            `json:"ideas"`
	Artifacts   []Artifact        `json:"artifacts"`
	Proposals   []Proposal        `json:"proposals"`
	Choices     []Choice          `json:"choices"`
	Pages       []MonitorPage     `json:"pages,omitempty"`
	SamplePages map[string]string `json:"sample_pages,omitempty"`
	LastIdeasAt time.Time         `json:"last_ideas_at,omitempty"`
	LastTickAt  time.Time         `json:"last_tick_at,omitempty"`
}
