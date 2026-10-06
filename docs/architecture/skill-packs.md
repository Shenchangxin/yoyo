# Skill Packs

最后审阅：**2026-10-06**。这是 Yoyo 对 Agent Skills 第三方库的长期运行时，不是一次 Superpowers 移植。

Harbor 评测、HeuristicSolver、默认套件 **永不**加载 pack。个人助手默认关闭方法论。

---

## 判断

Claude Code 用 `hooks.json` session-start 注入 `using-superpowers`；Codex 把 Superpowers 当成 Skill Pack，安装后始终在会话里；Cursor 用 `.cursor/skills`。Yoyo 是 Go 进程内循环，没有也不该引入 Shape A 的 `hooks.json` 副作用链。

正确形状是 **Shape B：进程内 bootstrap**：

1. Pack 是分发单元（`~/.yoyo/packs/<id>/`），不是 vendored 技能正文。
2. 启用解析：工作区 `.yoyo/packs.json` > 全局 `config.yaml` `packs.<id>.enabled` > **默认 false**。
3. 会话开始把 bootstrap skill（Superpowers 为 `using-superpowers`）包进 Dynamic loaded-skills，外加生成的 `yoyo-tools.md` 映射。不写进 identity，不改 system prompt。
4. 方法论 pack 让路 chatConduct 的「这轮就开始写」；`plan-first` overlay 关掉。YOYO.md / AGENTS.md / 用户直接跳过仍赢。
5. `task` 子会话 **结构上**跳过 bootstrap（`Depth>0` + `StripPackBootstrap`），不靠模型自觉。
6. 包文件在工作区 jail 外：读 `read_skill_file`，跑 `run_skill_script`。无扩展名脚本走 Git Bash，不用 WSL `bash.exe`。

不要 fork Superpowers 正文。适配器只有映射表和 bootstrap 包装。

---

## 磁盘

| 路径 | 作用 |
|---|---|
| `~/.yoyo/packs/<id>/pack.json` | 清单：origin、bootstrap、methodology、commit |
| `~/.yoyo/packs/<id>/skills/` | 安装器拷贝的 `skills/` 树 |
| `.../using-superpowers/references/yoyo-tools.md` | 安装时生成，不进上游仓库 |
| `.yoyo/packs.json` | 工作区开关覆盖 |
| `config.yaml` `packs.<id>.enabled` | 全局回退 |

`SkillRoots` 顺序：bundled < `~/.yoyo/skills` < **enabled packs** < 工作区 `.yoyo/skills` / `.agents/skills` / `skills`。同名后写覆盖。

---

## 工具映射（整份适配器）

技能只说动作。Yoyo 侧：

| 动作 | 工具 |
|---|---|
| 调用技能 | `load_skill` |
| 读包内文件 | `read_skill_file` |
| 读工作区 | `read_file` / `grep` / `glob` |
| 改文件 | `str_replace` / `apply_patch` / `write_file` |
| 包脚本 | `run_skill_script` |
| 待办 | `update_plan` |
| 子代理 | `task`（`isolate=true` 代替 git worktree） |

禁止发明 `Skill` / `Bash` / `TodoWrite` / `Task`。

---

## API

HTTP `GET /api/packs?workspace=`，`POST /api/packs` `{id, op: install\|uninstall\|enable\|update, path, scope, enabled}`。

JSON-RPC：`packs.list` / `install` / `uninstall` / `enable` / `update`。`enable.scope`：`workspace` \| `global` \| `inherit`。

CLI：`yoyo pack list|install|uninstall|enable|update`。GitHub 拉取可用 `YOYO_GITHUB_TOKEN` 或 `GITHUB_TOKEN`。

桌面：Skills → Packs。安装后默认打开 **当前工作区**。Composer 在方法论开启时显示 Methodology 芯片。

`load_skill` / `read_skill_file` / `run_skill_script` 在工具一发出时就把技能名推到流式过程条和桌宠便捷消息（例如 `brainstorming`），不要等回合结束。

---

## 评测

Harbor `Engine.runTask` 不设 `Packs`，也不走 `App.Send`。BehaviorIDs 与密封套件不变。

可选 live 探针 `evals/superpowers-brainstorm-first/` **不在** BehaviorIDs。需要先在一次性 home 里 `yoyo pack install superpowers` 并启用，再手动把该 id 放进套件 HeldIn。默认 CI 不跑。
