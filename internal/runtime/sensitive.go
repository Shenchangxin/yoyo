// 提示词敏感的夹具模型：只有系统提示里出现 placeholder file 才写文件。
// 不跑真实模型。rpc、cmd/yoyo 和演进测试用它证明提示词会改变是否落盘。

package runtime

import (
	"context"
	"strings"
)

// PromptSensitiveSolver 在系统或 developer 文本含 placeholder file（不区分大小写）时，才为 hello.txt 或 answer.txt 发出 write_file。
type PromptSensitiveSolver struct{}

// Chat 拼接系统与用户文本。已有对应写入记录则不再写。未武装时回复探索过工作区、没有写文件。其余回复 done。
func (PromptSensitiveSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	var sys, user string
	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem, RoleDeveloper:
			sys += m.Content
		case RoleUser:
			user += m.Content
		}
	}
	armed := strings.Contains(strings.ToLower(sys), "placeholder file")
	switch {
	case strings.Contains(user, "hello.txt") && armed && !toolWrote(req, "hello.txt"):
		return writeCall("hello.txt", "hello"), nil
	case strings.Contains(user, "answer.txt") && armed && !toolWrote(req, "answer.txt"):
		return writeCall("answer.txt", "42"), nil
	case strings.Contains(user, "hello.txt") && !armed:
		return Message{Role: RoleAssistant, Content: "explored workspace, no file written"}, nil
	case strings.Contains(user, "answer.txt") && !armed:
		return Message{Role: RoleAssistant, Content: "explored workspace, no file written"}, nil
	default:
		return Message{Role: RoleAssistant, Content: "done"}, nil
	}
}
