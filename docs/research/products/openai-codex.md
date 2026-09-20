# OpenAI Codex: agent loop and App Server (2026)

| | |
| --- | --- |
| 来源 | 工程博客，不是论文 |
| Loop | https://openai.com/index/unrolling-the-codex-agent-loop/ |
| App Server | https://openai.com/index/unlocking-the-codex-harness/ |
| 状态 | **implemented** |

## 主张

**Loop。** Codex CLI 的核心是：组装 prompt（系统、工具 schema、上下文）→ 调 Responses API → 要么得到最终 assistant message，要么执行工具并把输出追加后再采样。直到模型不再要工具。用户看到的「一轮」里面可以有几十次内部迭代。

**App Server。** Codex 的 Web / CLI / IDE / macOS 共用同一套 harness。Core 是库也是 runtime，管一条 thread 的持久化。App Server 是双向 JSON-RPC：client 请求，server 大量 notification；需要 approval 时 **server 反向请求并暂停 turn**。计划让 TUI 也变成「拉起一个 App Server 子进程」的普通 client。

## 对 Yoyo 的含义

这是桌面架构的直接模板：

- 一个 Go core，CLI / `yoyo serve` / Wails 都是 client。
- `internal/api/rpc.go`：`turn.start`、`item.event`、`approval.request`。
- `YOYO_ISOLATE=1` / `YOYO_WORKER=1`：窗口进程不运行可能卡死的 loop。

我们改的一处：compaction。Codex 走不可逆压缩（对企业 ZDR 有意义）。Yoyo 把工具原文 spill 到磁盘，Shape 只是投影。个人工作站更需要「打开 spill 就能对证」。

## 可偷 / 应拒

- 已偷：单 harness、多 client、双向 RPC、approval 暂停。
- 拒：不可逆丢掉工具字节；把桌面做成「另一个写了 loop 的前端」。
