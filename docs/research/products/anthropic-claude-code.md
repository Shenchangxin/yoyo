# Claude Code: official loop, permissions, extensions (2026)

| | |
| --- | --- |
| How it works | https://code.claude.com/docs/en/how-claude-code-works |
| Agent SDK loop | https://code.claude.com/docs/en/agent-sdk/agent-loop.md |
| 源码级对照 | [../corpus/2604.14228-claude-code.md](../corpus/2604.14228-claude-code.md) |
| 状态 | **partial** |

## 主张

官方把 Claude Code 说成模型 + harness：gather context → take action → verify，循环直到没有 tool call。工具让模型能行动；没有工具就只是聊天。Agent SDK 暴露同一套 loop：init → evaluate/respond → execute tools（hooks 可拦截）→ 重复 → ResultMessage（用量、费用、session id）。

周边系统（文档与 2604.14228 一致）：权限模式、分层压缩、CLAUDE.md、skills、MCP、plugins、hooks、subagents。

## 对 Yoyo 的含义

Transcript 语法（过程行、审批卡、hunk apply）刻意靠近「操作员看着智能体工作」，而不是聊天气泡产品。权限我们做得更硬、更少魔法：没有隐藏分类器，身份动作不能 Always。

Skills 与 CLAUDE.md 针已经接上，所以一个既用 Claude Code 又用 Yoyo 的仓库不必维护两套完全不同的公约。进化仍然只发生在 Yoyo 的 CAS 里 — Claude 不会给我们写 `refs/canary`。

## 可偷 / 应拒

- 偷：loop 的对外叙事（context / action / verify）；SDK 级可嵌入；审批可拦截。
- 拒：把 Claude 的 hook 词汇当成兼容性目标（Barbaste 观察到 Codex 在抄；我们不需要加入这场模仿）。
- 已对齐：depth-1 子代理；workspace markdown 针；MCP。
