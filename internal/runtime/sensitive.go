package runtime

import (
	"context"
	"strings"
)

// PromptSensitiveSolver only writes artifacts when the harness tells it to
// create a placeholder file. Used to prove L1 prompt evolution changes outcomes.
type PromptSensitiveSolver struct{}

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
