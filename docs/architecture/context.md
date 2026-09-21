# Yoyo context management

最后审阅：**2026-09-21**。对照的是各产品 **2026 年仍在演进的公开机制**（官方博客 / 文档 / 开源树），不是 2024 年的玩具 demo。

落地状态：**Slice 0–3 已进树。** `Run()` 只消费 `ContextKernel`（`seed` / `prompt` / `checkpoint`）。Chat 前缀 append-only 直到检查点；Dynamic 在 tail；Harbor 仍 Shape-only 且 `ModelWindow=0` 禁止 LLM compact。Playbook 永不进 compact prompt。

本文是工程方案，不是文献卡片。文献谱系仍走 [docs/research](../research/README.md)。Harbor / CAS / L4 不变量见 [invariants.md](invariants.md)。

---

## 0. 判断

Yoyo 已经有一条正确的产品轴线：

> 信任针（pins）每轮重建、永不被摘要；工具抄本是 untrusted working memory；压缩是只读投影；原文进 spill，可 `recall_context` 对证。

这条轴线对齐 Cursor 的 Dynamic Context Discovery、Claude 的「工具结果先牺牲、用户意图后牺牲」、以及 Anthropic 长程 harness「reset ≠ compaction」。**不要拆掉。**

它还不是生产级上下文系统。当前实现是 **loop 里的一串副作用**（assemble → Shape → Legalize → overflow 再 Shape → 启发式 notes），缺少：

1. **前缀契约**（prompt cache 命中是线性成本的前提）
2. **检查点状态机**（只有检查点才允许改写历史前缀）
3. **provider 真值 token**（启发式 token 不能驱动自动压缩）
4. **压缩后回锚**（plan / skills / 热文件必须再注入，否则模型凭记忆重写）
5. **分层记忆**（profile / 仓库 MEMORY / session notes / spill 混在 2000 rune 里）
6. **Harbor 行为门**（没有「压缩后是否还记得目标 / 是否再读热文件」的评测）

同类产品 2026 年已经把上下文当成 OS 的内存管理器：trim → persist → prune → summarise → rehydrate，而且有阈值、事件、溢出恢复和缓存纪律。Yoyo 有 persist（spill）和 trim（Shape），prune/summarise/rehydrate/cache 是半成品。

**重构目标：** 把「上下文」从 `Run()` 的局部变量提升为 **Context Kernel** —— 一个有契约、有账本、有检查点、有评测的子系统。Loop 只消费 `kernel.Prompt()`。Harbor 继续 Shape-only。Chat 走检查点 + 溢出恢复。Playbook 仍然禁止 LLM 重写。

---

## 1. 2026 工业界的共同模式

读 Claude Code、Codex、OpenCode V2、Cursor、Amp Neo、Gemini CLI、Copilot CLI、WorkBuddy 之后，收敛成四条（不是口号，是源码级行为）：

| 模式 | 含义 | 生产级做法 |
| --- | --- | --- |
| **Selection over dump** | 窗口里只放当前任务需要的字节 | Skills 目录静态、正文按需；MCP schema 落盘；长输出进文件 |
| **Append-only until checkpoint** | 两次检查点之间，已发送前缀字节不变 | 新内容只追加；计划/技能变化追加或放到 cache breakpoint 之后 |
| **User intent > tool bytes** | 空间不够时先扔可再生的工具输出 | 用户原话、当前 plan、热文件路径留下 |
| **Rehydrate after compress** | 压缩后不是「摘要 + 空白」 | 重读计划、技能、最近改过的文件；后台任务提醒；spill 指针 |

五条产品分叉轴（2026-05 对六家 CLI 的源码对照，加上 Cursor / WorkBuddy / Amp）：

| 轴 | 光谱 | Yoyo 今天 |
| --- | --- | --- |
| 压缩时机 | Gemini ~50% → Qwen 70% → Copilot 80% 异步 / 95% 阻塞 → Codex / Amp ~90% → OpenCode 溢出缓冲 → Claude 多层无单一百分比 | 启发式 `window - 16k - 13k`；没有百分比；没有异步预压缩 |
| 记忆深度 | OpenCode 无持久记忆 → Markdown 文件（Claude/Codex/Gemini）→ Copilot SQLite+向量+出处 | `memory.json` + `YOYO.memory.md` + session notes，三套未分层 |
| 子代理 | 浅（OpenCode）→ Claude 隔离窗口 → Copilot fleet + 小模型 | `task` depth≤2，但 child `MaxTurns=8` / `MaxToolMessages=12` |
| 可观测 | 弱 → Copilot `session.compaction_*` + OTel + `preCompact` | `ShapeReport` + UI meter；无 cache hit、无压缩质量、无 hook |
| 扩展 | 配置文件 → Claude/Copilot hooks | `.yoyo/hooks.json` 有 pre/stop；无 `preCompact` |

不要抄 Gemini 的 union-find 聚类（仍是实验 flag，默认关）。不要抄 Copilot 的向量记忆当代码检索（Yoyo 已拒绝；代码继续 grep）。不要抄 Codex 的不可逆 `/responses/compact` 当默认（ZDR 对企业有意义；个人工作站要对证 spill）。

---

## 2. 产品对照（只保留对 Yoyo 有约束力的机制）

### 2.1 Claude Code（文档 2026：context-window / memory / prompt-caching）

- 启动就占用窗口：CLAUDE.md、auto memory、MCP **工具名**、skill **描述**。正文不进前缀。
- 压缩是 harness 行为，不是模型自己决定。`/compact [focus]` 可导向。
- 压缩后 **从磁盘再注入**：根 CLAUDE.md、unscoped rules、auto memory、plan、已 load 的 skill（每 skill 5k / 合计 25k，超了丢最老的）、最多 5 个最近改过的文件（>5k token 只留路径）。
- 路径规则和嵌套 CLAUDE.md **故意不**进稳定前缀：读到匹配文件时再加载，压缩后等下次再读。这是 DCD，不是「漏了」。
- Prompt cache：系统 + 工具定义在前；项目针在中；对话在后。计划/技能变化 **追加为对话消息**，不改前缀。MCP 默认 deferred 时，连上/断开不 bust cache。
- 1M 窗口的 context rot 在 30–40% 就开始。更大窗口不是策略。

**对 Yoyo：** 再注入和 cache 分层是 P0。嵌套规则按 DCD 做，不要改成三份 AGENTS/CLAUDE/YOYO 叠进 pins。

### 2.2 OpenAI Codex（2026-01 loop 文 + `compact.rs` / `compact_remote_v2.rs`）

- Responses API。`instructions` + `tools` + `input`。环境/沙箱是 developer/user 项。
- 前缀必须是旧 prompt 的精确前缀，cache 才命中。改 tools 顺序、改模型、改 cwd，一律 miss。中途配置变化 **插入新消息**，不改旧项。
- 自动压缩：`auto_compact_limit`（内部夹到窗口的 ~90%）。`/responses/compact` 返回 `type=compaction` + `encrypted_content`（隐空间）。Remote Compaction V2 走普通 Responses 流，客户端重建历史：保留用户消息（有 token 预算）+ compaction item。
- 压缩前会 trim function_call 历史。Ghost snapshot 留下给 `/undo`。
- 不用 `previous_response_id`（无状态 + ZDR）。

**对 Yoyo：** 隐空间 compact 可做 **可选 adapter**（OpenAI 路由、L3），默认仍是本地 lossless。必须抄的是「配置变化追加、不改前缀」和「保留用户消息预算」。

### 2.3 OpenCode V2（`opencode.ai/v2/docs/compaction`，2026）

- 检查点模型，不再是 V1 的 tail-turn prune。
- 触发：`estimated >= min(input_limit - buffer, context_limit - max(output_reserve, buffer))`。默认 `keep.tokens=15_000`，`buffer=20_000`。
- 溢出：`model call → overflow → compact → retry same step once`。第二次失败才报错。
- 本地摘要有结构标题（Objective / Next Move / …）。最近 `keep.tokens` **原文保留**，与摘要并列。历史仍存在磁盘，只是不再进模型。
- 可走 provider native compact；失败则从 **原始存储历史** 重建再本地检查点。
- 指令更新和压缩分开记账；压缩完成那一刻的指令成为新 baseline。

**对 Yoyo：** 这是 Chat 路径最该对齐的状态机。Yoyo 已有 `PersistCheckpoint` + `MessagesFromEvents` 的 rebuild origin，缺的是 `keep.tokens` 尾、一次溢出重试、以及「检查点之前禁止改写前缀」。

### 2.4 Cursor（DCD 2026-01；Composer self-summarization 2026-03；hooks）

- 长工具输出 → 文件，模型 `tail`/`read`，而不是截断丢失。
- 压缩后把 **完整 chat history 当文件**，摘要里带路径；模型可以 grep 找回。
- MCP：每 server 一个目录，不把 schema 塞进 tools 数组。A/B：用到 MCP 的 run 总 token **-46.9%**。他们明确拒绝扁平 `tool_search` 当主索引。
- 集成终端输出同步为文件。
- Composer 在同一 harness 里 RL 训练 self-summary（约 1k token 摘要，compaction 误差约 -50%）。Yoyo 不训练权重。
- `preCompact` hook 可观测、不可拦截。

**对 Yoyo：** DCD 目录布局优于扁平 `tool_search`。History-as-file 是 spill 的自然延伸。不要做 Composer。

### 2.5 Amp Neo（2026-05 rebuild）

- 压缩优先：约 90% 自动 compact，handoff 从主路径拿掉。
- 线程引用用 `read_thread`（第二个模型抽相关片段），不是把另一个窗口整段贴进来。
- `@file` 有硬截断（约 500 行 / 行 2KB），其余让模型自己读。

**对 Yoyo：** 会话 fork 已有；缺的是「引用另一 thread 并抽取」而不是注入全文。不要把 handoff 做成产品主路径。

### 2.6 Gemini CLI / Copilot CLI（2026 源码对照）

- Gemini：专用压缩模型 `chat-compression-2.5-flash-lite`，约 50% 就压，让主模型远离 rot 区。文件四档 FULL/PARTIAL/SUMMARY/EXCLUDED。
- Copilot：80% 异步压缩 + 95% 阻塞；compaction 事件带前后 token、删了多少、摘要、trace；记忆写带 subject/fact/citations 权限门。

**对 Yoyo：** 异步预压缩和 compaction 事件是 P1。专用小模型压缩是可选（L3），Harbor 禁止。记忆写继续走 `memory_write` 审批（已有）。

### 2.7 WorkBuddy

- ContextCollector → ContextCurator：分源抓取、缓存、depth、`max_chars`。MCP：`context_block` / `context_drill_down`。
- 压缩四层：截断工具 → 同文件读去重只留最新 → 丢旧 turn → LLM 摘要。阈值约 80k（教程实现）。
- 五层记忆：transcript / work notes / project / user / cloud。

**对 Yoyo：** 个人 OS 面（inbox、projects、calendar）不要预渲染成一块巨大 `context_block`（那是反 DCD）。做一个 **有预算的「今日针」**（计数 + 指针，不是正文），细节继续工具拉取。同文件读去重是 P0。

---

## 3. Yoyo 现状（代码事实）

### 3.1 每轮实际组装顺序

`internal/app/session.go` `sendLocked` → `runtime.Run` → `ContextKernel`:

1. `ApplyChatHorizon`（chat：MaxTurns≥64，KeepTokens≥16k，RulesTokens≥8k，子代理 24/60）
2. `MessagesFromEventsOpts`（JSONL；遇 `kind=checkpoint` 从 spill `tail_id` + 摘要重建）
3. `MaybeCheckpoint`（历史已超预算才在 turn 前写检查点）
4. Kernel `seed`：`AssembleIdentity` + `AssemblePins`（playbook / skill catalog / 第一份规则 / profile / MEMORY.md≤4k）
5. Kernel `prompt`：`AssembleDynamic`（plan、loaded skills cap、notes、今日针）→ `RoleMemory` **挂在历史之后**；chat 对 hot **恒等**（只 Legalize）直到检查点；Harbor 每轮 Shape
6. `@` mention 作为 user 注入（总预算 2400 **字节**）
7. `prompt_cache_key` = harness + prefix + tools JSON；`cached_tokens` 写入账本
8. 溢出：checkpoint → 重放当前 step **一次**；第二次失败才 `StopOverflow`
9. 检查点：stale 读去重、结构摘要、回锚最多 5 个热文件、JSONL 存 spill 指针；`pre_compact` 只观测
10. ChatOverlay：`tool_search` 写入 `ExtraLive`，schema 延到下一检查点才进 tools 数组

Harbor 不设 `ModelWindow`，预算钉死 `CompactionTokens`（默认 24k），且 `AllowLLMCompact` 被评测强制关掉。这条必须保持。

### 3.2 已经做对的部分

| 能力 | 位置 | 为什么是对的 |
| --- | --- | --- |
| Pins / working memory 分裂 | `assemble.go` | ACE：摘要 playbook 就是 context collapse |
| Shape 是投影 | `context_shape.go` | 现场 JSONL / live `messages` 不被 Shape 原地改写 |
| Spill-first ingest | `context_ingest.go` | 先落盘再预览，对证优于 Codex 丢字节 |
| 重写调用体 stub | `stubHeavyCalls` | 防「凭记忆整树重写」 |
| 配对边界 snip | `pairingBoundary` + `Legalize` | 不会把半个 tool_call 交给 Chat Completions |
| Skill catalog / `load_skill` | `AssemblePins` | progressive disclosure |
| Chat 工具菜单 + `tool_search` | `chat_menu.go` | 主机工具不全量广告 |
| MCP schemaCap=8 后延迟 | `AllToolJSON` | 粗粒度 DCD |
| `.yoyo/context/<session>/INDEX.md` | `context_discover.go` | 可 grep |
| depth-1 `task` 不回灌 child 抄本 | `tools_task.go` | Claude 子代理模式 |
| 账本 + UI meter | `ShapeReport` / `context-usage.ts` | 操作员看得见 |
| 控制消息不进 notes | `isControlUser` | nudge 不污染 objective |

### 3.3 缺失和不足（按危害，不是按文件）

#### P0 — 会在长任务上系统性失败

**C1. 每轮改写前缀，prompt cache 名存实亡**

`setDynamic` 每轮 **原地替换** pins 后的 working-memory 消息（plan / notes / loaded skills 都在变）。`Shape` 的 microcompact 又改写历史里更早的 `tool_result`。Cache 是精确前缀匹配。结果：identity 也许能命中，**历史几乎每轮全量重算**。Claude / Codex 把这件事当成成本主轴；Yoyo 只发了 `prompt_cache_key`，没有前缀契约测试。

Working memory 的位置是最坏的：夹在 pins 和历史中间。动态块一变，整段历史都 miss。

**C2. Token 真值是错的，窗口目录是过时的**

`ModelContextWindow`：Claude=200k、gpt-5=200k、gemini-2=1M、默认 128k。2026 年 Claude Opus/Sonnet 4.6 与若干 GPT-5.x 已到 1M；Codex CLI 对 GPT-5.5 还夹过 400k。`effectiveBudget` 写死 `16_000` 输出预留 + `13_000` buffer。前端 meter 用「4 字节/token」，Go 侧是另一套 approx-cl100k。自动检查点用的是估计值，不是 `provider_prompt`。预算错了，要么过早 checkpoint（丢细节），要么撞墙 `StopOverflow`。

**C3. 检查点不是状态机，是副作用**

`MaybeCheckpoint` 只在 turn **开始前**看历史是否超预算。Turn 内可以打几十次工具，窗口是在 turn 内爆的（Codex 原文）。溢出路径会写 checkpoint，但成功的 Shape/snip **不**写。OpenCode 是「压缩完重放 **同一个** pending step」；Yoyo 溢出三次后停。

检查点把 **整段 tail** 塞进 JSONL payload。长会话的 event log 会二次膨胀。OpenCode / Claude 存的是指针 + 摘要 + 保留尾，原文另存。

LLM compact（`AllowLLMCompact`）产出的 markdown 没有 Files/Spill 纪律的强制校验，且 `Decisions` 在启发式 `NotesFromMessages` 里 **永远是空的**（只在 LLM 路径才会出现）。默认关 LLM 时，检查点摘要就是「最后一句用户 + 文件路径列表」。

**C4. 压缩后不回锚**

Claude 压缩后重读最多 5 个热文件、再注入 plan 和 skill。Yoyo 的 snip 会在 marker 上列「Recent writes still on disk」，但 **不会 read_file 把内容放回去**。模型在 elide 之后最常见的失败就是凭记忆重写——`rewriteStallNudge` 是事后补丁，不是回锚。

**C5. `recall_context` 是上下文炸弹**

`noStub=true`，一次召回最多 16k rune，并且这笔内容会留在 live transcript 里直到以后被 microcompact。没有 offset/limit，没有「只读 spill 的某一段」。Cursor 的 page-fault 是 `tail` + 分段 read。Yoyo 的 page-fault 是把整页重新装进 RAM。

**C6. Session notes 语义是错的**

`Objective` = 最后一条非控制 user 消息，不是任务目标。多轮「再改一下」会覆盖原始目标。`Decisions` 不提取。`Files` 是 tool 参数里的 path 并集，不是「当前磁盘上这些文件的哪一版还有效」。`persistWorkingMemory` 把 notes **和** `YOYO.memory.md` 拼进 2000 rune——仓库记忆和会话草稿抢同一预算，长 MEMORY.md 会把 notes 挤掉。

#### P1 — 生产产品已经有、Yoyo 只有雏形

**C7. 规则发现是 first-hit，不是层次 + DCD**

`LoadWorkspaceRules` 在 `YOYO.md` / `AGENTS.md` / `CLAUDE.md` / `.cursor/rules/yoyo.mdc` 里取 **第一份**，1500 rune。Codex 从 git root 走到 cwd 堆叠 AGENTS.md（默认 32KiB）。Claude 拼接树，嵌套文件按读路径加载。Yoyo 拒绝「三份全文叠针」是对的，但因此也没有：

- root → cwd 的 AGENTS.md 层次
- 路径作用域规则
- 子目录 CLAUDE.md 的按需加载

1500 rune 对真实仓库公约过小。

**C8. DCD 是扁平索引，不是文件树**

Cursor 按 MCP server 分目录，模型 `ls`/`rg`/`jq`。Yoyo 把延迟工具名 **写进 `tool_search` 的 description 字符串**，`tool_search` 命中后 **改 Advertised**——这会改 tools 数组，正是 Codex 说的 cache miss。`.yoyo/mcp/INDEX.md` 有了，但模型没有被强制「先读目录再启用」，启用还会污染前缀。

**C9. 没有同文件读去重 / 过期失效**

WorkBuddy L2、Gemini 的文件档位：同一 path 读十次只留最新。Yoyo 每次 `read_file` 都进抄本。磁盘上文件被 `str_replace` 之后，旧 read 仍在窗口里，模型和磁盘不一致——这是「凭记忆编辑」的另一来源。

**C10. 子代理窗口是玩具预算**

`spawnTask`：`MaxTurns=8`，`MaxToolMessages=12`，`AllowLLMCompact=false`，`ChatOverlay=false`。Claude 把「大文件研究」丢给子代理，正是因为 child 窗口要够用。8 轮不够 grep 一个中型包。没有 explore / implement / qa 人格，没有独立 evaluator（Anthropic 2026-03 长程文的核心）。

**C11. 记忆三套平行，没有晋升语义**

| 层 | 现在 | 问题 |
| --- | --- | --- |
| Profile | `memory.ProfilePin(800)`，staging 需人 promote | 对，但不会自动从会话提炼 |
| 仓库 MEMORY.md | 塞进 notes 2000 | 应是 pin 或 DCD 文件，不该和 session notes 抢预算 |
| Session notes | spill `notes` | 启发式，且 objective 被覆盖 |
| `memory_search` | 工具 | 不会按当前任务意图召回（研究前沿；不要假装有） |

Claude auto-memory：`MEMORY.md` 前 200 行或 25KB，超限会让模型把细节拆到 topic 文件。Codex 用会话结束后的子代理做记忆巩固（不占热路径）。

**C12. 可观测不够用**

没有：cache read/write tokens、压缩前后 token、压缩是否导致后续 `read_file`/`recall_context`、`preCompact` 事件。Copilot 和 Claude `/context` 已经把这当成操作员工具。Yoyo 的 meter 只有 slices，没有建议。

#### P2 — 个人 OS 与长程质量

**C13. 没有「今日针」也没有终端文件**

Inbox / calendar / 活动项目要么不在窗口里，要么模型自己工具捞。Cursor 把终端历史当文件；Yoyo 的 shell 输出走 ingest，GUI 终端不同步。个人 OS 的优势没有变成 **预算内的指针层**。

**C14. 没有 rewind / focused compact / 跨 thread 抽取**

Claude `/rewind` + summarize from here；Amp `read_thread`；Yoyo 有 fork 和 `CompactHistory`。操作员不能说「从这条开始摘要，聚焦 auth」。

**C15. 模型协议停在 Chat Completions**

没有 Responses 的 reasoning item / encrypted compaction。对 OpenAI 官方模型这是质量损失；对兼容网关这是正确的最小公约。Adapter 必须是 L3，默认路径不能绑死 Responses。

**C16. Harbor 不评测上下文**

现有 eval 是终局文件/安全。没有行为断言：压缩后是否还记得 objective、是否再读热文件、是否调用 `recall_context` 而不是默写、是否在 elide 后 `write_file` 整文件。Google 2026-09 harness engineering 的微观断言正是给这个用的。

---

## 4. 明确保持 / 明确拒绝

保持：

- Shape 对 live transcript 只读；JSONL 不因 Shape 截断
- Playbook / policy / vault 禁止进 compact prompt
- Harbor：`CompactionTokens` 钉死，禁止 LLM compact，禁止 `ModelWindow` 改评测预算
- 代码检索继续 grep/glob，不上向量库
- 前端不出现第二套 agent loop
- 不训练 Composer
- 不把三份规则文件全文叠进 pins

拒绝（即使竞品在做）：

- 默认不可逆丢工具字节（Codex compact / 纯 LLM 摘要替换历史）
- 默认 Gemini union-find 聚类压缩
- 默认 50% 就 LLM 摘要（浪费钱且 Harbor 不可复现）
- 用 LLM 自评替代 Harbor
- 无界多小时无人值守改用户机器来「治 context anxiety」

---

## 5. 目标架构：Context Kernel

Loop 不再自己拼消息。每轮：

```
pins, dyn, hot, tools, ledger = kernel.Assemble(turn)
prompt                    = kernel.Legalize(pins + [dyn@tail] + hot, tools)
if kernel.ShouldCheckpoint(ledger) { kernel.Checkpoint(reason); retry this step }
send(prompt)  // 前缀契约：与上一轮相比只允许 suffix 增长，除非刚 checkpoint
on overflow:    kernel.Checkpoint("overflow"); retry once; else StopOverflow
```

### 5.1 四层，预算显式

```
┌──────────────────────────────────────────────────────────┐
│ L0 Pins (cache-stable, 每会话冻结直到 checkpoint)          │
│  identity · playbook · skill catalog · rules · profile   │
│  advertised tool schemas（本会话冻结；DCD 启用延后到检查点） │
├──────────────────────────────────────────────────────────┤
│ L1 Hot transcript (append-only)                          │
│  user / assistant / tool_result(ingest 预览) · nudges    │
├──────────────────────────────────────────────────────────┤
│ L2 Dynamic tail (cache breakpoint 之后，允许每轮变)        │
│  plan · loaded skills · session notes · 今日针指针        │
├──────────────────────────────────────────────────────────┤
│ L3 Cold (不进模型，除非 page-fault)                        │
│  spill/ · history.jsonl · mcp/<server>/ · terminals/     │
│  .yoyo/context/<id>/{INDEX.md,notes.md,HOT.md}           │
└──────────────────────────────────────────────────────────┘
```

**把 Dynamic 挪到历史之后。** 这是成本最大、代码最小的一次结构修复。pins+hot 才能成为稳定前缀。

预算分配（chat，按窗口比例，Harbor 仍用绝对 `CompactionTokens`）：

| 桶 | Chat 默认 | 硬上限 |
| --- | --- | --- |
| Pins | ≤ 8% | 12k tokens |
| Tool schemas | ≤ 8% | 12k tokens |
| Dynamic tail | ≤ 6% | 8k tokens |
| Hot transcript | 其余 − 输出预留 | — |
| 输出预留 | max(8k, 10%) | 32k |
| 触发检查点 | used ≥ min(input_limit − 20k, context − 输出预留) | 与 OpenCode 同构 |
| 保留尾 `keep.tokens` | 16k（chat）/ 评测用 CompactionKeep | L3 |

Token 真值：有 `provider_prompt` 就用；否则 tiktoken-class 估计；禁止前端 4B 启发式驱动压缩。窗口目录按模型 ID + 可选 `config.context_window` 覆盖（网关别名）。

### 5.2 两条时间轴

**Ingest（每条 tool_result 一次，之后不变）**

1. 全文进 spill（已有）
2. 预览：头+尾，预算 `ToolResultBudget`
3. 指针：`id` + `name` + `bytes` + `path?`
4. `read_file` 预览带 mtime/size；同 path 的旧 preview 标 `stale`（检查点时物理丢掉）

**Checkpoint（唯一允许改写前缀的时刻）**

触发：预检超阈值、provider overflow、用户 `/compact [focus]`、任务边界（plan 全部 complete 且 operator 新开主题——可选）。

动作：

1. 发 `context.checkpoint_start`（token 前后、trigger、是否第一次）
2. 确定性层：去重同 path 读、stub 重写调用、丢掉 stale preview、保留用户消息和最近 `keep.tokens`
3. 结构摘要：默认启发式 SessionNotes（修 C6）；`AllowLLMCompact` 时才 LLM，输出必须含 `## Objective/Files/Decisions/Errors/Next`，否则一次纠正，再失败则只用启发式
4. 回锚：plan 全文、loaded skills（Claude 同款 cap）、最多 5 个热文件（>N token 只留路径）、spill INDEX、后台 task id
5. JSONL 只存：summary + kept 的 **id 列表** + `keep.tokens` 原文（或 spill 指针）。禁止再把整段 tail 嵌进 event
6. 从检查点重建 hot；**重放当前 pending model step 一次**（OpenCode）
7. 发 `context.checkpoint_complete`

Harbor 永不走 3–7 的 LLM 分支；评测继续 `Compact()`。

### 5.3 Page-fault

| 工具 | 行为 |
| --- | --- |
| `recall_context` | **必须** `offset/limit`（rune 或 line）。默认 200 行。禁止 `noStub` 把 16k 钉进 hot |
| `read_file` | 已有 offset/limit；检查点后自动对热路径调用（内核做，不指望模型记得） |
| `grep` `.yoyo/context/<id>` | 已提示；INDEX 增加 `history.jsonl` 与 `terminals/` |
| MCP | `tool_search` 只返回路径；启用写入 `ExtraEnabled`，**schema 延到下一检查点才进 tools 数组**；本轮用「单次临时 tool」或让模型 `read_file` schema 再在下轮用 |

### 5.4 规则与记忆分层

**规则**

- Pins：仅 **cwd 向上到 workspace root 的第一份** `YOYO.md` 或 `AGENTS.md` 或 `CLAUDE.md`（保持不叠三份），预算提到 8k rune（L3）
- DCD：`.yoyo/rules/` 与嵌套 `**/CLAUDE.md` 写入规则索引文件，模型读到匹配 path 时再 load（Claude paths frontmatter 同构）
- 层次 AGENTS.md（root→cwd）作为 **可选** L3：默认关，打开时按 Codex 32KiB 总帽拼接，且只在会话开始 / 检查点重读（避免每轮 bust cache）

**记忆**

| 层 | 进窗口方式 | 谁写 | 审批 |
| --- | --- | --- | --- |
| Profile | Pins，≤800B，非 staging | 人 promote | 已有 |
| Project MEMORY.md | Pins 独立槽，≤4k rune，**不进 notes** | 人 / 模型 `memory_write` | `memory_write` |
| Session notes | Dynamic tail，修 C6 | 内核每检查点 | 无（untrusted） |
| Episodic | 冷；`memory_search` | 会话结束可选子代理巩固（Codex 模式，P2） | 写要审批 |
| Spill | 冷 | 内核 | 无 |

Objective 规则：第一条非控制、非 resume 的 operator 消息冻结为 Objective，直到用户显式新开任务（新 thread 或 `/clear` 语义）。后续用户话放 `Decisions`/`Next`，不覆盖 Objective。

### 5.5 子代理

`LoopPreset` 增加 L3 字段（人确认才进 topology）：

```
task_max_turns            default 24   // 不再写死 8
task_max_tool_messages    default 60
task_profiles             explore | implement | qa
```

- `explore`：只读 + 大窗口，回传结构化发现（路径、符号、风险），禁止把文件正文带回 parent
- `implement`：当前 child 行为，但预算对齐 chat horizon 的一半
- `qa`：独立 evaluator 人格（Anthropic generator/evaluator）。只在长程办公/多文件任务由 plan 或操作员启用。默认关。

Parent 仍然只收 `SUBAGENT_SUMMARY`。Child spill 隔离已有，保留。

### 5.6 前缀契约（测试即规范）

Harbor 外的 chat 必须有这些测试，失败即禁止合并：

1. `TestPrefixStableAcrossToolRounds`：连续 8 个 tool round，`wireMessages(pins+hot)` 的字节前缀等于上一轮（只允许 suffix 增长）。Dynamic 在 tail，允许变。
2. `TestCheckpointBreaksPrefixOnce`：检查点后允许整段重写一次，下一轮重新稳定。
3. `TestToolUnlockDoesNotMutateSchemaUntilCheckpoint`：`tool_search` 启用 MCP 不改变本轮 `tools` JSON。
4. `TestRecallIsPaged`：`recall_context` 不把全文写进下一条 tool_result。
5. `TestObjectiveSticky`：第二条用户消息不覆盖 notes.Objective。
6. `TestHotFileHydrate`：检查点后 prompt 含最近写入 path 的新内容或 `Referenced file` 指针。
7. `TestHarborShapeUnchanged`：`AllowLLMCompact=true` 时 Harbor 仍不调用 LLM compact。

Cache 观测：解析 provider `cached_tokens` / Anthropic `cache_read_input_tokens`，写入 `ShapeReport` 和 UI meter。没有字段就显示「provider 未报告」，禁止假数。

---

## 6. 落地迭代（按可晋升切片，不是大爆炸）

不变量 10：loop **文案** 是 L1；compaction 拓扑（阈值、keep.tokens、是否 native compact、子代理预算）是 **L3**。Kernel 骨架是 Go，属 L4 外围但仍是内核代码——evolve 不得改。每一切片都要有测试 + 至少一条 Harbor 行为题（P1 起）。

### Slice 0 — 契约与账本（约 1 周）

把 `assemble` / `Shape` / `checkpoint` / `notes` / `spill` / `measure` 收到 `internal/runtime/ctxkernel`（或 `runtime` 内明确的 `ContextKernel` 类型）。**行为先不改**，只让 `Run` 只调 `kernel.Turn()`。加 C1 的前缀测试（预期现在失败，作为红灯）。

交付：`ShapeReport` 增加 `cache_stable bool`、`prefix_hash`、`dynamic_at string`（`mid`|`tail`）。

### Slice 1 — P0 正确性（约 2–3 周）

顺序固定，不要并行开 LLM compact：

1. Dynamic 移到 tail（C1）
2. Ingest 之后 Shape 对 hot **恒等**，直到 checkpoint（C1）
3. 窗口目录 + `config.context_window` + 比例预算；meter 与 Go 共用同一估计器；优先 provider 真值（C2）
4. `recall_context` 分页；去掉 `noStub` 或仅对 ≤2k 的 spill 生效（C5）
5. Notes：Objective 冻结；MEMORY.md 从 notes 预算拆出（C6）
6. 同 path 读：检查点时去重；ingest 带 mtime，写后标 stale（C9 的一半）
7. 检查点：JSONL 存指针；溢出 → checkpoint → **重试当前 step 一次**（C3）
8. 回锚 5 热文件 + plan + skills cap（C4）

完成标准：Slice 0 的前缀测试变绿；人工跑一个 40+ tool 的桌面任务，cache read 占比在支持的 provider 上可见（或诚实显示未报告）；溢出不再三次致盲失败。

### Slice 2 — 生产循环（约 3–4 周）

1. `/compact [focus]`、检查点事件、UI 显示 trigger/layers/cache
2. MCP 每 server 目录为主，`tool_search` 为辅；启用延迟到检查点（C8）
3. 规则：pins 8k + 规则索引 DCD（C7）
4. 子代理预算 L3 字段 + `explore` profile（C10）
5. `preCompact` hook（观测，不可拦截）
6. Harbor 行为题：`context-retain-objective`、`context-reread-after-compact`、`context-no-rewrite-from-memory`（已有 rewrite stall，升级成 eval）

可选 L3：OpenAI native compact adapter。失败回退本地检查点（OpenCode 同构）。默认关。

### Slice 3 — 个人 OS 与长程（约 4+ 周）

1. 今日针：inbox 未读计数、下一个 schedule、活动 project 名，≤400 token，全是指针
2. 终端 session 文件化
3. 会话结束记忆巩固子代理（写 staging，人 promote）——对齐 Codex，不占热路径
4. `read_thread`：引用另一 session，子模型抽取，不贴全文
5. `qa` evaluator 人格，默认关
6. rewind/summarize from here

### 不在本方案里做

- 向量记忆检索代码
- 默认开启 LLM 摘要
- 把 Context Kernel 放到 frontend
- 为 1M 窗口关掉检查点（rot 在 30% 就开始）

---

## 7. 操作员可见变化（产品，不是内部重构）

做完 Slice 1–2，桌面应具备：

- Context meter：system / tools / chat / dynamic / free + **cache hit**（有则显示）
- 压缩发生时 transcript 里一条 **过去时** 过程行：`Checkpoint · kept 16k · elided 42 · hydrated 4 files`，可展开看 summary
- Composer 可 `/compact focus on …`（或菜单：Compact this thread）
- Review → Trace 能滤 `compact` / `overflow` / `hydrate`
- 操作员改 MCP / 工具菜单：**本轮不改 schema**，提示「将在下一检查点生效」

这与 PRODUCT.md 一致：GUI 是 Go loop 的客户端，不在 React 里做第二套压缩。

---

## 8. 为什么这是生产级而不是 toy

Toy 方案：调大 `CompactionKeep`、把 notes cap 改成 8k、溢出再 Shape 两次。窗口更大、失败更晚、cache 仍然每轮 miss、压缩后仍然凭记忆重写。

生产方案的硬条件：

1. **有不可破坏的前缀契约，并且有测试锁住**
2. **只有检查点改写历史，检查点可回放、可对证、可评测**
3. **Token 用 provider 真值驱动策略，启发式只是后备**
4. **压缩是「丢可再生字节 + 回锚不可再生状态」，不是「摘要替代现实」**
5. **Harbor 继续确定性；chat 的 LLM 摘要是 opt-in 且失败有退路**
6. **拓扑变更走 L3；evolve 只能改 compact 文案（L1）**

Yoyo 的护城河仍然是 CAS + Harbor + lossless spill。Context Kernel 是让这条护城河在 200k–1M 窗口、几十到上百 tool round 的真实会话里 **还成立**。
