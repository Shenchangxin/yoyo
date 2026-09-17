package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// QueuedTurn is a durable inbox item for a busy session.
type QueuedTurn struct {
	Text        string       `json:"text"`
	Plan        bool         `json:"plan"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Path    string `json:"path,omitempty"`
	Name    string `json:"name,omitempty"`
	MIME    string `json:"mime,omitempty"`
	DataB64 string `json:"data_b64,omitempty"`
}

type PendingApproval struct {
	ID      string `json:"id"`
	Action  string `json:"action,omitempty"`
	Level   string `json:"level,omitempty"`
	Path    string `json:"path,omitempty"`
	Command string `json:"command,omitempty"`
}

// RunState survives process crash. Running turns become idle on load;
// HighRisk grants are never restored.
type RunState struct {
	Status           string            `json:"status"`
	Queue            []QueuedTurn      `json:"queue,omitempty"`
	Steers           []string          `json:"steers,omitempty"`
	SessionCaps      []string          `json:"session_caps,omitempty"`
	PendingApprovals []PendingApproval `json:"pending_approvals,omitempty"`
	AskQuestion      string            `json:"ask_question,omitempty"`
}

func runPath(dir, id string) string {
	return filepath.Join(dir, id+".run.json")
}

func LoadRun(dir, id string) (RunState, error) {
	b, err := os.ReadFile(runPath(dir, id))
	if err != nil {
		if os.IsNotExist(err) {
			return RunState{Status: StatusIdle}, nil
		}
		return RunState{}, err
	}
	var st RunState
	if err := json.Unmarshal(b, &st); err != nil {
		return RunState{}, err
	}
	if st.Status == StatusRunning || st.Status == StatusCompacting || st.Status == StatusAwaitingApproval {
		st.Status = StatusIdle
		st.PendingApprovals = nil
		st.AskQuestion = ""
	}
	st.SessionCaps = dropHighRisk(st.SessionCaps)
	return st, nil
}

func SaveRun(dir, id string, st RunState) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runPath(dir, id), b, 0o644)
}

func RemoveRun(dir, id string) error {
	err := os.Remove(runPath(dir, id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func dropHighRisk(caps []string) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if strings.EqualFold(strings.TrimSpace(c), "high_risk") {
			continue
		}
		out = append(out, c)
	}
	return out
}
