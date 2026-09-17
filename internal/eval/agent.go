package eval

// AgentArgs is the Harbor-compatible invocation for running Yoyo inside a
// task container. Harbor should exec this after unpacking instruction.md:
//
//	yoyo run --workspace /app "$(cat /instruction.md)"
//
// The same Go eval engine then grades the workspace with the task verifier.
func AgentArgs(workspace, instruction string) []string {
	if workspace == "" {
		workspace = "/app"
	}
	return []string{"yoyo", "run", "--workspace", workspace, instruction}
}
