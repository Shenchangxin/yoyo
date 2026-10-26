// 拼 Harbor 容器里启动 Yoyo 的命令参数。不启动进程，也不给工作区打分。

package eval

// AgentArgs 返回 yoyo run --workspace 和说明。workspace 为空时用 /app。
// Harbor 解开 instruction.md 之后应按这个顺序执行，例如 yoyo run --workspace /app，再跟上说明文本。
func AgentArgs(workspace, instruction string) []string {
	if workspace == "" {
		workspace = "/app"
	}
	return []string{"yoyo", "run", "--workspace", workspace, instruction}
}
