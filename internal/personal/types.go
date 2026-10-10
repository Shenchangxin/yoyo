package personal

import "time"

type Status string

const (
	StatusQueued          Status = "queued"
	StatusRunning         Status = "running"
	StatusWaitingInput    Status = "waiting_input"
	StatusWaitingApproval Status = "waiting_approval"
	StatusScheduled       Status = "scheduled"
	StatusPaused          Status = "paused"
	StatusSucceeded       Status = "succeeded"
	StatusFailed          Status = "failed"
	StatusCancelled       Status = "cancelled"
)

type Kind string

const (
	KindAgent    Kind = "agent"
	KindDocument Kind = "document"
	KindMonitor  Kind = "monitor"
	KindFinance  Kind = "finance"
	KindPlan     Kind = "plan"
)

type ProposalKind string

const (
	ProposalEmailSend      ProposalKind = "email.send"
	ProposalCalendarCreate ProposalKind = "calendar.create"
	ProposalCalendarUpdate ProposalKind = "calendar.update"
	ProposalCalendarDelete ProposalKind = "calendar.delete"
	ProposalPageSave       ProposalKind = "page.save"
	ProposalPageEdit       ProposalKind = "page.edit"
)

type ProposalStatus string

const (
	ProposalAwaiting       ProposalStatus = "awaiting_review"
	ProposalExecuting      ProposalStatus = "executing"
	ProposalSucceeded      ProposalStatus = "succeeded"
	ProposalFailed         ProposalStatus = "failed"
	ProposalOutcomeUnknown ProposalStatus = "outcome_unknown"
	ProposalDenied         ProposalStatus = "denied"
	ProposalCancelled      ProposalStatus = "cancelled"
	ProposalExpired        ProposalStatus = "expired"
)

type Evidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	URL     string `json:"url,omitempty"`
}

type Step struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

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

type Goal struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Category    string      `json:"category,omitempty"`
	Status      string      `json:"status"`
	Milestones  []Milestone `json:"milestones,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Milestone struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

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

type MonitorPage struct {
	ID           string   `json:"id"`
	Hash         string   `json:"hash"`
	TimelessHash string   `json:"timeless_hash,omitempty"`
	Lines        []string `json:"lines,omitempty"`
}

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

type Activity struct {
	At     time.Time `json:"at"`
	Status string    `json:"status"`
	Detail string    `json:"detail,omitempty"`
}

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

type Choice struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id,omitempty"`
	Title     string         `json:"title"`
	Options   []ChoiceOption `json:"options"`
	Selected  string         `json:"selected,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ChoiceOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Source string `json:"source,omitempty"`
}

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

type WorkerStatus struct {
	Running    bool      `json:"running"`
	LastTickAt time.Time `json:"last_tick_at,omitempty"`
}

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
