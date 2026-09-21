# Yoyo logging and observability

最后审阅：**2026-09-21**。对照 Claude Code debug 目录、Cursor `/logs`、Copilot CLI events+OTel、Codex App Server 分进程纪律，以及 Wails v3 slog。

落地状态：**Slice 0–4 已进树。** 四平面不得互相写入。诊断日志默认不含 prompt / tool body / vault。OTLP 默认关。

---

## 0. 判断

Yoyo 有会话账本和审计链，曾经没有诊断平面。Settings 曾把 journal 尾部标成 Logs。那是误标，不是体系。

四平面：

| 平面 | 路径 | 谁写 | 谁读 |
| --- | --- | --- | --- |
| A 诊断 | `YOYO_HOME/logs/` | 进程、RPC、MCP、前端桥 | 操作员、支持包、CLI |
| B 轨迹 | `sessions/{id}.jsonl` | `runtime.emit` | loop rebuild、Review Trace |
| C 审计 | `journal/chain.jsonl` | evolve / checkout / ACE | L4 完整性、Settings Journal |
| D spans | `observe/spans.jsonl` | turn / model / tool / checkpoint | `yoyo trace`、可选 OTLP |

Loop 继续只写 B 和 D。A 只记元事件。C 仍是低频 hash-chain。

硬约束：

- worker stdout 是 JSON-RPC，诊断不得写 stdout
- 密钥只在 vault；A 平面强制脱敏
- journal 不是 syslog
- 轨迹不是 syslog
- OTLP 是 L3、opt-in；关掉出口后本地仍完整
- agent 不能 `read_file` `logs/` `journal/` `observe/` `vault.json`

---

## 1. 诊断平面

包 `internal/diaglog`。标准库 `log/slog` JSON。自研 rotator。不引入 zap。

```
logs/yoyo.log              GUI / 单进程
logs/worker.log            YOYO_WORKER=1
logs/cli.log               cmd/yoyo
logs/renderer.jsonl        前端 error/warn/uncaught
logs/mcp/{name}.stderr.log MCP stderr
logs/crash/panic-*.txt     recover stack
```

`YOYO_LOG_DIR` 覆盖目录。轮转默认 32MiB × 14 份 × 14 天，旧文件 gzip。

GUI / CLI 同时把可读文本打到 **stderr**（文件仍是 JSON）。worker 不写 stdout，也不把 slog 打到 stderr，以免污染 JSON-RPC。`YOYO_LOG_STDERR=0` 关掉终端镜像。

固定字段：`ts` `level` `msg` `component` `cat` `process` `pid` `goos` `goarch` `version`，以及关联核 `session_id` `turn_id` `task_id` `request_id` `trace_id` `span_id`。

DEBUG 默认关。`YOYO_DEBUG=rpc,mcp,hooks` 或 `config.log.debug` 开类别，不是全局 verbose。

---

## 2. 关联核

`turn_id` 在 `sendLocked` 生成，写入 context。同一 W3C `trace_id` 贯穿 A 和 D。子代理继承 trace，换自己的 turn_id。

轨迹事件可带 `turn_id`（向后兼容）。

---

## 3. Spans

`internal/observe`：W3C ids、parent、嵌套 `turn` → `model.complete` → `tool.exec`，checkpoint 独立 span。属性用 token 真值 / cache / compaction 前后，不把摘要全文当 attr。

`YOYO_OTEL_ENDPOINT` 空则只写文件。出口前再跑 redactor。

---

## 4. 支持包

`yoyo doctor --export FILE.zip` / Settings Export。默认：脱敏日志、doctor JSON、spans、剥密钥的 config、isolation、版本。不含 vault、CAS、spill、MEMORY.md、connectors token、held-out 题面。操作员勾选才附加某 session 的 Trace 投影（仍无 spill body）。Harbor 密封套件含 `logs-no-read`：agent 读诊断平面必须失败。

---

## 5. 操作员面

Settings → Advanced 拆成 **Diagnostics**（A 的 tail，level 过滤）和 **Journal**（C 的尾，标注 agent 不能改）。Copy path / Reveal folder / Export support bundle / Diagnose this turn（脱敏 inject，`TypeInject`）。Doctor 增加 `logs.dir`、`last_error`、`disk_ok`、`worker_alive`。

CLI：

```
yoyo logs path
yoyo logs tail [--level] [--component] [--session]
yoyo doctor --export FILE.zip
```

崩溃：`logs/crash/panic-*.txt`，下一条错误卡带 `crash_path`。

---

## 6. 配置与类别

`config.yaml` 的 `log:`（L3，evolve 不可改 `internal/diaglog`）：

| 字段 | 默认 | 作用 |
| --- | --- | --- |
| `level` | info | slog 水位 |
| `debug` | `[]` | 类别，与 `YOYO_DEBUG=rpc,mcp,hooks` 合并 |
| `max_size_mb` | 32 | 单文件 |
| `max_files` | 14 | 轮转份数 |
| `max_age_days` | 14 | 保留 |
| `otel_endpoint` | 空 | 被 `YOYO_OTEL_ENDPOINT` 覆盖 |

DEBUG 类别 `payload` 才允许 ≤512 rune 的已脱敏预览。禁止 `--verbose` 把 prompt 打进磁盘。

---

## 7. 路径与能力

`capability.ForbiddenDiagPath` 挡住 `YOYO_HOME/{logs,journal,observe}` 和 `vault.json`。workspace jail 通常已挡住；workspace==home 时仍显式 deny。shell argv 走同一检查。

前端只桥 `window.onerror` / `unhandledrejection` / ErrorBoundary，不捕获 `console.log`。Wails `Options.Logger` 是同一 slog，带 `component=wails`。GUI/CLI 文本镜像到 stderr。worker `cmd.Stderr` 进 `worker.log`，stdout 只给 JSON-RPC。

---

## 8. 明确拒绝

- journal 或 trajectory 当 syslog
- zap / ELK / 默认 OTLP / 默认云同步
- 前端第二套 logger
- 让模型默认 `read_file` 日志目录
- 为 1M 窗口关掉诊断

