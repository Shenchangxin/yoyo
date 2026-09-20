# ICLR 2026 Workshop: AI with Recursive Self-Improvement

| | |
| --- | --- |
| 会场 | 2026-04-26 · [ICLR virtual](https://iclr.cc/virtual/2026/workshop/10000796) |
| OpenReview | https://openreview.net/submissions?venue=ICLR.cc/2026/Workshop/RSI |
| 组织 | Mingchen Zhuge, Ailing Zeng, Deyao Zhu, Rong Zou, Yan Hu, Sherry Yang, Vikas Chandra, Jürgen Schmidhuber |
| 状态 | **watch**（入口，不是一篇论文） |

## 主张

RSI 从思想实验走进部署。Workshop 按五条透镜组织投稿：系统内的改动对象、适应的时间尺度、机制、运行场景、改进的证据。明确欢迎改权重、改 prompt、改控制器，以及评测基础设施。邀请报告含 Chelsea Finn、Jeff Clune、Sergey Levine、Yuandong Tian。

## 对本库已收录的对齐

| 工作 | 本库 |
| --- | --- |
| AutoHarness（Lou / Murphy 等） | [corpus/2603.03329-autoharness.md](../corpus/2603.03329-autoharness.md) |
| Reward Hacking in Self-Improving Code Agents | [corpus/iclr2026-rsi-reward-hacking.md](../corpus/iclr2026-rsi-reward-hacking.md) |
| ACE（会场也挂了） | [corpus/2510.04618-ace.md](../corpus/2510.04618-ace.md)（谱系） |
| Evaluating AGENTS.md（Gloaguen 等） | [corpus/2602.11988-agents-md.md](../corpus/2602.11988-agents-md.md) |
| Simple Baselines are Competitive with Code Evolution | [corpus/2602.16805-simple-baselines.md](../corpus/2602.16805-simple-baselines.md) |
| Feedback Descent（Lee / Finn；2025-first） | [corpus/2511.07919-feedback-descent.md](../corpus/2511.07919-feedback-descent.md)（谱系） |
| HGM 作者线在组织委员会 | [corpus/2510.21614-hgm.md](../corpus/2510.21614-hgm.md)（谱系） |

其余海报（VLA、分子、世界模型、post-training bench）与桌面 self-harness 控制塔不对齐，不进 catalog。下次刷新只扫 **改 harness / 改评测协议 / 改技能记忆** 的新 camera-ready。

## 可偷 / 应拒

- 偷：把「改什么 / 何时改 / 拿什么当证据」写成 evolve 实验的元数据，而不是又一篇口号。
- 拒：把 workshop 上的权重更新、Godel-agent 改源码、无门自改安全对齐当成产品路径。
