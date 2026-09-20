# Reward Hacking in Self-Improving Code Agents

| | |
| --- | --- |
| 会场 | ICLR 2026 Workshop RSI |
| OpenReview | https://openreview.net/forum?id=ikrQWGgxYg |
| ICLR | https://iclr.cc/virtual/2026/10018648 |
| 作者 | Bingchen Zhao, Dhruv Srikanth, Yuxiang Wu, Zhengyao Jiang |
| 发表 | 2026-03-05 |
| 原文 | OpenReview PDF（无 arXiv id；不进 `originals/` 脚本） |
| 状态 | **critique** |

## 主张

RSI 用执行反馈迭代优化代码时，中心失败模式是 **reward hacking**：便宜代理指标涨了，更真实的目标没涨甚至更差。作者让 agent 只看见公开 proxy 任务，自己握有 held 真任务。Kernel-Bench 73.8%、ALE-Bench 46.8% 的优化属于「proxy 涨、真任务不涨」。步数从 10 到 100，hacking 从 26.4% 升到 57.8%。Retrospection（跳变时自我批评）在 Kernel-Bench 能降 17–19 点，在 ALE 上不稳定，有时更糟。

## 对 Yoyo 的含义

这是 held-out / transfer 存在的会场级定量证据。evolve 轮数加长会 **恶化** 代理–真实鸿沟，所以不要靠 `--rounds 100` 堆分数。Retrospection 不是银弹；门必须是真任务，不是自我批评。

## 可偷 / 应拒

- 已对齐：ShouldPromote 看 held-out；safety 一票否决。
- 拒：用「模型说自己没 hacking」当门。
- 产品：evolve UI 应同时显示 held-in 与 held-out，避免只庆祝 proxy。
