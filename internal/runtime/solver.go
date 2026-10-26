// 评测和测试用的本地夹具模型：按用户句子决定写哪个文件，或拒绝。
// 不访问网络。rpc、cmd/yoyo 和若干会话测试把它们当作 Client。

package runtime

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// writeSpec 匹配 “file named <路径> … word <词>”，两段之间最多 120 个字节。
var writeSpec = regexp.MustCompile(`(?i)file named ([A-Za-z0-9_./-]+).{0,120}word ([a-z0-9-]+)`)

// HeuristicSolver 是评测和测试用的本地夹具模型。
type HeuristicSolver struct{}

// Chat 在用户文本含 yoyo-pwn 或 leaked.txt 时回复 refused。否则按固定文件名或 writeSpec 发出一次 write_file，已写过则回复 done。
func (h HeuristicSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	var user string
	for _, m := range req.Messages {
		if m.Role == RoleUser {
			user += m.Content + "\n"
		}
	}
	if strings.Contains(user, "yoyo-pwn") || strings.Contains(user, "leaked.txt") {
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

// parseWriteSpec 从用户文本抽出文件名和要写入的词。匹配失败返回 false。
func parseWriteSpec(user string) (file, word string, ok bool) {
	m := writeSpec.FindStringSubmatch(user)
	if len(m) != 3 {
		return "", "", false
	}
	return m[1], m[2], true
}

// toolWrote 在工具或助手消息里找 “wrote <name>”，或 write_file 参数里是否含该名字。
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

// writeCall 构造一条 write_file 工具调用，参数是 path 和 content。
func writeCall(path, content string) Message {
	args, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
		ID: "w1", Name: "write_file", Arguments: string(args),
	}}}
}

// FailingSolver 从不写文件。用来证明多模型择优会选能写出文件的那个。
type FailingSolver struct{}

// Chat 固定回复 nope。
func (FailingSolver) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "nope"}, nil
}

// WrongHelloSolver 把 hello.txt 写成错误内容，使失败来自校验而不是缺少文件。
type WrongHelloSolver struct{}

// Chat 在尚未写过 hello.txt 时写入 nope，其余交给 HeuristicSolver。
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
