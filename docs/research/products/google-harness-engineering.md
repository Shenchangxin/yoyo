# Google: The Anatomy of Harness Engineering

| | |
| --- | --- |
| 来源 | Google Developers Blog |
| 原文 | https://developers.googleblog.com/the-anatomy-of-harness-engineering-how-to-evaluate-iterate-and-guard-ai-coding-agents/ |
| 日期 | 2026-09-09 |
| 作者 | Taylor Mullen, Christian Gunderman |
| 状态 | **candidate** |

## 主张

团队盯 Terminal-Bench / DeepSWE 复合分涨跌几个点，往往不知道为什么。端到端基准像期末考试：分数掉了，不知道是猜了歧义需求、忘了跑测试，还是捏造了 CLI flag。

**行为评测** 是 harness 的集成测试：断言中间动作，而不是最终散文。

- 需求含糊时有没有提问而不是猜。
- 改构建文件后、声称完成前，有没有跑本地校验。
- 写文档时有没有给仓库里的权威链接。

架构：本地、确定、秒级的微观断言 + 宏观基准。简单任务可以卡「必须调用 test-runner」；复杂任务不要锁死工具序列，改用更糊的结果检查。因为模型非确定，不要用单次 eval 卡 PR，要看批量通过率趋势。

Dogfooding 在前：智能体还不会改自己的仓、写样板之前，上评测套件没有意义。评测的第一目的不是庆祝 +2%，而是保证 prompt / schema / 模型升级没有整体变坏。

文中示例：用 LLM 改自己的 system prompt，直到失败的行为测试通过，其余套件当 CI 护栏 — 这是 evolve 的微观版。

## 对 Yoyo 的含义

我们只有 Harbor 终局。evolve 实验室缺一层便宜的行为套件：声称完成前必须跑 verifier；proposer 不得碰 `tests/`。这与 RHB、ICLR RSI reward-hacking 同一层，只是延迟从「几小时 Harbor」变成「几秒 pytest」。

## 可偷 / 应拒

- 偷：微观行为断言；批量看趋势；宏观 + 微观互补。
- 拒：用 LLM-as-judge 替换 Harbor 的确定性测试；把 prompt 自改环接到 `refs/active`。
