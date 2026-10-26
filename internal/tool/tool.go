// Package tool 定义宿主工具的调用接口、注解和并发分类。规格字符串在 specs.go，那些 Description 是给模型看的文本。
package tool

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Annotations 是工具的只读、可并发、开放世界、破坏性和独占标记。
type Annotations struct {
	ReadOnly        bool `json:"read_only,omitempty"`
	ConcurrencySafe bool `json:"concurrency_safe,omitempty"`
	OpenWorld       bool `json:"open_world,omitempty"`
	Destructive     bool `json:"destructive,omitempty"`
	Exclusive       bool `json:"exclusive,omitempty"`
}

// Invocation 是一次调用的工具名、JSON 参数、会话和工作区。
type Invocation struct {
	Name      string
	ArgsJSON  string
	SessionID string
	Workspace string
}

// ContentBlock 是结果里的一块类型和文本。
type ContentBlock struct {
	Type string
	Text string
}

// FileChange 记录这次调用碰到的路径和补丁文本。
type FileChange struct {
	Paths []string
	Patch string
}

// Result 是工具返回的正文、分块、错误和可选的文件变更。
type Result struct {
	Content    string
	Blocks     []ContentBlock
	Err        error
	FileChange *FileChange
}

// Tool 是可登记的工具：规格、注解和一次调用。
type Tool interface {
	Spec() artifact.ToolSpec
	Annotations() Annotations
	Call(ctx context.Context, inv Invocation) Result
}

// AnnFromSpec 从规格抄出只读、可并发、开放世界、破坏性和独占五个标记。
func AnnFromSpec(s artifact.ToolSpec) Annotations {
	return Annotations{
		ReadOnly:        s.ReadOnly,
		ConcurrencySafe: s.ConcurrencySafe,
		OpenWorld:       s.OpenWorld,
		Destructive:     s.Destructive,
		Exclusive:       s.Exclusive,
	}
}

// ConcurrencyClass 在 Exclusive 或 OpenWorld 时返回 "exclusive"。只读且 ConcurrencySafe 时返回 "read"。其余返回 "write"。
func ConcurrencyClass(ann Annotations) string {
	if ann.Exclusive || ann.OpenWorld {
		return "exclusive"
	}
	if ann.ReadOnly && ann.ConcurrencySafe {
		return "read"
	}
	return "write"
}
