# Yoyo Buddy：外部 agent 当工人，多 agent 协作

最后审阅：**2026-11-09**。

落地状态：**P0–P10 已进树。** 任何支持 ACP v1 的本机 agent（Claude Code、Codex、Gemini CLI、Qwen Code、自定义命令……）都能在 Buddy 里装好、配好、登录，在线后接 yoyo 派的工单。yoyo 自带工人继续是默认工人和兜底。安全规则见 [invariants.md](invariants.md) 第 21–24 条和 [threat-model.md](threat-model.md) 的 Buddy 一节；日志见 [logging.md](logging.md)。

设计借鉴了 Multica 的思路（工人可替换、协调者唯一、共享工作区），没有复制它的代码或文案。

---

## 0. 判断

yoyo 的会话已经是「路由 + 工人」：人只跟主会话说话，路由把事情开成工单，执行器派给工人。要让工人可以是任何 agent，改动只在三处：

1. **工人缝**：执行器只认 `superagent.Worker` 接口，yoyo 自带工人（dsh）和外部工人（acpworker）是它的两种实现。
2. **控制面**：外部 agent 不在 yoyo 进程里，回报进度要走 MCP。yoyo 自己当 MCP 服务器，只给这张工单的 `orchestrator_*` 工具。
3. **路由的工具**：`task_route` 多一个 `agent` 字段；多个 agent 一起做时用 `task_split` 开任务组，任务组停下时叫醒路由。

工人之间不直接对话。跨成员的东西放进任务组共享目录和 `NOTES.md`，协调只由路由做。

---

## 1. 分层

| 层 | 位置 | 拥有 |
| --- | --- | --- |
| 下载 | `internal/fetch` | 只走 HTTPS、校验 sha256、解压、镜像回退（没有 sha256 不走镜像）、去掉下载标记 |
| 进程组 | `internal/procgroup` | 整棵进程树一起停；父进程退出时子进程跟着退 |
| ACP 客户端 | `internal/acp` | 按换行分隔的 JSON-RPC 2.0；握手、会话、提示、取消、权限、fs、登录、续接；`acptest` 是同进程的假 agent |
| Buddy 管理器 | `internal/buddy` | 目录合并、检测、托管或系统安装、配置和密钥、登录、探测与健康检查、状态、磁盘、打码 |
| 控制面 | `internal/mcpserve` | 本机回环 MCP 服务器（Streamable HTTP 子集）和 stdio 桥 |
| 外部工人 | `internal/acpworker` | 把一个 ACP 会话包成 `superagent.Worker`：日志归一化、审批桥、目录限制、续接、崩溃重连、用量、单轮超时 |
| 引擎 | `internal/superagent` | 工单的 agent、改派与交接、任务组（分工和对比）、依赖放行与挂起、唤醒 |
| 接线 | `internal/app` | `buddy.go`（管理器和 RPC 包装）、`buddy_worker.go`（Opener、审批、写入检查、控制面）、`route_*.go`（路由工具、@ 提及、唤醒）、`buddy_diag.go`（诊断和 Doctor） |
| 界面 | `frontend/src/features/buddy`、`superagent`、`transcript` | Buddy 栏目、任务栏目的任务组、主会话的任务组卡片和唤醒回执 |

---

## 2. 主线

1. 人在输入框里 `@` 一个或几个 agent，消息里带 `@agent:<id>`。
2. 路由用 `task_route`（一个 agent）或 `task_split`（几个 agent）开工单或任务组。
3. 执行器按每次派工的 agent 选工人：`yoyo` 走 dsh，外部 agent 走 acpworker。
4. 外部 agent 经控制面的 `orchestrator_report` 等工具回报。
5. 任务组停下（全部结束、有人挂起或失败）时，`route_wake` 叫醒路由，路由在主会话里汇总。单张工单不叫醒路由，只把交接写回主会话。

---

## 3. Buddy 管理器

**目录。** 内置的精选表（`curated.json`）、ACP 注册表（`registry.json`，可在后台刷新）和人加的自定义 agent 合成一张目录。`yoyo` 自己是内置项，永远在线。

**检测。** 第一次列出时在后台做：看托管安装（`buddies/<id>/install.json` 且入口还在）、PATH 上的 ACP 命令、只有 CLI 没有适配器三种情况。PATH 是扩展过的：进程 PATH，macOS 和 Linux 再加登录 shell 的 PATH（3 秒超时，缓存 30 分钟），最后补常见安装目录（npm、pnpm、bun 的全局目录，`~/.local/bin`，Windows 上还有 scoop 和 winget 放命令的地方）。检测只看文件和 PATH，不跑命令、不读凭证文件；做完就不再「检测中」。系统版本随后用 `--version` 补上（并发 8，单个 10 秒，文件没变就用缓存）。

**这台电脑上已经装着的应用。** 厂商安装器和桌面应用常把命令放在不进 PATH 的地方。精选表给每个 agent 按系统列出这些位置（`host`），PATH 上没有时再去看（`host.go`）。位置里可以写 `%NAME%`、`$NAME` 和开头的 `~`，有变量没设就整条不找；按版本分的目录写 `*`，取版本最高的。找到的东西分三种：

| 种类 | 是什么 | 算什么 | 界面上 |
| --- | --- | --- | --- |
| `acp` | 自己会说 ACP 的命令，只是不在 PATH 里（例如 `~/.opencode/bin/opencode`） | 和 PATH 上的系统版一样：算已安装，第一次列出时就探测 | 在「在线」或「需要处理」里，详情写明是在哪找到的 |
| `cli` | 厂商的命令行，自己不说 ACP（例如 Claude 桌面应用带的 `claude`） | 不算已安装，标成「只差适配器」 | 在「在这台电脑上发现」里，主按钮是「连接」：装的是 yoyo 托管的适配器 |
| `app` | 只装了桌面应用，没有命令行（例如 Cursor） | 不算已安装，只记一笔 | 在「在这台电脑上发现」里，说清还要装它的命令行；安装照旧先给人看完整命令 |

PATH 上找得到的优先，也不标成「在别处找到」。桌面应用本身只是 ACP 的客户端、不能当工人的（Zed、JetBrains 这类），不列。

**用人已经装好的 CLI。** 适配器支持改用外部 CLI 时（精选表的 `cliEnv`，现在只有 Claude 的 `CLAUDE_CODE_EXECUTABLE`），启动适配器会把本机 CLI 的路径交给它。详情页有一个开关可以关掉，关掉就用适配器自带的；人在环境变量里自己写了同名变量时用人写的；文件不在了就不交。Codex 的适配器自带固定版本的 codex，官方说明别的版本可能不兼容，所以不指过去。

**这些位置的依据。** `~/.local/bin`（Claude Code 原生安装器，Windows 上是 `%USERPROFILE%\.local\bin`）和 `CLAUDE_CODE_EXECUTABLE` 出自官方文档和适配器的 README。Claude 桌面应用里 CLI 的位置（`%APPDATA%\Claude\claude-code\<版本>\`，macOS 在 `~/Library/Application Support/Claude/claude-code/<版本>/`）、`%LOCALAPPDATA%\cursor-agent` 和 `~/.opencode/bin` 来自用户报告和安装脚本，没有在真机上逐个验证；位置写错的后果只是找不到，不会误报。Cursor 桌面应用的两个位置在 Windows 11 上核对过。

**安装。** 混合，默认托管：

- 托管：用 yoyo 私有的 Node 和 uv（装在 `<YOYO_HOME>/runtime`），agent 装进 `<YOYO_HOME>/buddies/<id>`。npm 源在 npmmirror 和 npmjs 之间择优；二进制从官方地址下载，有 sha256 时才允许走镜像。装完去掉 Windows 的 `Zone.Identifier` 和 macOS 的隔离属性。
- 装到系统（行内菜单）：`npm -g`、`uv tool`，或者官方安装命令；后者先把完整命令给人看，人确认后才跑。
- 系统里已有的 CLI 一律算已安装。正在服务工单的 agent 不能卸载。

**配置与密钥。** `config.yaml` 的 `buddies` 段记每项的开关、模型、审批方式（`yoyo` 或 `ask`）、同时几张工单（默认 2，1 到 4）、额外参数和非密钥环境变量、擅长什么，以及偏好（npm 源、默认 agent、自动挑选、单轮时长上限，默认 60 分钟）。密钥存进 vault，名字是 `buddy.<id>.<NAME>`；配置里只记名字，视图只给 `hasKey`。「从 yoyo 复用」是每项单独的开关，打开后才把 yoyo 同类提供商的钥匙注入给它。像密钥的变量不能写进普通环境变量，要放进密钥栏。

**启动环境。** 白名单变量，加 `TERM=dumb`、`NO_COLOR=1`，加这一项自己的密钥；不继承 `YOYO_*` 和其他像密钥的变量。工作目录是工单目录。

**登录。** 按 agent 在握手里报的登录方式（没报时用精选表的兜底命令）：agent 型调 `authenticate`，最多等 5 分钟；终端型打开可见终端让人操作，结束后或每 5 秒看一次，最多 10 分钟；env_var 型到配置里填密钥。每种做完都全探一次。

**探测与健康。** 全探：启动一次，握手、建会话，记下模型、能力和登录方式（20 秒超时）。轻探：只握手。Start 之后 20 秒做第一次健康检查，之后每 10 分钟一次。轻探失败先记成「刚断开」，5 分钟内没恢复才落成离线；正在服务工单的 agent 算在线。

**状态。** `not_installed`、`installing`、`probing`、`needs_auth`、`needs_config`、`online`、`recently_lost`、`offline`、`error`、`unsupported`。变化经 Hub 推给界面（只带编号、状态、进度和在做的任务数），界面要全量视图时再调 `buddy.get`。

**RPC。** `buddy.list`、`buddy.get`、`buddy.refresh`、`buddy.probe`、`buddy.install`、`buddy.cancel`、`buddy.uninstall`、`buddy.configure`、`buddy.login`、`buddy.logout`、`buddy.prefs`、`buddy.add`、`buddy.log.tail`、`buddy.disk.get`、`buddy.cache.clear`。成功的改动发 `state.changed`。手机上 install、uninstall、login、logout、cache.clear 禁用；configure、prefs、add 要确认；返回里没有密钥值。

---

## 4. 外部工人和控制面

**打开。** 执行器要给 agent 派工时调 Opener：agent 不在线先探一次，还不在线就返回「离线」（要登录时注明，执行器等人登录，不擅自换人；没被点名时退回 yoyo）。在线时取启动方式，在控制面登记一个新 token，打开 acpworker。

**会话。** 对方支持时说明放进 `session/new` 的 system prompt，不支持时放进第一条消息；可写目录和只读目录在对方支持时作为额外目录给出。人选了模型时换一次模型；**yoyo 从不改 agent 的模式**，所以不会把它切到降低监督的模式。

**续接。** 外部会话编号记在派工上。下次先 `session/resume`，不行再 `session/load`（各自要对方声明了才试），都不行才新开，并在第一条消息前补一段延续说明。连接断开或要求登录时不再往下试。

**审批。** `session/request_permission` 和 client fs 读写都走 `capability.Broker`，会话名 `ticket-<id>`，目录限制和 yoyo 自带工人相同：工单目录、命名目录和任务组共享目录可写，前置任务的目录只读。允许时先选 `allow_once`；危险命令照样被确定性名单挡下；审批方式为 `ask` 时每次都问人。一轮被取消时挂着的请求答 cancelled。

**日志。** 外部 agent 的更新归一化成一份 `dsh.Session` 日志，任务页、`worker_log` 和转录行原样复用。

**韧性。** 进程崩溃带着 stderr 尾部（打过码）重连；单轮超过时长上限就取消；用量（上下文和花费）按 agent 报的记到工单上。

**控制面。** `mcpserve` 只监听 `127.0.0.1`，第一次登记时才开始监听。每张工单的工人一个 Bearer token，工人关掉时撤销；只暴露这张工单的 `orchestrator_*` 工具（`orchestrator_report`、`orchestrator_reply`、`orchestrator_handback`、`orchestrator_follow`、`orchestrator_unrelated`）；外来的 `Origin` 和 `Host` 拒绝。agent 支持 HTTP MCP 时直连，token 在 `Authorization` 头里；不支持时 agent 启动 yoyo 自己当 stdio 桥（`YOYO_MCP_BRIDGE=1`），token 在桥的环境变量里。两种都不进命令行参数。

---

## 5. 路由和多 agent

**@ 提及。** 输入框的 `@` 菜单多一个 Agent 分区，选中后是芯片，发出去是 `@agent:<id>`。被点名的 agent 离线时不擅自换人，等它回来或人改派；没点名时用默认 agent（缺省 yoyo）；打开「自动挑选」后路由才按名单里的「擅长」自己挑。

**单个 agent。** `task_route` 的 `agent` 字段：`accept` 时用它开单，`continue` 时和工单现在的不同就改派。改派结束当前派工、清掉续接编号和用量、关掉旧工人（它之后再调工具会被告知停下），新工人从交接说明开始，不继承旧会话。

**任务组。** `task_split` 开任务组：

- `split`：2 到 6 份，每份有 title、agent、role、brief、depends_on。brief 要能独立读懂；几份会改同一个目录时划开区域，做不到就用 depends_on 串起来。依赖只能指向前面的项，天然无环。
- `compare`：2 到 4 个不同的 agent 做同一份说明，各自在自己的工单目录里做，不能有依赖。
- 拆分时建共享目录并写一份 `NOTES.md`（成员、分工、要等谁），已有的不覆盖。
- 依赖全部做完时放行，并把前置的结果作为输入带过去；前置没做成时挂起（`deps_failed`），重试做成后再放行。
- 引擎每两秒、以及每次报告后检查一遍任务组。任务组状态变化时叫醒路由：同一个状态只叫一次，每个任务组最多 6 次，每个会话每小时最多 10 次；超过只写一句通知。
- 写给路由的摘要最多 3000 字，先说明「这是任务组的回报，不是人说的话」，最后是行动规则：分工做完就汇总、不再派工（除非人要求过）；有失败就给出重试、换 agent、停掉整组三种选择；对比做完就比较、推荐、问人要不要采用。
- `task_read` 读一张工单的详情（最后一段话、最后的报告、写过的文件、目录、用量）；`task_pick` 在人同意后记下对比里采用哪一份，并停掉其余成员；`task_stop` 带 group 停掉整组。

**并发。** 全局工单槽不变；每个外部 agent 另有上限（默认 2）。超出的成员排队。

---

## 6. 界面

- **Buddy 页**：会话左侧今天栏的第四个页签，和今天、任务、记忆并排；页签上的数字只数要人处理的 agent。底部坞回到七个图标，Buddy 不再是一个表面，打开它时中间的对话不动。
  - 名单：需要处理、在这台电脑上发现、在线、可安装四组，其余收在「再看几个」里；一个外部 agent 都没有时先给一张欢迎卡和三个推荐。
  - 点一项翻到它的详情，一栏宽、从上到下一张张卡片：概况和主按钮、要填的配置（需要配置时在最上面，保存后马上测试）、登录、手上的工单、派任务的设置、来源和卸载。
  - 偏好、添加自定义 agent、日志也在这一栏里翻页。装到系统、卸载、打不开终端时的登录命令是就地的一张确认卡，不弹对话框；装到系统的卡上是完整命令，人点了「运行」才执行。
  - 样子用今天栏那一套（`features/superagent/day-ui.tsx` 和 `day-pane.css`）：胶囊页签、软卡片、圆钮、胶囊按钮。第一次检测做完之前显示占位，对话照常能用。
  - 别处要打开 Buddy（@ 了一个还不能派的 agent、任务上的「去登录」、命令面板）调 `useUI.openBuddy(id)`：回到对话，今天栏翻到 Buddy 页并落在那一项；今天栏收着时把它滑出来。
- **输入框**：`@` 菜单的 Agent 分区和芯片。
- **任务栏目**：任务组是一行，展开是成员车道；分工页按时间合并成员的动态，对比页按列并排；工单行带 `BuddyMark` 和名字。
- **主会话**：任务组卡片（成员状态折叠成一行）、交接身份、唤醒回执，以及路由的汇总和推荐。

---

## 7. 安全

要点见 [invariants.md](invariants.md) 第 21–24 条，威胁和对策见 [threat-model.md](threat-model.md)。最容易误解的一条：**第三方 agent 以用户权限运行**，它自己的沙箱、shell 和文件工具不归 yoyo 管；yoyo 只管住经过 yoyo 的写入和权限请求。

---

## 8. 诊断

- 组件 `buddy`：状态变化和人做的操作。组件 `acp`：连接诊断和工人失败。
- 调试分类 `acp`：`YOYO_DEBUG=acp` 或 `config.yaml` 的 `log.debug` 打开后记协议帧，打码后每帧截到 2 KB；控制面的每次工具调用也记在这个分类里。
- Doctor 的 `buddy` 一节：摘要、控制面、帧日志开关。读 Doctor 不会开始检测。

细节见 [logging.md](logging.md)。

---

## 9. 性能目标

| 目标 | 怎么保证 | 实测 |
| --- | --- | --- |
| `buddy.list` 小于 50 ms（读缓存） | 检测做完后列出只读内存里的记录，不找命令、不跑命令（`TestListReadsCacheOnly`） | 45 项，每次约 32 µs |
| 全量检测小于 3 秒 | 只看文件和 PATH；`--version` 在后台补（`TestDetectionDoesNotWaitForVersions`：版本命令各要 5 秒，检测照样在 3 秒内做完） | Windows 开发机上 45 项约 160 ms（`BenchmarkDetectOnThisMachine`） |
| 打开 Buddy 不阻塞界面 | 列表立即返回，`detecting` 为真时界面先出骨架 | e2e「opening Buddy during the first scan…」 |
| 任务组让 2 秒轮询的 view 增加不超过 10% | 任务组一行只带编号、状态、计数和成员编号 | 一天六张普通工单加一个分工组、一个对比组：+5.0%（`TestGroupsAddLittleToTheView`） |

两处边界：

- macOS 和 Linux 上，第一次检测还要等登录 shell 给出 PATH（超时 3 秒，缓存 30 分钟）。登录 shell 慢到接近 3 秒时，第一次检测会略超 3 秒；之后的检测不再等它。
- view 里只有刚拆出来的成员时，任务组占的比例更高（两组四张约 +14%，约 270 字节一组）；成员一开始报告就降下来。

量本机检测：`go test ./internal/buddy -run '^$' -bench BenchmarkDetectOnThisMachine -benchtime 5x`。

---

## 10. 手动验收脚本

Windows 和 macOS 各跑一遍。准备：能上网；有 Claude 账号或钥匙；最好装着 Codex 和 Gemini CLI（没有就在 Buddy 里装）。开着 `YOYO_DEBUG=acp` 跑，事后看日志。

| 步骤 | 做什么 | 应该看到 |
| --- | --- | --- |
| 1 | 打开 Buddy，安装 Claude（默认托管），登录 | 安装依次走过各阶段，然后要求登录；登录后状态变成在线 |
| 2 | 在主会话输入 `@Claude @Codex 做 todo 应用，后端给 codex` | 路由开出分工任务组，主会话出现任务组卡片；任务栏目里一行任务组，两条车道 |
| 3 | 等后端成员做完 | 前端成员自动开始，输入里带着后端的结果 |
| 4 | 等全部结束 | 路由被叫醒一次，在主会话里汇总：做成了什么、东西在哪、下一步 |
| 5 | 输入 `@Claude @Gemini 分别实现快速排序` | 出现对比卡；全部结束后路由给出推荐并问要不要采用；同意后记下采用的那份，其余停下 |
| 6 | 有工单在跑时，去 Buddy 卸载正在用的 agent | 被拒绝，提示它正在服务工单 |
| 7 | 断网 | 状态依次变成刚断开、离线（5 分钟内）；恢复网络后重新在线 |

跑完检查：

- `yoyo logs tail --component buddy` 有安装、登录和状态变化；`--component acp` 有工人打开和连接诊断。
- 日志、Doctor 的 `buddy` 一节和支持包里搜不到任何钥匙或 token 的值。
- 手机上打开 Buddy：安装、卸载、登录、退出、清缓存是禁用的；改配置要确认。

---

## 11. 不做的事

- Codex app-server、Claude stream-json 这类原生协议。以后可以作为 `Worker` 的另一种实现补上。
- A2A、远程或云端 agent、跨机器调度。
- agent 之间直接对话（用共享目录加路由协调代替）。
- 手机端新界面（只同步方法表和权限）。
- ACP v2（草案）。只实现 v1，版本不符时明确报错。
