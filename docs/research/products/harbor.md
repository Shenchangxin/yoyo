# Harbor framework (2026)

| | |
| --- | --- |
| 文档 | https://www.harborframework.com |
| Index | https://harbor-index.org |
| 论文 | [../corpus/2609.04298-harbor-index.md](../corpus/2609.04298-harbor-index.md)、[../corpus/2601.11868-terminal-bench.md](../corpus/2601.11868-terminal-bench.md) |
| 状态 | **partial** |

## 主张

Harbor 是沙盒 agent 任务的规格与执行器：同一任务布局评任意 agent，用于评价和优化。Adapter 规范要求 **parity** — 在相同 agent/模型/prompt 下，Harbor 分与原基准分统计不可区分。2026-09 的 Harbor-Index 把这件事收成 82 道高信号题，并用独立 verifier sandbox 防止 agent 偷看 grader。

## 对 Yoyo 的含义

我们实现的是 **兼容布局**，不是嵌入 Harbor 发行版：

- 任务目录形状见 `docs/architecture/harbor.md`。
- `evals/yoyo-agent/run.sh` 提供 `harbor run -a yoyo` 时应 exec 的同一 CLI。
- 晋升逻辑是我们自己的 `ShouldPromote`，不是 Harbor 产品 UI。

保持「布局兼容、门在本地」：这样将来跑 Index 或 TB 不必改 evolve，只要换 suite 对象。

## 可偷 / 应拒

- 偷：parity；独立 grader；任务即代码。
- 拒：把晋升权交给云端排行榜。
- 已对齐：worktree 隔离、safety 任务、held-out 对 proposer 不可见。
