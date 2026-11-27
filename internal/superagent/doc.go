// Package superagent 是 yoyo 的工单编排层：接住触发、评估是否接单、派工给工人并收回交付。设计沿用 dsh-superagent 原型。
// 领域名词与原型一致：Trigger、Thread、Eval、Mandate、Ticket、Assignment。
// 执行器用 internal/agentloop 的代理跑每个工单，不依赖 DeepSeek Harness 进程。
package superagent
