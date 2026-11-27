// Package desk 是 agentloop 在桌面上的适配层，把桌面的一轮发送交给 agentloop.Agent。
// 桌面聊天（App.sendLocked）和会话 overlay 探针（session_overlay.go）都经 Host.Run 走这里；task 子代理也在这里挂上，见 subagent.go。
// 工具仍是 runtime.WorkspaceTools，模型客户端仍是 runtime.Client，经 Bridge 接到 agentloop.Completer。轨迹事件仍是界面已经认识的 trace 事件，见 project.go。
// 评测 Harbor（internal/eval）也经 Host.Run 跑每一题：提示换成 runtime.HarborPrompt 拼的完整前缀，Turn.MaxSteps 给出硬步数上限。
package desk
