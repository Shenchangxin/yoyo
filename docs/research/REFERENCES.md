# Yoyo 引用与落地说明

本文回答：我们到底站在谁的肩膀上，代码里哪一段是那只肩膀，以及哪些流行做法我们明确拒绝。

对应前沿排序与 owner 下一步见 [FRONTIER.md](FRONTIER.md)。扫读：[report.html](report.html)。卡片与原文见 [corpus/](corpus/) 和 [originals/](originals/)。

审阅日期：2026-09-20。

---

## 1. 一张图：文献 → 机制 → 代码

Yoyo 把智能体拆成 **冻结的模型 + 可进化的 harness + 冻结的评测器**。进化对象是 CAS 里的快照，不是 Go 内核。

```
Weakness mining ──► bounded L1 proposal ──► Harbor regression ──► refs/canary
        │                     │                      │
   Self-Harness          ACE deltas              ShouldPromote
   (Zhang 2026)          (Zhang 2025/ICLR 2026)  (Self-Harness gate)
        │
        └── parent sampled from archive  (DGM, Zhang 2025 — lineage)
```

晋升通道：

| 层 | 可变对象 | 谁可以改 | 学术对应 |
| --- | --- | --- | --- |
| L1 | prompt fragments、playbook bullets、skills、loop 文案 / 声明式 middleware | evolve，必须过 Harbor | Self-Harness proposal; ACE curator |
| L2 | 签名 WASM 工具 | admit + HighRisk | 技能/插件信任分层（SoK Skills, Barbaste 2026） |
| L3 | loop 拓扑、policy pack | 人确认 | HSI「固定 injection seam」；Claude 权限模式 |
| L4 | Go 内核、eval、vault、updater keys | **没人** | HSI 的 frozen outer anchor；Gödel 机不可证自改的工程替代 |

---

## 2. 已经写进内核的文献

### 2.1 Self-Harness（Zhang et al., 2026）— 主算法

- 卡片：[corpus/2606.09498-self-harness.md](corpus/2606.09498-self-harness.md)
- 主张：固定模型，智能体改进自己的 operating harness。三步：**Weakness Mining → Harness Proposal → Proposal Validation**。晋升要求 held-in / held-out 不回退。
- Yoyo 落地：
  - `internal/evolve/mine.go` — 按 verifier cause × causal status × agent mechanism 聚类失败轨迹。
  - `internal/evolve/propose.go` — 宽度 `K` 的有界 L1 提案（prompt / playbook / skill / instruction / middleware），禁止改控制环与评测器。
  - `internal/eval/promote.go` `ShouldPromote` — held-in 与 held-out 都不能降，至少一边升；`safety_fail` 一票否决。
  - `internal/evolve/engine.go` `Cycle` — 评基线 → 挖证据 → 反射/策展 → 提案 → 隔离 worktree 里跑 Harbor → 默写 `refs/canary`。
- 我们比论文更严的地方：evolve **默认不移动** `refs/active`（不变量 9）。论文把通过回归的编辑合并进下一版 harness；Yoyo 把「通过」和「成为你信任的那一份」拆开，中间是人的 checkout / `--promote`。
- 模型特异性：`refs/models/<fingerprint>/active` 与论文「harness 必须按模型定制」一致。

### 2.2 ACE（Zhang et al., ICLR 2026；arXiv 首发 2025-10）— playbook 增量

- 卡片：[corpus/2510.04618-ace.md](corpus/2510.04618-ace.md)（**谱系**，不是 2026-first）
- 主张：上下文是会生长的 playbook。Generator / Reflector / Curator。增量 delta，禁止整篇重写，以对抗 brevity bias 与 context collapse。
- Yoyo 落地：
  - `internal/evolve/curate.go` — `Curate` 只 ADD 与 Helpful/Harmful 计数，注释写明 never rewrite the playbook as a blob。
  - `App.StageACE` — 在线路径只写 `refs/staging`。点赞同样。
  - `runtime.AssemblePins` — 每轮按 `helpful − harmful` 预算渲染 playbook；工具抄本是 untrusted working memory，`Shape` **绝不摘要 playbook**。
- 威胁模型第一条对应 ACE 失败模式：「Playbook full-rewrite (context collapse)」。

### 2.3 Darwin Gödel Machine（Zhang / Clune et al., 2025）— 档案采样

- 卡片：[corpus/2505.22954-dgm.md](corpus/2505.22954-dgm.md)（**谱系**）
- 主张：不要只改当前最优。维护 archive，按质量与开放度采样父代，长出多样的改进树。经验验证替代「可证有益」。
- Yoyo 落地：
  - `internal/evolve/archive.go` — `nodes.json` 父子节点。
  - `internal/evolve/select.go` — `ParentWeight = score / (1+children)`，接受过的节点加权，transfer 失败降权。
  - 快照带 `parent`；lineage 由 refs + 这条链重建（`internal/app/harness_lineage.go`）。
- 我们拒绝 DGM 的哪一点：DGM 进化的是 **智能体代码本身**。Yoyo 的进化对象永远是 CAS 快照；评测器不能当父代（`SelectParent` 注释）。这是对 Gödel 自指的工程否决，不是疏忽。ICLR 2026 的 [HGM](corpus/2510.21614-hgm.md) 把「看后代质量」说得更清楚（CMP），采样思想上是 `ParentWeight` 的亲戚；改 Go 仍然禁止。

### 2.4 Terminal-Bench 2.0 + Harbor（Merrill et al., ICLR 2026；Harbor-Index, 2026-09）

- 卡片：[corpus/2601.11868-terminal-bench.md](corpus/2601.11868-terminal-bench.md)、[corpus/2609.04298-harbor-index.md](corpus/2609.04298-harbor-index.md)、[products/harbor.md](products/harbor.md)
- 主张：有容器、有人写的参考解、有测试的终端任务，才叫可验证的 agent 评测。Harbor 把任意 agent 接到同一任务布局；verifier 应与 agent 隔离。
- Yoyo 落地：
  - `docs/architecture/harbor.md` — `instruction.md` / `task.toml` / `tests/` / 可选 `environment/Dockerfile`。
  - `evals/` 与 `evals/yoyo-agent/` 入口。
  - smoke vs sealed：held-out / transfer **身份在 suite 对象上，永不进 proposer 证据包**（不变量 6）。
  - 候选在 **detached git worktree** 里跑，不脏工作树。
- 尚未落地：Harbor-Index 82 题作为晋升门；独立 verifier sandbox（Harbor 2026 新特征）。

### 2.5 配置机制实证（Galster et al., 2026-02）

- 卡片：[corpus/2602.14690-harness-config.md](corpus/2602.14690-harness-config.md)
- 主张：仓库级 harness 有八种旋钮。实践上 Context Files 一家独大，`AGENTS.md` 是跨工具公约。
- Yoyo 落地：`LoadWorkspaceRules` 读取 `YOYO.md` / `AGENTS.md` / `CLAUDE.md`（有 token 上限），每轮重建进 pins。Skills 是 CAS 对象，不是散落的 markdown 希望。

---

## 3. 已经写进内核的产品报告

### 3.1 OpenAI Codex：agent loop + App Server（2026）

- 卡片：[products/openai-codex.md](products/openai-codex.md)
- 核心思想：
  1. **Harness 是 loop**：模型要么给最终 assistant message，要么要工具；工具输出回灌再采样，直到 `done`。
  2. **一个 core，多个 client**：CLI / IDE / 桌面都通过双向 JSON-RPC 连同一套 harness。Server 可以反向请求 approval 并暂停 turn。
- Yoyo 落地：
  - `internal/runtime/loop.go` — 同一套 ReAct 循环服务 CLI、HTTP、Wails。
  - `internal/api/rpc.go` + `yoyo serve --stdio` + `/api/ws` — `thread.*`、`turn.start`、`item.event`、`approval.request`。
  - `main.go` 在 `YOYO_WORKER=1` 时变成 worker；桌面 UI 不跑第二套 agent loop（PRODUCT.md 非目标）。
- 我们没照抄：Codex 的 compaction 是不可逆压缩。Yoyo 的 `Shape` 是只读投影，原文进 spill，可用 `recall_context` 取回。

### 3.2 Claude Code：权限、压缩、skills、subagent（2026）

- 卡片：[products/anthropic-claude-code.md](products/anthropic-claude-code.md)、[corpus/2604.14228-claude-code.md](corpus/2604.14228-claude-code.md)
- 核心思想：真正的代码量在 loop 周围 — 七种权限模式、分层 compaction、MCP / plugins / skills / hooks、隔离 subagent。
- Yoyo 落地：
  - `internal/capability` — Once / Session / Always；`send_as_you` / computer use 是 NeverAlways。
  - `internal/runtime/context_shape.go` — legalize → budget/spill → microcompact → forceFit → snip。LLM 摘要不是 Shape 的一部分。
  - Skills：目录只注入 catalog line，`load_skill` 才加载正文（progressive disclosure）。
  - `task` 子智能体 **depth-1**。
- 我们拒绝：把审批交给隐藏的 ML classifier；在前端再做一个 LangChain / Vercel AI 循环。

### 3.3 Cursor Composer 2（2026-03）

- 卡片：[corpus/2603.24477-composer-2.md](corpus/2603.24477-composer-2.md)
- 核心思想：**在部署用的同一套 harness 里训练**，减少 train–test mismatch。模型与 harness 共同进化。
- Yoyo 立场：本产品不训练权重。RHI（Lee et al., 2026）把 harness 看成未来模型的数据发动机 — 我们只取「trace 要可回放、可审计」这一半：JSONL session、hash-chain journal。不把 RSI 做成静默改模型。

### 3.4 Wails v3

- 卡片：[products/wails-v3.md](products/wails-v3.md)
- 用途：原生窗口、菜单、托盘、通知。**不是**第二套 runtime。GUI 是 Go loop 的客户端（PRODUCT.md）。

### 3.5 十一套生产 harness 解剖（Barbaste et al., 2026）

- 卡片：[corpus/2609.00006-harness-anatomy.md](corpus/2609.00006-harness-anatomy.md)
- 我们与该审计的交集（他们的 13 条观察里，Yoyo 已经站边的）：
  - 手写 async loop，不引入通用 agent 框架。
  - 检索是 grep / glob，不是向量库。
  - SKILL.md 与 MCP 并存；技能有信任分层（签名 WASM / 未签名不能上 `refs/active`）。
  - 一个 core 可被多种 client 托管（CLI / HTTP / Wails），接近他们说的「harness 变成平台」。

---

## 4. 明确拒绝或尚未采用的思想

| 思想 | 来源 | 为什么现在不采用 |
| --- | --- | --- |
| 进化智能体源码本身 | DGM / Gödel Agent | L4 禁止。可回滚的是快照，不是 `argv[0]` |
| 用更强外部模型当 Meta-Harness | Self-Harness 论文对比的范式 | 产品目标是 **self**-harness，同一模型改自己的脚手架 |
| 在评测集上直接进化再报同一套分数 | Wang et al. 2026 批评的协议 | 已有 held-out / transfer；下一步要 **benchmark-disjoint** 进化集（ModularRSI） |
| 复杂进化但不跟随机采样 / best-of-N 比预算 | Gideoni et al. 2026；Wang et al. 2026 | evolve 实验室必须先画出匹配预算图 |
| LLM 生成的长 AGENTS.md / 仓库概览 | Gloaguen et al. 2026 | 针只写非标准惯例；有 token 上限 |
| 无界 harness 递归 | RAH 2026 | `task` 保持 depth-1 |
| 进化进化程序本身 | AEvo / HSI meta-evolver / HGM 改源码 | L4 |
| 整页重写 system prompt | ACE 反模式 | curator 只接受 delta |
| 向量记忆当代码检索 | Barbaste 2026 观察：生产系统都不用 | 继续 grep |
| 前端第二套 loop | Codex / PRODUCT.md | 禁止 |
| 静默覆盖运行中的二进制 | 更新器设计 | `yoyo update apply` 是人的动作 |

---

## 5. 机制对照表（给 owner 扫）

| Yoyo 机制 | 主要来源 | 实现 | 状态 |
| --- | --- | --- | --- |
| Mine → Propose → Harbor | Self-Harness 2026 | `internal/evolve`, `internal/eval` | implemented |
| Playbook delta / 在线 staging | ACE ICLR 2026 | `curate.go`, `StageACE` | implemented |
| Archive 父代采样 | DGM 2025 | `select.go` | implemented |
| held-in / held-out / safety / transfer | Self-Harness + Harbor | `promote.go`, sealed suite | implemented |
| CAS + refs | Git 内容寻址（工程，非论文） | `internal/artifact` | implemented |
| 只读 Shape + spill | Claude compaction 的反设计 | `context_shape.go` | implemented |
| JSON-RPC App Server | Codex 2026 | `internal/api/rpc.go` | implemented |
| Skills catalog | Claude + SoK Skills 2026 | `artifact.Skill`, `load_skill` | partial |
| 模块化 RSI（五模块独立进化） | ModularRSI 2026-09 | — | candidate |
| 参考轨迹对齐 | HarnessEvolve 2026-09 | — | candidate |
| 与 test-time scaling 的匹配预算对照 | Wang et al. 2026-07；Gideoni et al. 2026-02 | — | critique |
| Harbor-Index 门 | Shi et al. 2026-09 | — | candidate |
| 技能可变执行状态 | SKILL.state 2026-08 | — | candidate |
| 文件系统证据 / 可证伪 change manifest | Meta-Harness ICML 2026；AHE 2026-04 | journal + `PredictedFixes` 字段已有，未对账 | candidate |
| 规则裁剪先于 LLM 摘要 | Fan et al. 2026-09 | `context_shape.go` | partial |
| 微观行为评测 | Google 2026-09；RHB ICML 2026 | — | candidate |
| 长程 generator/evaluator 拆分 | Anthropic 2026-03 | Plan mode 有；独立 QA 人格无 | candidate |

Git 式 CAS/refs 没有 2026 年论文可引，它是工程类比（Git objects / refs）。学术上对应的是「harness 必须是声明的、可差分的状态」，Self-Harness 与 Barbaste 都把 harness 定义成这份状态。
