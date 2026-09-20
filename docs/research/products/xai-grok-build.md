# xAI Grok Build（开源 harness）

| | |
| --- | --- |
| 来源 | 产品公告 + GitHub，不是论文 |
| 公告 | https://x.ai/news/grok-build-open-source （2026-07-15） |
| 代码 | https://github.com/xai-org/grok-build （Apache-2.0，Rust） |
| 状态 | **watch** |

## 主张

xAI 开源的是 **coding agent harness 和 TUI**，不是 Grok 权重。仓库同步自内部 monorepo，包含：上下文组装与工具分发、读写搜跑命令、全屏 TUI（plan review、inline diff）、以及 skills / plugins / hooks / MCP / subagents。可本机编译，把 `config.toml` 指到自有 OpenAI 兼容推理。也支持 ACP 嵌进编辑器、headless 跑 CI。

## 对 Yoyo 的含义

又一个「harness 是产品、模型可替换」的工业证据，和 Codex / Claude / Barbaste 的平台化观察同方向。对照用，不要移植：

- 他们是 Rust TUI；我们是 Go core + Wails / CLI / JSON-RPC。
- 他们没有 CAS/refs、没有 Harbor 晋升、没有 L4 冻结叙事。
- ACP 是编辑器嵌入协议；我们已有 App Server。不要为了对齐再做第三套 IPC。

本地推理（Ollama 等）是他们的卖点之一。Yoyo 的模型指纹 refs 已经按模型分 harness；接本地 endpoint 是产品问题，不是文献缺口。

## 可偷 / 应拒

- 偷：harness 与权重解耦；extensions 的加载路径可读。
- 拒：把 TUI 做成第二套 loop；为了「也开源一份 Rust agent」重写 Go 内核。
