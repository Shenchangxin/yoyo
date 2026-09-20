# 2026 前沿文献体系

给 owner 用的工作文档：哪些 2026 年论文/报告现在最值钱，它们和 Yoyo 差在哪，下一轮该改什么。扫读请打开 [report.html](report.html)。

**年份门**：本文件只讨论 2026 年首次出现的工作。ACE、DGM 已在 [REFERENCES.md](REFERENCES.md)。2025 年的 Reflexion / GEPA / STOP 视为过时基线，不再展开。

检索窗口：2026-01-01 → 2026-09-20（今天）。渠道见 [SOURCES.md](SOURCES.md)：arXiv、OpenReview、ICLR/ICML 2026、HF Daily Papers、alphaXiv、厂商博客（OpenAI / Anthropic / Google / xAI / Cursor / Harbor）、Semantic Scholar / Connected Papers。ACL/EMNLP/NeurIPS 2026 主会本季无直接命中，下次刷新再扫。

---

## 1. 这一季的判断

2026 上半年，harness 从「prompt 工程的别名」变成一门有自己评测协议的学科。三件事实叠在一起，决定 Yoyo 该怎么走：

1. **固定模型、改脚手架，是真的能涨分的。** Self-Harness 在 TB2.0 / SWE-Verified / AppWorld 九个模型–基准组合上，终态 harness 的 held-in 与 held-out 都升（相对增益最高约 132%）。这直接为 `yoyo evolve` 提供了主定理。
2. **涨分很容易是过拟合或「多搜了几次」。** Wang et al.（2026-07）在 TB 2.1 上用匹配预算对照：自动 harness 进化 **并不稳定地赢过** 简单 test-time scaling。ModularRSI（2026-09-14）把解法说成：进化数据必须与评测基准 **不相交**，并且按模块归因。
3. **生产系统已经平台化。** Barbaste 等解剖十一套代码：没有人用 LangChain；技能标准赢了 MCP 的采用率；Codex 在抄 Claude 的 hook 词汇。Yoyo 的「一个 Go core + 多 client」站在这条线上，不要倒回去做 IDE 皮肤。

因此 owner 的默认策略不是「再写长一点的 system prompt」，而是：

- 守住已经落地的 Self-Harness 门（held-out、safety、canary ≠ active）。
- 下一跃迁是 **模块化 + 评测不相交 + 预算诚实**，不是更宽的 K。
- 任何会让 evolve 改 Go / 评测器 / 密钥的想法，直接丢。

---

## 2. 分层目录（只含 2026）

价值从高到低。★ 是本季必读。

### A. RSI / Self-Harness 主线（产品控制塔）

| 优先级 | 工作 | 日期 | 为什么现在读 | Yoyo 状态 |
| --- | --- | --- | --- | --- |
| ★ P0 | [Self-Harness](corpus/2606.09498-self-harness.md) | 2026-06-08 | 我们的 evolve 循环几乎是这篇的实现 | implemented |
| ★ P0 | [ModularRSI](corpus/2609.14857-modular-rsi.md) | 2026-09-14 | **本季最新、最该偷**：五模块、对比轨迹、2000 道不相交进化题 | candidate |
| ★ P0 | [Rethinking harness-evolution eval](corpus/2607.12227-rethink-eval.md) | 2026-07-14 | 否证压力。没有匹配预算对照，evolve 实验室是自欺 | critique |
| ★ P1 | [HarnessEvolve](corpus/2609.00829-harness-evolve.md) | 2026-09-01 | 参考轨迹解决 credit assignment；质量门打 prompt 膨胀 | candidate |
| P1 | [HSI](corpus/2608.08466-hsi.md) | 2026-08-09 | 分层自改 + 冻结外锚。我们 L4 已经是外锚；不要做 meta-evolver 改 evolve 代码 | partial |
| P1 | [RHI](corpus/2607.15524-rhi.md) | 2026-07-17 | prompt 级 harness、成对修订历史、少步数。适合「用户任务特化」而非全局晋升 | candidate |
| P1 | [AHE](corpus/2604.25850-ahe.md) | 2026-04-28 | 可观测性三柱：组件文件化、轨迹蒸馏、可证伪的 change manifest | candidate |
| P1 | [Meta-Harness](corpus/2603.28052-meta-harness.md) | 2026-03-30 | ICML 2026。用文件系统当证据通道，coding agent 自己诊断。不要让它 grep 评测器 | candidate |
| P2 | [HarnessBank](corpus/2607.13683-harness-bank.md) | 2026-07-15 | 语义 gene bank + gated screening。档案升级备选 | watch |
| P2 | [Recuris](corpus/2608.24876-recuris.md) | 2026-08-25 | 工作记忆引导技能选择；局部、过门的 skill 更新 | candidate |
| P3 | [Dream-RSI](corpus/2609.14858-dream-rsi.md) | 2026-09-14 | 用历史发现树做 off-policy「做梦」。搜索域；先看模式 | watch |
| P3 | [VideoHarness-RSI](corpus/2608.24302-videoharness-rsi.md) | 2026-08-25 | 把 **上下文构造程序** 当成独立优化层。对 `Shape` 有启发 | watch |
| P3 | [HELIX](corpus/2608.13951-helix.md) | 2026-08-14 | 模型–harness 共进化的证据平面。我们冻结权重 | watch |
| P3 | [AutoHarness](corpus/2603.03329-autoharness.md) | 2026-02-10 | ICLR RSI workshop。合成合法性代码包装；DeepMind 作者线 | watch |
| P3 | [Harness-R1](corpus/2608.02276-harness-r1.md) | 2026-08-03 | 用 RL 训练 harness 工程师。我们不训权重 | watch |
| P3 | [AEvo](corpus/2605.13821-aevo.md) | 2026-05-13 | 把进化程序本身当环境。L4 禁止 | watch |
| P3 | [RAH](corpus/2606.13643-rah.md) | 2026-06-11 | 递归单位是整份 harness 不是一次模型调用。我们 depth-1 | watch |

### B. 评测与「harness 到底有没有用」

| 优先级 | 工作 | 日期 | 为什么现在读 | Yoyo 状态 |
| --- | --- | --- | --- | --- |
| ★ P0 | [Terminal-Bench 2.0](corpus/2601.11868-terminal-bench.md) | 2026-01-17 | 领域标准。Self-Harness / ModularRSI 都在这上面说话 | partial |
| ★ P0 | [Harness design study](corpus/2609.20804-harness-design.md) | 2026-09-17 | 规则裁剪先于 LLM 摘要最有效；可恢复性几乎不涨分。直接给 `Shape` 背书 | partial |
| ★ P1 | [Harbor-Index](corpus/2609.04298-harbor-index.md) | 2026-09-03 | 82 道难而多样、全模型 <30%。比盲目扩 sealed suite 更值 | candidate |
| P1 | [Harness or Model?](corpus/2609.11987-harness-or-model.md) | 2026-09-08 | 同模型换 harness，平均差接近 0，但仓库 vs 竞赛交互显著 | critique |
| P1 | [Same model, different harness](corpus/2608.26218-same-model-harness.md) | 2026-08-26 | 紧窗口下裁旧工具结果显著涨分。评测必须带 harness | critique |
| P1 | [HarnessDev](corpus/2609.01437-harness-dev.md) | 2026-09-01 | 把「会不会造/进化 harness」本身当基准。进化增益不稳、难跨模型 | critique |
| P1 | [RSI reward hacking](corpus/iclr2026-rsi-reward-hacking.md) | 2026-03 | ICLR RSI workshop：代理指标涨、真任务不涨。held-out 的存在理由 | critique |
| P1 | [Simple baselines vs code evolution](corpus/2602.16805-simple-baselines.md) | 2026-02-18 | IID/SCS 打平复杂进化。evolve 必须先赢随机采样 | critique |
| P2 | [RHB](corpus/2605.02964-rhb.md) | 2026-05-03 | ICML 2026。工具链路上的捷径/篡改评测。安全套件的下一刀 | critique |
| P2 | [Evaluating AGENTS.md](corpus/2602.11988-agents-md.md) | 2026-02-12 | 仓库针不涨分、成本 +20%。只写非标准惯例 | critique |

### C. 生产 harness 解剖（桌面 agent 的工艺）

| 优先级 | 工作 | 日期 | 为什么现在读 | Yoyo 状态 |
| --- | --- | --- | --- | --- |
| ★ P1 | [十一套源码解剖](corpus/2609.00006-harness-anatomy.md) | 2026-07-15 | 2026 上半年的工业地图：七子系统、29 模式、平台化 | partial |
| ★ P1 | [Dive into Claude Code](corpus/2604.14228-claude-code.md) | 2026-04-14 | 设计空间：权限、五层压缩、四种扩展、subagent | partial |
| P1 | [Codex loop / App Server](products/openai-codex.md) | 2026 | 我们 JSON-RPC 的直接模板 | implemented |
| P1 | [Anthropic 长程 harness](products/anthropic-long-running.md) | 2026 | 冲刺合同、生成器/评测器、上下文重置随模型变强而可删 | candidate |
| P1 | [Google 行为评测](products/google-harness-engineering.md) | 2026-09-09 | 微观行为断言 + 宏观基准。Harbor 的本地快测层 | candidate |
| P2 | [配置机制 2853 仓](corpus/2602.14690-harness-config.md) | 2026-02-16 | AGENTS.md 是公约；skills/subagents 实际很少被仓库用好 | implemented |
| P2 | [Evaluating AGENTS.md](corpus/2602.11988-agents-md.md) | 2026-02-12 | 公约很容易变成税。与 Galster 一起读 | critique |
| P2 | [Composer 2](corpus/2603.24477-composer-2.md) | 2026-03-25 | 模型–harness 共训。我们不训模型，但 trace 质量是 RHI 命题 | watch |
| P2 | [Grok Build 开源](products/xai-grok-build.md) | 2026-07-15 | xAI 的 Rust harness：ACP/skills/hooks。对照而不是移植 | watch |
| P3 | [ICLR RSI workshop](products/iclr2026-rsi-workshop.md) | 2026 | 当年 RSI 会场入口 | watch |

### D. Skills / 上下文（窗口是稀缺资源）

| 优先级 | 工作 | 日期 | 为什么现在读 | Yoyo 状态 |
| --- | --- | --- | --- | --- |
| P1 | [SoK: Agentic Skills](corpus/2602.20867-sok-skills.md) | 2026-02-24 | 技能生命周期 + ClawHavoc 供应链（~1200 恶意技能） | partial |
| P1 | [SkillReducer](corpus/2603.29919-skill-reducer.md) | 2026-03-31 | 公开技能 26% 没有 routing 描述；压缩后质量反而升 | candidate |
| P2 | [SKILL.state](corpus/2608.26263-skill-state.md) | 2026-08-26 | 丢掉 append-only 对话，改用可变执行状态。长程 skill 的下一代形状 | candidate |

---

## 3. 与 Yoyo 的差距（只写可执行的）

对照当前代码（v0.3.0），不是愿望清单。

### 3.1 已经领先或对齐的

- **晋升语义比 Self-Harness 原文更适合桌面产品**：canary / staging / active 三指针，人的 checkout 才动信任根。
- **L4 冻结** 比 HSI 的「冻结 meta-evolver 执行逻辑」更硬：连 evolver 策略代码都不可被 agent 改。对个人工作站这是正确的恐惧。
- **Shape 是无损投影**，比 Codex 不可逆 compaction 更可审计。
- **held-out 对 proposer 保密** 已经写进不变量，对齐 Wang / ModularRSI 的核心批评。

### 3.2 本季该补的洞

1. **进化数据与评测数据相交。** smoke/sealed 的 held-in 既是 evolve 的证据源，也是 `ShouldPromote` 的一半。ModularRSI 的答案是外部 2000 题。最小动作：把 sealed held-in 再切一刀，evolve 只用其中一部分，晋升仍看全量 held-out + transfer。
2. **单轨迹更新。** `Mine` 聚的是失败；`Passing` 只是 id 列表。ModularRSI 要求 **同一任务** 的成功/失败对比。最小动作：Harbor repeats≥2 时，把 pass 与 fail 轨迹成对送进 proposer。
3. **整包 L1，而不是五模块。** 提案表面已经分开（playbook vs instruction vs middleware），但没有「只改 Observation Management」这种受限补丁，也没有集成阶段解冲突。
4. **没有质量门。** HarnessEvolve / SkillReducer / Gloaguen：过滤泄漏、prompt 膨胀、无动作内容、仓库概览。Yoyo 只靠 Harbor 分数。一个过拟合的长 bullet 只要 held-in 升、held-out 没掉，就能进 canary。
5. **evolve 的算力账没记。** Wang et al. 与 Gideoni et al. 会问：这 K 个提案 × Harbor repeats，是不是还不如 IID 采样或 `eval --best N`？实验室没有把三者画在同一张预算图上。
6. **技能还是静态说明书。** SoK 的 Pattern-2（可执行 skill）与 SKILL.state（可变状态）都还没碰。当前 skill 是 markdown + 可选 WASM，执行仍走主对话。
7. **没有微观行为评测。** Google 2026-09 的 harness 文把「有没有跑测试就声称完成」做成秒级断言。我们只有 Harbor 终局。evolve 实验室需要一层便宜的行为套件。
8. **提案的 predicted_fixes 没有被下一轮核实。** AHE 的 decision observability 要求每条编辑是可证伪合同。`Proposal.PredictedFixes` 已经有字段，Harbor 报告没有拿它对照。

### 3.3 不要做的

- 不要做 HSI / AEvo 式 **meta-evolver 改 `internal/evolve`**。那是 L4。
- 不要为了 RSI 把评测器、vault、updater 做成「也可以进化的模块」（Meta-Harness 的文件系统搜索尤其危险）。
- 不要把 Dream-RSI 的世界模型做进桌面编码循环。
- 不要引入向量记忆来「对齐 2025 的 RAG agent」。Barbaste 的语料里生产系统都不用。
- 不要把 `task` 做成无界 harness 递归（RAH）。depth-1 是产品选择。
- 不要用 LLM init 生成长 `AGENTS.md` 当默认（Gloaguen 2026：不涨分、成本 +20%）。

---

## 4. Owner 迭代议程（建议顺序）

按「改变信任根的风险」从低到高。

| # | 动作 | 文献依据 | 风险 |
| --- | --- | --- | --- |
| 1 | Evolve 实验室同时跑 `eval --best N`，并加 IID / SCS 对照，把 token / USD / 墙钟画在一起 | Wang 2026、Gideoni 2026 | 低：只观测 |
| 2 | Proposer 证据改为「失败簇 + 同任务成功对比」；held-out 继续剥离 | ModularRSI、HarnessEvolve | 低 |
| 3 | Canary 增加质量门：playbook 总 token 上限、禁止复制 held-in 指令原文、bullet 去重、禁止仓库概览类针 | HarnessEvolve、ACE、SkillReducer、Gloaguen | 低 |
| 3b | Harbor 报告核对 `predicted_fixes`；对不上的提案记 manifesto miss | AHE | 低 |
| 3c | 加一层秒级行为 eval（「声称完成前必须跑验证器」） | Google 2026-09 | 低 |
| 4 | 进化集与晋升集切开（哪怕只是本地 suite 的再划分） | ModularRSI、Wang、ICLR RSI reward hacking | 中：要重新标定 sealed |
| 5 | LoopPreset 按 ModularRSI 五模块打标签，提案一次只动一个模块，过 Harbor 后再 merge | ModularRSI | 中：要 L3 确认拓扑 |
| 6 | 可选：接入 Harbor-Index 子集作 transfer 门，替代继续堆本地任务 | Harbor-Index 2026-09 | 中：外部依赖 |
| 7 | Skill 执行状态外置（SKILL.state 思想），主对话只看当前 state + 最新观察 | SKILL.state | 高：改 runtime |

完成 1–4 之后，再考虑把 ModularRSI 的外部进化集引进来。不要先上 7。

---

## 5. 前瞻（2026 下半年会打过来的问题）

这些还不是 Yoyo 的实现项，但是 owner 做方向判断时要能回答：

1. **Harness-in-the-loop 学习（RHI）。** harness 不只服务当前用户，还在生产未来模型的训练轨迹。Yoyo 若坚持本地、不训模型，仍然应该让 trace 干净到「将来可以捐给一次共训」。
2. **平台化 / 可托管。** Barbaste：OpenHands 已经在跑别人的 Claude/Codex/Gemini 当 backend。Yoyo 的 JSON-RPC 已经是这个形状。保持「我们是 harness，不是又一个聊天皮肤」。
3. **技能供应链。** SoK 记录的 ClawHavoc 说明 marketplace 默认不可信。Yoyo 的「未签名不能上 active」要保持 fail-closed，即使以后做 skill market。
4. **「harness 效应」可能被高估。** Arjmandi 的同模型配对在平均意义上分不出原生 harness 与中立 harness。Yoyo 的护城河必须是 **可证明、可回滚的进化过程**，而不是「我们的 prompt 比别人神」。
5. **上下文构造成为独立层。** VideoHarness-RSI 把这件事说死了：冻结模型，搜索的是构造程序。`runtime.Shape` 已经是这个层；以后 evolve 可以碰 Shape 策略，但那是 L3。

---

## 6. 下次刷新清单

2026-12-20 或出现以下信号时重跑检索：

- ModularRSI / HarnessEvolve / AHE / Meta-Harness 出现 v2 或正式会议接收
- Harbor 发布独立 verifier sandbox 的稳定 API
- 有论文在 **匹配预算** 下推翻或确认 Wang et al.
- Anthropic / OpenAI / Google / xAI 发布 2026 下半年 Codex/Claude/Gemini/Grok 架构文
- EMNLP 2026、NeurIPS 2026 出程序
- 出现把 **桌面个人 agent**（非 SWE-bench）当作 RSI 对象的工作

刷新步骤：按 [SOURCES.md](SOURCES.md) 扫渠道 → 改 `catalog.yaml` → 新卡片 → 更新本文件第 2–4 节 → 手写更新 [report.html](report.html) → 跑 `scripts/fetch-research-originals.ps1`。只看当年。把满 12 个月的条目降为谱系。
