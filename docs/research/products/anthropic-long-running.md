# Anthropic: harness design for long-running apps

| | |
| --- | --- |
| 来源 | 工程博客，不是论文 |
| 原文 | https://www.anthropic.com/engineering/harness-design-long-running-apps |
| 日期 | 2026-03-24 |
| 作者 | Prithvi Rajasekaran（Anthropic Labs） |
| 状态 | **candidate** |

## 主张

长程自主写应用会撞两堵墙：窗口填满后失焦（含「context anxiety」提前收工），以及模型给自己打高分。作者用 GAN 式 **generator / evaluator 拆分**，前端实验把品味写成可打分准则（设计质量、原创、工艺、功能），评测器拿 Playwright MCP 真点页面。

扩到全栈后是三代理：**planner**（一句话扩成规格，故意不写死实现细节）→ **generator**（按 sprint 做一块）→ **evaluator**（先谈 sprint 合同，再点应用打分；任一准则低于阈值则 sprint 失败）。代理之间用 **文件** 交接，不是共享一份无限对话。

关键工程观察：

1. **Context reset ≠ compaction。** 清窗口 + 结构化交接能治 anxiety；原地摘要留着同一条焦虑。Sonnet 4.5 必须 reset；Opus 4.5/4.6 可以丢掉 reset，靠 SDK 自动 compaction。
2. **Harness 组件编码的是「模型自己做不到的假设」。** 模型变强后要一项一项拆掉。Opus 4.6 上 sprint 分解可以删；evaluator 只在任务仍超出 solo 能力时才值得付钱。
3. 全套 harness 比 solo 贵一个数量级（游戏制作器：6h / $200 vs 20min / $9），质量差在「能不能玩」，不是排版。

## 对 Yoyo 的含义

Plan mode 已经是「提案、不写盘」。缺的是 **独立 QA 人格**：evolve 的 Harbor 是终局测试，不是 sprint 合同。办公长任务（inbox、长 PDF、多文件文档）会撞同一类 coherence 失败。

不要默认开 context reset。PRODUCT 要的是 spill 可对证；reset 丢掉 journal 链。等模型变强再减脚手架，和这篇同构 — 但减的是 L1 文案/middleware，不是 L4。

## 可偷 / 应拒

- 偷：生成与评判拆开；sprint 合同先于写代码；交接走文件；脚手架随模型能力收缩。
- 拒：无界多小时无人值守改用户机器；用 LLM 自评当 Harbor；为治 anxiety 做不可审计的窗口清空。
