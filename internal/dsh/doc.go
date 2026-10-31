// Package dsh 是 Yoyo 技术栈上的 DeepSeek Harness 核心。
// 它复刻 packages/core 里会话日志、收件箱、工具批次和回合循环的可观察契约：
// followup、steer、inject、取消、独占屏障、并行池，以及 in-history 提示追加。
// 桌面发送经 internal/dshdesk 走本包的代理。Harbor 和会话 overlay 仍调用 runtime.Run。
package dsh
