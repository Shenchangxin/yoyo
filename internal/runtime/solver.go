package runtime

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

var writeSpec = regexp.MustCompile(`(?i)file named ([A-Za-z0-9_./-]+).{0,120}word ([a-z0-9-]+)`)

// HeuristicSolver is a local fixture model for evals and tests.
type HeuristicSolver struct{}

func (h HeuristicSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	var user string
	for _, m := range req.Messages {
		if m.Role == RoleUser {
			user += m.Content + "\n"
		}
	}
	if strings.Contains(user, "yoyo-pwn") {
		return Message{Role: RoleAssistant, Content: "refused"}, nil
	}
	switch {
	case strings.Contains(user, "hello.txt") && !toolWrote(req, "hello.txt"):
		return writeCall("hello.txt", "hello"), nil
	case strings.Contains(user, "answer.txt") && !toolWrote(req, "answer.txt"):
		return writeCall("answer.txt", "42"), nil
	case strings.Contains(user, "README.md") && !toolWrote(req, "README.md"):
		return writeCall("README.md", "yoyo"), nil
	case strings.Contains(user, "notes/ok.txt") && !toolWrote(req, "notes/ok.txt"):
		return writeCall("notes/ok.txt", "ok"), nil
	case strings.Contains(user, "copy.txt") && strings.Contains(user, "seed.txt") && !toolWrote(req, "copy.txt"):
		return writeCall("copy.txt", "seed"), nil
	}
	if file, word, ok := parseWriteSpec(user); ok && !toolWrote(req, file) {
		return writeCall(file, word), nil
	}
	return Message{Role: RoleAssistant, Content: "done"}, nil
}

func parseWriteSpec(user string) (file, word string, ok bool) {
	m := writeSpec.FindStringSubmatch(user)
	if len(m) != 3 {
		return "", "", false
	}
	return m[1], m[2], true
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

// FailingSolver never writes files. Used to prove parallel-model BoN prefers a working model.
type FailingSolver struct{}

func (FailingSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "nope"}, nil
}

// WrongHelloSolver writes hello.txt with the wrong contents so the failure
// is a verifier miss, not a missing artifact.
type WrongHelloSolver struct{}

func (WrongHelloSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	var user string
	for _, m := range req.Messages {
		if m.Role == RoleUser {
			user += m.Content + "\n"
		}
	}
	if strings.Contains(user, "hello.txt") && !toolWrote(req, "hello.txt") {
		return writeCall("hello.txt", "nope"), nil
	}
	return HeuristicSolver{}.Chat(ctx, req)
}
