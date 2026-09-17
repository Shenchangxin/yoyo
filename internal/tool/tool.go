package tool

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type Annotations struct {
	ReadOnly        bool `json:"read_only,omitempty"`
	ConcurrencySafe bool `json:"concurrency_safe,omitempty"`
	OpenWorld       bool `json:"open_world,omitempty"`
	Destructive     bool `json:"destructive,omitempty"`
	Exclusive       bool `json:"exclusive,omitempty"`
}

type Invocation struct {
	Name      string
	ArgsJSON  string
	SessionID string
	Workspace string
}

type ContentBlock struct {
	Type string
	Text string
}

type FileChange struct {
	Paths []string
	Patch string
}

type Result struct {
	Content    string
	Blocks     []ContentBlock
	Err        error
	FileChange *FileChange
}

type Tool interface {
	Spec() artifact.ToolSpec
	Annotations() Annotations
	Call(ctx context.Context, inv Invocation) Result
}

func AnnFromSpec(s artifact.ToolSpec) Annotations {
	return Annotations{
		ReadOnly:        s.ReadOnly,
		ConcurrencySafe: s.ConcurrencySafe,
		OpenWorld:       s.OpenWorld,
		Destructive:     s.Destructive,
		Exclusive:       s.Exclusive,
	}
}

func ConcurrencyClass(ann Annotations) string {
	if ann.Exclusive || ann.OpenWorld {
		return "exclusive"
	}
	if ann.ReadOnly && ann.ConcurrencySafe {
		return "read"
	}
	return "write"
}
