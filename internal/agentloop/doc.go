// Package agentloop 是 yoyo 自带的代理回合循环（agent loop）。
// 一个 Agent 负责自己的会话日志、收件箱（followup、steer、inject）、取消、工具批次（独占调用自成屏障，可并行调用走上限 DefaultMaxParallel 的并发池）、
// 前台超过预算的命令转后台任务，以及中断后补齐缺失的工具结果。
// 设计沿用 DeepSeek Harness packages/core 的可观察契约，本包旧名 dsh 即由此而来；轨迹事件的 Source 仍写 "dsh"，以兼容已落盘的会话日志。
// 使用方：桌面发送和会话 overlay 经子包 desk 进入；编排工单（internal/superagent）直接建 Agent；acpworker 和 mcpserve 复用本包的日志与工具类型。
package agentloop
