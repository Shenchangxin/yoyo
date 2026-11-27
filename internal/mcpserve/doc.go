// Package mcpserve 是 yoyo 内置的 MCP 控制面，让外部 agent 调用自己那张工单上的 orchestrator_* 工具。
//
// server.go 提供 HTTP 端点、按 token 的工单登记和会话号（Server、New、NewToken）；bridge.go 的 RunBridge 是 stdio 到这个
// HTTP 端点的桥，给只会启动 stdio MCP 服务器的 agent 用（以 YOYO_MCP_BRIDGE=1 启动 yoyo 自身）。
//
// 这个包不拥有工具本身（superagent 的执行器建好，app 登记），不拥有外部进程（acpworker），也不决定何时启动桥。
// 安全边界：只监听 127.0.0.1；每张工单一个随机 token，只能列出和调用这张工单登记的工具；Host 和 Origin 必须指向本机；请求体最多 1 MB。
package mcpserve
