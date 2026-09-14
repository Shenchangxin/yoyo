package runtime

import (
	"context"
	"encoding/json"
	"strings"
)

// HeuristicSolver is a local fixture model for evals and tests.
type HeuristicSolver struct{}

func (h HeuristicSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	var user string
	for _, m := range req.Messages {
		if m.Role == RoleUser {
			user += m.Content + "\n"
		}
	}
	switch {
	case strings.Contains(user, "hello.txt") && !toolWrote(req, "hello.txt"):
		return writeCall("hello.txt", "hello"), nil
	case strings.Contains(user, "answer.txt") && !toolWrote(req, "answer.txt"):
		return writeCall("answer.txt", "42"), nil
	default:
		return Message{Role: RoleAssistant, Content: "done"}, nil
	}
}

func toolWrote(req ChatRequest, name string) bool {
	for _, m := range req.Messages {
		if m.Role != RoleTool && m.Role != RoleAssistant {
			continue
		}
		if strings.Contains(m.Content, "wrote "+name) {
			return true
		}
		for _, tc := range m.ToolCalls {
			if tc.Name == "write_file" && strings.Contains(tc.Arguments, name) {
				return true
			}
		}
	}
	return false
}

func writeCall(path, content string) Message {
	args, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
		ID: "w1", Name: "write_file", Arguments: string(args),
	}}}
}
