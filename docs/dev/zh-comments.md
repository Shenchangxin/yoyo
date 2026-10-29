# 中文注释清单

这份清单是长程任务的台账。勾选表示该文件已经按下面的标准写过中文注释。未勾选就是还没做。新文件加进来时保持未勾选。

视频前端后来补了参数注释。`IslandBoundary` 已有类注释。

后来在核心函数体里补了决定行为的行注释：循环怎么停、投影钉在哪、路径监狱、密钥、对象库、日志链、晋升、子代理预算、桌面发送、进程拆分、WASM、MCP、更新替换，以及前端怎么选门。行注释只写在这些关键语句旁，不给普通赋值逐行配文。

参数句不写进步循环这几份文件：`internal/runtime/loop.go`、`context_kernel.go`、`context_legal.go`、`context_shape.go`、`defaults.go`、`budget.go`、`tools_task.go`，以及 `sendLocked`、`launchSend`、`StartSend`、`StartSendOpts`、`Send`、`SendOpts`。其余有参数的函数，注释里要出现每个参数名，一句写清这个函数怎么用它。空、nil、小于等于 0 只有函数体真的判断了才写。

## 标准

每个自有文件只保留一套说明，写成中文。已有英文注释里的约束、数字、否定句译进中文后删掉英文。`//go:build`、`//go:embed`、`//nolint`、`//export` 原样留下。

- 每个包只在一个非测试文件里写 `// Package 名字`，紧贴 `package` 行，供 `go doc` 使用。
- 其余文件在 `package` 行之前空一行，写这个文件拥有什么、不拥有什么、谁调用它。空行是为了避免变成第二份包注释。
- 每个函数、方法、导出类型、导出常量，以及有行为含义的字段，至少一句。导出符号的注释以标识符开头。
- 一行转发如果名字已经说完行为，只写半句。函数体内只在约束不明显处写，不给一眼能看懂的语句逐行配文。
- 测试文件写它锁住的不变量；每个 `Test*` 一句。测试函数名保持英文。
- 注释不改行为。数字只写本文件或它引用的常量里出现的那个。说明文和代码不一致时，以代码和测试为准。

写的时候不要把这些旧句子写回去：规则预算 8k、子代理 8 轮或 12 条工具消息、溢出三次才停、每轮改写系统提示前缀。聊天路径把 `RulesTokens` 赋成 1600。桌面子代理是 40 轮和 80 条工具消息，32 只是下限。`WithinWorkspace` 不调用 `EvalSymlinks`。`ShouldPromote` 不含安全失败，`Promote` 先看 `SafetyFail`。Windows Job Object 的探针把 `Sandbox` 写成 false。`GetStored` 不回落环境变量。会话 overlay 结束时断言 active、canary、staging 没被写。`web_fetch` 用 `http.DefaultClient`，重定向不再过 `blockedHost`。

## 排除

这些不手写注释，避免被当成漏做：

- `frontend/src/vendor/**`：第三方画布，约 871 个 ts/tsx。
- `frontend/bindings/**`：Wails 生成，约 47 个文件，手改会被覆盖。
- `frontend/dist`、`node_modules`、`docs/research/corpus`。
- CSS、JSON、Markdown，以及只放用户可见文案的数据表。文案文件若在自有 Go/TS 清单里，只在文件头说明这是数据。

## 自有 Go

共 770 个。


### `build/android`

- [x] `build/android/main_android.go`

### `build/android/scripts/deps`

- [x] `build/android/scripts/deps/install_deps.go`

### `build/ios`

- [x] `build/ios/app_options_default.go`
- [x] `build/ios/app_options_ios.go`
- [x] `build/ios/main_ios.go`

### `build/ios/scripts/deps`

- [x] `build/ios/scripts/deps/install_deps.go`

### `cmd/yoyo`

- [x] `cmd/yoyo/evolve_overlay_test.go`
- [x] `cmd/yoyo/main.go`
- [x] `cmd/yoyo/pack.go`

### `(根目录)`

- [x] `console_windows.go`

### `internal/api`

- [x] `internal/api/acp.go`
- [x] `internal/api/acp_test.go`
- [x] `internal/api/client.go`
- [x] `internal/api/idol_http.go`
- [x] `internal/api/remote_hook.go`
- [x] `internal/api/rpc.go`
- [x] `internal/api/rpc_test.go`
- [x] `internal/api/server.go`
- [x] `internal/api/server_personal.go`
- [x] `internal/api/server_surfaces.go`
- [x] `internal/api/server_video.go`
- [x] `internal/api/ws.go`

### `internal/app`

- [x] `internal/app/app.go`
- [x] `internal/app/app_test.go`
- [x] `internal/app/browser_takeover_test.go`
- [x] `internal/app/canvas_agent.go`
- [x] `internal/app/chat_providers.go`
- [x] `internal/app/chat_providers_test.go`
- [x] `internal/app/config.go`
- [x] `internal/app/config_json.go`
- [x] `internal/app/config_json_test.go`
- [x] `internal/app/harness.go`
- [x] `internal/app/harness_lineage.go`
- [x] `internal/app/hub.go`
- [x] `internal/app/hub_batch_test.go`
- [x] `internal/app/hub_coalesce_test.go`
- [x] `internal/app/hub_seq_test.go`
- [x] `internal/app/idol_app.go`
- [x] `internal/app/idol_media.go`
- [x] `internal/app/model_catalog.go`
- [x] `internal/app/model_catalog_test.go`
- [ ] `internal/app/learn_api.go`
- [ ] `internal/app/learn_tools.go`
- [x] `internal/app/personal.go`
- [x] `internal/app/personal_api.go`
- [x] `internal/app/personal_os.go`
- [x] `internal/app/provider.go`
- [x] `internal/app/provider_test.go`
- [x] `internal/app/pwsh.go`
- [x] `internal/app/queue.go`
- [x] `internal/app/queue_test.go`
- [x] `internal/app/session.go`
- [x] `internal/app/session_actor_test.go`
- [x] `internal/app/session_channel_test.go`
- [x] `internal/app/session_context.go`
- [x] `internal/app/session_dump.go`
- [x] `internal/app/session_dump_test.go`
- [x] `internal/app/session_extra.go`
- [x] `internal/app/session_inventory.go`
- [x] `internal/app/session_inventory_test.go`
- [x] `internal/app/session_ops.go`
- [x] `internal/app/session_overlay.go`
- [x] `internal/app/session_overlay_test.go`
- [x] `internal/app/session_persist_test.go`
- [x] `internal/app/session_retention_test.go`
- [x] `internal/app/session_trace.go`
- [x] `internal/app/session_worktree.go`
- [x] `internal/app/session_worktree_test.go`
- [x] `internal/app/skill_market.go`
- [x] `internal/app/skill_market_test.go`
- [x] `internal/app/skill_pack.go`
- [x] `internal/app/skill_pack_test.go`
- [x] `internal/app/skin.go`
- [x] `internal/app/skin_test.go`
- [x] `internal/app/state.go`
- [x] `internal/app/state_test.go`
- [x] `internal/app/studio_lane_test.go`
- [x] `internal/app/studio_p4.go`
- [x] `internal/app/surfaces.go`
- [x] `internal/app/ui_scale.go`
- [x] `internal/app/ui_scale_test.go`
- [x] `internal/app/update.go`
- [x] `internal/app/video.go`
- [x] `internal/app/video_rpc_test.go`
- [x] `internal/app/wasm.go`
- [x] `internal/app/workspace.go`
- [x] `internal/app/workspace_test.go`

### `internal/artifact`

- [x] `internal/artifact/artifact_test.go`
- [x] `internal/artifact/cas.go`
- [x] `internal/artifact/loop.go`
- [x] `internal/artifact/loop_test.go`
- [x] `internal/artifact/material_diff.go`
- [x] `internal/artifact/material_diff_test.go`
- [x] `internal/artifact/playbook.go`
- [x] `internal/artifact/refs.go`
- [x] `internal/artifact/skill.go`
- [x] `internal/artifact/snapshot.go`
- [x] `internal/artifact/types.go`

### `internal/browser`

- [x] `internal/browser/browser.go`
- [x] `internal/browser/cdp.go`
- [x] `internal/browser/cdp_conn_test.go`
- [x] `internal/browser/cdp_targets.go`
- [x] `internal/browser/cdp_targets_test.go`
- [x] `internal/browser/docpath_test.go`
- [x] `internal/browser/fileserver.go`
- [x] `internal/browser/fileserver_test.go`
- [x] `internal/browser/gestures.go`
- [x] `internal/browser/gestures_test.go`
- [x] `internal/browser/interact.go`
- [x] `internal/browser/interact_e2e_test.go`
- [x] `internal/browser/lanes.go`
- [x] `internal/browser/lanes_test.go`
- [x] `internal/browser/launch.go`
- [x] `internal/browser/launch_test.go`
- [x] `internal/browser/page_errors.go`
- [x] `internal/browser/page_errors_e2e_test.go`
- [x] `internal/browser/page_errors_test.go`
- [x] `internal/browser/page_helpers.go`
- [x] `internal/browser/proc_other.go`
- [x] `internal/browser/proc_windows.go`
- [x] `internal/browser/profiles.go`
- [x] `internal/browser/profiles_test.go`
- [x] `internal/browser/record.go`
- [x] `internal/browser/record_inject.go`
- [x] `internal/browser/record_privacy.go`
- [x] `internal/browser/record_privacy_test.go`
- [x] `internal/browser/record_semantic.go`
- [x] `internal/browser/record_semantic_test.go`
- [x] `internal/browser/record_skill.go`
- [x] `internal/browser/refs.go`
- [x] `internal/browser/refs_test.go`
- [x] `internal/browser/snapshot.go`
- [x] `internal/browser/snapshot_test.go`
- [x] `internal/browser/tabs.go`
- [x] `internal/browser/view.go`
- [x] `internal/browser/view_test.go`

### `internal/capability`

- [x] `internal/capability/broker.go`
- [x] `internal/capability/broker_test.go`
- [x] `internal/capability/gate.go`
- [x] `internal/capability/levels.go`
- [x] `internal/capability/path.go`
- [x] `internal/capability/path_test.go`

### `internal/cite`

- [x] `internal/cite/cite.go`
- [x] `internal/cite/cite_test.go`

### `internal/computeruse`

- [x] `internal/computeruse/computeruse.go`
- [x] `internal/computeruse/computeruse_test.go`
- [x] `internal/computeruse/inject_other.go`
- [x] `internal/computeruse/inject_windows.go`
- [x] `internal/computeruse/virtual_other.go`
- [x] `internal/computeruse/virtual_windows.go`

### `internal/connection`

- [x] `internal/connection/models.go`
- [x] `internal/connection/models_test.go`
- [x] `internal/connection/probe.go`
- [x] `internal/connection/probe_test.go`
- [x] `internal/connection/registry.go`
- [x] `internal/connection/registry_test.go`
- [x] `internal/connection/search.go`
- [x] `internal/connection/search_test.go`
- [x] `internal/connection/seed.go`
- [x] `internal/connection/templates.go`
- [x] `internal/connection/types.go`

### `internal/connector`

- [x] `internal/connector/connector.go`
- [x] `internal/connector/connector_test.go`
- [x] `internal/connector/live.go`
- [x] `internal/connector/oauth.go`
- [x] `internal/connector/view.go`
- [x] `internal/connector/view_test.go`

### `internal/desktop`

- [x] `internal/desktop/autostart_darwin.go`
- [x] `internal/desktop/autostart_other.go`
- [x] `internal/desktop/autostart_windows.go`
- [x] `internal/desktop/chrome.go`
- [x] `internal/desktop/chrome_status.go`
- [x] `internal/desktop/chrome_status_test.go`
- [x] `internal/desktop/companion.go`
- [x] `internal/desktop/companion_hit.go`
- [x] `internal/desktop/companion_hit_test.go`
- [x] `internal/desktop/companion_shape_stub.go`
- [x] `internal/desktop/companion_shape_windows.go`
- [x] `internal/desktop/companion_taskbar_stub.go`
- [x] `internal/desktop/companion_taskbar_windows.go`
- [x] `internal/desktop/cursor_stub.go`
- [x] `internal/desktop/cursor_windows.go`
- [x] `internal/desktop/handoff.go`
- [x] `internal/desktop/hotkey_other.go`
- [x] `internal/desktop/hotkey_windows.go`
- [x] `internal/desktop/hotkeys.go`
- [x] `internal/desktop/icon.go`
- [x] `internal/desktop/idol.go`
- [x] `internal/desktop/idol_test.go`
- [x] `internal/desktop/locale.go`
- [x] `internal/desktop/menus.go`
- [x] `internal/desktop/pulse.go`
- [x] `internal/desktop/pulse_test.go`
- [x] `internal/desktop/reveal.go`
- [x] `internal/desktop/service.go`
- [x] `internal/desktop/service_ops.go`
- [x] `internal/desktop/service_personal.go`
- [x] `internal/desktop/service_remote.go`
- [x] `internal/desktop/service_skin.go`
- [x] `internal/desktop/service_surfaces.go`
- [x] `internal/desktop/service_video.go`
- [x] `internal/desktop/signal.go`
- [x] `internal/desktop/toast_history.go`
- [x] `internal/desktop/toast_history_other.go`
- [x] `internal/desktop/toast_history_test.go`
- [x] `internal/desktop/toast_history_windows.go`
- [x] `internal/desktop/webview_memory.go`
- [x] `internal/desktop/window.go`
- [x] `internal/desktop/worker.go`

### `internal/diaglog`

- [x] `internal/diaglog/bundle.go`
- [x] `internal/diaglog/crash.go`
- [x] `internal/diaglog/diaglog.go`
- [x] `internal/diaglog/diaglog_test.go`
- [x] `internal/diaglog/frontend.go`
- [x] `internal/diaglog/handler.go`
- [x] `internal/diaglog/ids.go`
- [x] `internal/diaglog/redact.go`
- [x] `internal/diaglog/rotator.go`
- [x] `internal/diaglog/tail.go`

### `internal/eval`

- [x] `internal/eval/agent.go`
- [x] `internal/eval/agent_test.go`
- [x] `internal/eval/catalog.go`
- [x] `internal/eval/catalog_test.go`
- [x] `internal/eval/docker.go`
- [x] `internal/eval/expect.go`
- [x] `internal/eval/expect_test.go`
- [x] `internal/eval/harbor.go`
- [x] `internal/eval/harbor_test.go`
- [x] `internal/eval/materialize.go`
- [x] `internal/eval/personal_verify.go`
- [x] `internal/eval/personal_verify_test.go`
- [x] `internal/eval/promote.go`
- [x] `internal/eval/promote_test.go`
- [x] `internal/eval/quiet.go`
- [x] `internal/eval/replay.go`

### `internal/evolve`

- [x] `internal/evolve/admit.go`
- [x] `internal/evolve/admit_test.go`
- [x] `internal/evolve/archive.go`
- [x] `internal/evolve/baseline.go`
- [x] `internal/evolve/curate.go`
- [x] `internal/evolve/curate_test.go`
- [x] `internal/evolve/engine.go`
- [x] `internal/evolve/isolate.go`
- [x] `internal/evolve/library.go`
- [x] `internal/evolve/library_test.go`
- [x] `internal/evolve/materials.go`
- [x] `internal/evolve/mine.go`
- [x] `internal/evolve/mine_test.go`
- [x] `internal/evolve/overlay.go`
- [x] `internal/evolve/overlay_context.go`
- [x] `internal/evolve/overlay_modes.go`
- [x] `internal/evolve/overlay_prompt.go`
- [x] `internal/evolve/overlay_select.go`
- [x] `internal/evolve/overlay_select_test.go`
- [x] `internal/evolve/overlay_test.go`
- [x] `internal/evolve/propose.go`
- [x] `internal/evolve/quality.go`
- [x] `internal/evolve/quality_test.go`
- [x] `internal/evolve/reflect.go`
- [x] `internal/evolve/reflect_llm.go`
- [x] `internal/evolve/reflect_llm_test.go`
- [x] `internal/evolve/select.go`
- [x] `internal/evolve/select_test.go`
- [x] `internal/evolve/trialog.go`
- [x] `internal/evolve/trialog_test.go`

### `internal/expert`

- [x] `internal/expert/expert.go`
- [x] `internal/expert/expert_test.go`

### `internal/filestore`

- [x] `internal/filestore/store.go`
- [x] `internal/filestore/store_test.go`

### `internal/home`

- [x] `internal/home/home.go`

### `internal/hostopen`

- [x] `internal/hostopen/open.go`
- [x] `internal/hostopen/open_test.go`

### `internal/idol`

- [x] `internal/idol/affinity.go`
- [x] `internal/idol/engine.go`
- [x] `internal/idol/export.go`
- [x] `internal/idol/gifts.go`
- [x] `internal/idol/idol_test.go`
- [x] `internal/idol/preset.go`
- [x] `internal/idol/preset_test.go`
- [x] `internal/idol/presets.go`
- [x] `internal/idol/presets_test.go`
- [x] `internal/idol/prompt.go`
- [x] `internal/idol/segments.go`
- [x] `internal/idol/stand.go`
- [x] `internal/idol/stand_test.go`
- [x] `internal/idol/store.go`
- [x] `internal/idol/types.go`
- [x] `internal/idol/world.go`

### `internal/inbox`

- [x] `internal/inbox/inbox.go`

### `internal/isolation`

- [x] `internal/isolation/isolation.go`
- [x] `internal/isolation/isolation_darwin.go`
- [x] `internal/isolation/isolation_linux.go`
- [x] `internal/isolation/isolation_other.go`
- [x] `internal/isolation/isolation_test.go`
- [x] `internal/isolation/isolation_wait_test.go`
- [x] `internal/isolation/isolation_windows.go`
- [x] `internal/isolation/isolation_windows_test.go`

### `internal/journal`

- [x] `internal/journal/journal.go`
- [x] `internal/journal/journal_test.go`

### `internal/kernel`

- [x] `internal/kernel/context.go`
- [x] `internal/kernel/event.go`
- [x] `internal/kernel/fiber.go`
- [x] `internal/kernel/kernel_test.go`

### `internal/learn`

- [ ] `internal/learn/book.go`
- [ ] `internal/learn/db.go`
- [ ] `internal/learn/index.go`
- [ ] `internal/learn/journal.go`
- [ ] `internal/learn/learn_test.go`
- [ ] `internal/learn/library.go`
- [ ] `internal/learn/mastery.go`
- [ ] `internal/learn/migrate.go`
- [ ] `internal/learn/notes.go`
- [ ] `internal/learn/parse.go`
- [ ] `internal/learn/practice.go`
- [ ] `internal/learn/reading.go`

### `internal/memory`

- [x] `internal/memory/memory.go`
- [x] `internal/memory/memory_test.go`

### `internal/modelcatalog`

- [x] `internal/modelcatalog/catalog.go`
- [x] `internal/modelcatalog/catalog_test.go`
- [x] `internal/modelcatalog/family.go`

### `internal/observe`

- [x] `internal/observe/export.go`
- [x] `internal/observe/observe.go`
- [x] `internal/observe/observe_test.go`

### `internal/office`

- [x] `internal/office/edit.go`
- [x] `internal/office/office.go`
- [x] `internal/office/office_test.go`
- [x] `internal/office/pdf.go`
- [x] `internal/office/xlsx.go`

### `internal/osutil`

- [x] `internal/osutil/hide_other.go`
- [x] `internal/osutil/hide_windows.go`
- [x] `internal/osutil/hide_windows_test.go`
- [x] `internal/osutil/osutil.go`

### `internal/pages`

- [x] `internal/pages/pages.go`
- [x] `internal/pages/pages_test.go`
- [x] `internal/pages/review.go`

### `internal/personal`

- [x] `internal/personal/engine.go`
- [x] `internal/personal/execute.go`
- [x] `internal/personal/fetch.go`
- [x] `internal/personal/finance.go`
- [x] `internal/personal/finance_test.go`
- [x] `internal/personal/ideas.go`
- [x] `internal/personal/lifecycle.go`
- [x] `internal/personal/pagediff.go`
- [x] `internal/personal/pagediff_test.go`
- [x] `internal/personal/personal_test.go`
- [x] `internal/personal/proposal.go`
- [x] `internal/personal/store.go`
- [x] `internal/personal/types.go`
- [x] `internal/personal/worker.go`

### `internal/plugin/mcp`

- [x] `internal/plugin/mcp/host.go`
- [x] `internal/plugin/mcp/host_test.go`

### `internal/plugin/wasm`

- [x] `internal/plugin/wasm/host.go`
- [x] `internal/plugin/wasm/host_test.go`
- [x] `internal/plugin/wasm/sign.go`
- [x] `internal/plugin/wasm/sign_test.go`

### `internal/preview`

- [x] `internal/preview/preview.go`

### `internal/profile`

- [x] `internal/profile/profile.go`
- [x] `internal/profile/profile_test.go`

### `internal/project`

- [x] `internal/project/project.go`

### `internal/pwsh`

- [x] `internal/pwsh/extract.go`
- [x] `internal/pwsh/http.go`
- [x] `internal/pwsh/install.go`
- [x] `internal/pwsh/look.go`
- [x] `internal/pwsh/manager.go`
- [x] `internal/pwsh/motw.go`
- [x] `internal/pwsh/pin.go`
- [x] `internal/pwsh/pwsh_test.go`

### `internal/remote`

- [x] `internal/remote/browser.go`
- [x] `internal/remote/browser_test.go`
- [x] `internal/remote/catalog_test.go`
- [x] `internal/remote/cert.go`
- [x] `internal/remote/hosts.go`
- [x] `internal/remote/hosts_test.go`
- [x] `internal/remote/mdns.go`
- [x] `internal/remote/node.go`
- [x] `internal/remote/policy.go`
- [x] `internal/remote/policy_test.go`
- [x] `internal/remote/scrub.go`
- [x] `internal/remote/scrub_test.go`
- [x] `internal/remote/server.go`
- [x] `internal/remote/server_test.go`

### `internal/runtime`

- [x] `internal/runtime/ask_form.go`
- [x] `internal/runtime/ask_form_test.go`
- [x] `internal/runtime/assemble.go`
- [x] `internal/runtime/assemble_skill_test.go`
- [x] `internal/runtime/assembler.go`
- [x] `internal/runtime/assembler_test.go`
- [x] `internal/runtime/attachments.go`
- [x] `internal/runtime/attachments_test.go`
- [x] `internal/runtime/attempt.go`
- [x] `internal/runtime/browser_act.go`
- [x] `internal/runtime/browser_act_test.go`
- [x] `internal/runtime/browser_local.go`
- [x] `internal/runtime/browser_local_test.go`
- [x] `internal/runtime/browser_signature_test.go`
- [x] `internal/runtime/browser_verify.go`
- [x] `internal/runtime/browser_verify_test.go`
- [x] `internal/runtime/budget.go`
- [x] `internal/runtime/budget_test.go`
- [x] `internal/runtime/cache.go`
- [x] `internal/runtime/cache_test.go`
- [x] `internal/runtime/chat_conduct.go`
- [x] `internal/runtime/chat_conduct_test.go`
- [x] `internal/runtime/chat_menu.go`
- [x] `internal/runtime/chat_overlay_test.go`
- [x] `internal/runtime/compact_stream.go`
- [x] `internal/runtime/context_accept_test.go`
- [x] `internal/runtime/context_bill.go`
- [x] `internal/runtime/context_checkpoint.go`
- [x] `internal/runtime/context_coldstart_test.go`
- [x] `internal/runtime/context_contract_test.go`
- [x] `internal/runtime/context_discover.go`
- [x] `internal/runtime/context_gitignore.go`
- [x] `internal/runtime/context_hydrate.go`
- [x] `internal/runtime/context_ingest.go`
- [x] `internal/runtime/context_kernel.go`
- [x] `internal/runtime/context_kernel_test.go`
- [x] `internal/runtime/context_ledger.go`
- [x] `internal/runtime/context_legal.go`
- [x] `internal/runtime/context_lifecycle.go`
- [x] `internal/runtime/context_measure.go`
- [x] `internal/runtime/context_measure_test.go`
- [x] `internal/runtime/context_notes.go`
- [x] `internal/runtime/context_notes_test.go`
- [x] `internal/runtime/context_prompt_test.go`
- [x] `internal/runtime/context_reads_test.go`
- [x] `internal/runtime/context_rules.go`
- [x] `internal/runtime/context_rules_test.go`
- [x] `internal/runtime/context_shape.go`
- [x] `internal/runtime/context_spill.go`
- [x] `internal/runtime/context_test.go`
- [x] `internal/runtime/context_window.go`
- [x] `internal/runtime/context_window_test.go`
- [x] `internal/runtime/defaults.go`
- [x] `internal/runtime/edit_guard.go`
- [x] `internal/runtime/edit_guard_test.go`
- [x] `internal/runtime/errors.go`
- [x] `internal/runtime/errors_stream_test.go`
- [x] `internal/runtime/failover.go`
- [x] `internal/runtime/failover_test.go`
- [x] `internal/runtime/filediff.go`
- [x] `internal/runtime/filediff_test.go`
- [x] `internal/runtime/filesearch.go`
- [x] `internal/runtime/filesearch_test.go`
- [x] `internal/runtime/filesnap.go`
- [x] `internal/runtime/filesnap_test.go`
- [x] `internal/runtime/history.go`
- [x] `internal/runtime/history_test.go`
- [x] `internal/runtime/hooks.go`
- [x] `internal/runtime/hooks_file.go`
- [x] `internal/runtime/horizon_test.go`
- [x] `internal/runtime/html_guard.go`
- [x] `internal/runtime/html_guard_test.go`
- [x] `internal/runtime/hunks.go`
- [x] `internal/runtime/hunks_test.go`
- [x] `internal/runtime/internet_reach_skill_test.go`
- [x] `internal/runtime/ladder.go`
- [x] `internal/runtime/ladder_test.go`
- [x] `internal/runtime/live_event.go`
- [x] `internal/runtime/live_event_test.go`
- [x] `internal/runtime/live_progress.go`
- [x] `internal/runtime/loop.go`
- [x] `internal/runtime/loop_dispatch_test.go`
- [x] `internal/runtime/loop_emit_test.go`
- [x] `internal/runtime/loop_log.go`
- [x] `internal/runtime/loop_observe_test.go`
- [x] `internal/runtime/loop_plan_test.go`
- [x] `internal/runtime/loop_prefetch_test.go`
- [x] `internal/runtime/loop_progress.go`
- [x] `internal/runtime/mentions.go`
- [x] `internal/runtime/mentions_test.go`
- [x] `internal/runtime/middleware_test.go`
- [x] `internal/runtime/model.go`
- [x] `internal/runtime/model_baseurl_test.go`
- [x] `internal/runtime/model_cancel_test.go`
- [x] `internal/runtime/model_wire.go`
- [x] `internal/runtime/model_wire_test.go`
- [x] `internal/runtime/pack_bootstrap.go`
- [x] `internal/runtime/pack_bootstrap_test.go`
- [x] `internal/runtime/page.go`
- [x] `internal/runtime/page_test.go`
- [x] `internal/runtime/planfile.go`
- [x] `internal/runtime/planfile_test.go`
- [x] `internal/runtime/quiet.go`
- [x] `internal/runtime/reach_status.go`
- [x] `internal/runtime/reach_status_test.go`
- [x] `internal/runtime/repeat_guard.go`
- [x] `internal/runtime/repeat_guard_test.go`
- [x] `internal/runtime/sensitive.go`
- [x] `internal/runtime/session_replay_test.go`
- [x] `internal/runtime/session_trace.go`
- [x] `internal/runtime/session_trace_test.go`
- [x] `internal/runtime/shell_find.go`
- [x] `internal/runtime/shell_find_test.go`
- [x] `internal/runtime/shell_inline.go`
- [x] `internal/runtime/shell_inline_test.go`
- [x] `internal/runtime/shell_policy.go`
- [x] `internal/runtime/shell_policy_test.go`
- [x] `internal/runtime/shell_ps.go`
- [x] `internal/runtime/shell_ps_test.go`
- [x] `internal/runtime/shell_quote_test.go`
- [x] `internal/runtime/shell_read.go`
- [x] `internal/runtime/shell_start.go`
- [x] `internal/runtime/shell_start_test.go`
- [x] `internal/runtime/skill_dirless_test.go`
- [x] `internal/runtime/skill_file.go`
- [x] `internal/runtime/skill_label_test.go`
- [x] `internal/runtime/skill_match.go`
- [x] `internal/runtime/skill_match_test.go`
- [x] `internal/runtime/skill_script.go`
- [x] `internal/runtime/skills_disk.go`
- [x] `internal/runtime/solver.go`
- [x] `internal/runtime/stop.go`
- [x] `internal/runtime/stream.go`
- [x] `internal/runtime/tokens.go`
- [x] `internal/runtime/tool_names_test.go`
- [x] `internal/runtime/tools.go`
- [x] `internal/runtime/tools_args.go`
- [x] `internal/runtime/tools_args_test.go`
- [x] `internal/runtime/tools_codex.go`
- [x] `internal/runtime/tools_codex_test.go`
- [x] `internal/runtime/tools_extra.go`
- [x] `internal/runtime/tools_feed.go`
- [x] `internal/runtime/tools_feed_test.go`
- [x] `internal/runtime/tools_git.go`
- [x] `internal/runtime/tools_host.go`
- [x] `internal/runtime/tools_os.go`
- [x] `internal/runtime/tools_pages.go`
- [x] `internal/runtime/tools_patch.go`
- [x] `internal/runtime/tools_personal.go`
- [x] `internal/runtime/tools_range_test.go`
- [x] `internal/runtime/tools_search.go`
- [x] `internal/runtime/tools_skill_test.go`
- [x] `internal/runtime/tools_task.go`
- [x] `internal/runtime/tools_task_test.go`
- [x] `internal/runtime/tools_test.go`
- [x] `internal/runtime/tools_thread.go`
- [x] `internal/runtime/tools_thread_test.go`
- [x] `internal/runtime/truncation.go`
- [x] `internal/runtime/webnet.go`
- [x] `internal/runtime/webnet_test.go`
- [x] `internal/runtime/worktree.go`

### `internal/safeguard`

- [x] `internal/safeguard/safeguard.go`
- [x] `internal/safeguard/safeguard_test.go`

### `internal/schedule`

- [x] `internal/schedule/schedule.go`
- [x] `internal/schedule/schedule_test.go`

### `internal/session`

- [x] `internal/session/index.go`
- [x] `internal/session/manager.go`
- [x] `internal/session/meta.go`
- [x] `internal/session/runstate.go`
- [x] `internal/session/session_test.go`

### `internal/skillmarket`

- [x] `internal/skillmarket/catalog.go`
- [x] `internal/skillmarket/catalog_test.go`
- [x] `internal/skillmarket/clawhub.go`
- [x] `internal/skillmarket/clawskillhub.go`
- [x] `internal/skillmarket/enrich.go`
- [x] `internal/skillmarket/github.go`
- [x] `internal/skillmarket/ha.go`
- [x] `internal/skillmarket/hubs.go`
- [x] `internal/skillmarket/hubs_test.go`
- [x] `internal/skillmarket/install.go`
- [x] `internal/skillmarket/mdr.go`
- [x] `internal/skillmarket/pack.go`
- [x] `internal/skillmarket/scan.go`
- [x] `internal/skillmarket/skillhubcn.go`
- [x] `internal/skillmarket/skillsmp.go`
- [x] `internal/skillmarket/tessl.go`

### `internal/skillpack`

- [x] `internal/skillpack/allow.go`
- [x] `internal/skillpack/enable.go`
- [x] `internal/skillpack/github.go`
- [x] `internal/skillpack/install.go`
- [x] `internal/skillpack/known.go`
- [x] `internal/skillpack/label.go`
- [x] `internal/skillpack/layout.go`
- [x] `internal/skillpack/mapping.go`
- [x] `internal/skillpack/pack_test.go`
- [x] `internal/skillpack/types.go`

### `internal/skin`

- [x] `internal/skin/admit.go`
- [x] `internal/skin/builtin.go`
- [x] `internal/skin/compile.go`
- [x] `internal/skin/contrast.go`
- [x] `internal/skin/pack.go`
- [x] `internal/skin/schema.go`
- [x] `internal/skin/serve.go`
- [x] `internal/skin/skin_test.go`
- [x] `internal/skin/store.go`
- [x] `internal/skin/validate.go`
- [x] `internal/skin/wallpaper.go`

### `internal/tool`

- [x] `internal/tool/registry.go`
- [x] `internal/tool/specs.go`
- [x] `internal/tool/tool.go`
- [x] `internal/tool/tool_test.go`

### `internal/trace`

- [x] `internal/trace/index.go`
- [x] `internal/trace/outline.go`
- [x] `internal/trace/outline_test.go`
- [x] `internal/trace/page.go`
- [x] `internal/trace/store.go`
- [x] `internal/trace/store_page_test.go`
- [x] `internal/trace/store_test.go`

### `internal/update`

- [x] `internal/update/apply.go`
- [x] `internal/update/apply_install.go`
- [x] `internal/update/apply_install_test.go`
- [x] `internal/update/apply_test.go`
- [x] `internal/update/channels.go`
- [x] `internal/update/check.go`
- [x] `internal/update/check_test.go`
- [x] `internal/update/download.go`
- [x] `internal/update/feed.go`
- [x] `internal/update/feed_test.go`
- [x] `internal/update/pubkey.go`
- [x] `internal/update/semver.go`
- [x] `internal/update/semver_test.go`
- [x] `internal/update/sys_other.go`
- [x] `internal/update/sys_windows.go`
- [x] `internal/update/update.go`
- [x] `internal/update/verify.go`
- [x] `internal/update/verify_test.go`

### `internal/vault`

- [x] `internal/vault/keychain_darwin.go`
- [x] `internal/vault/keychain_linux.go`
- [x] `internal/vault/keychain_other.go`
- [x] `internal/vault/keychain_windows.go`
- [x] `internal/vault/vault.go`
- [x] `internal/vault/vault_test.go`

### `internal/version`

- [x] `internal/version/version.go`

### `internal/video`

- [x] `internal/video/assets.go`

### `internal/video/canvas/capability`

- [x] `internal/video/canvas/capability/builtin.go`
- [x] `internal/video/canvas/capability/descriptor.go`
- [x] `internal/video/canvas/capability/registry.go`
- [x] `internal/video/canvas/capability/registry_test.go`

### `internal/video/canvas/contract`

- [x] `internal/video/canvas/contract/generation.go`

### `internal/video/canvas/layout`

- [x] `internal/video/canvas/layout/layout.go`
- [x] `internal/video/canvas/layout/layout_test.go`

### `internal/video`

- [x] `internal/video/canvas_agent.go`
- [x] `internal/video/canvas_agent_test.go`
- [x] `internal/video/canvas_assets_blackmyth.go`
- [x] `internal/video/canvas_assets_cast.go`
- [x] `internal/video/canvas_assets_seed.go`
- [x] `internal/video/canvas_http.go`
- [x] `internal/video/canvas_ops.go`
- [x] `internal/video/canvas_plugins.go`
- [x] `internal/video/canvas_projects.go`
- [x] `internal/video/canvas_protocol.go`
- [x] `internal/video/canvas_resources.go`
- [x] `internal/video/canvas_skill_catalog.go`
- [x] `internal/video/canvas_skill_catalog_extra.go`
- [x] `internal/video/canvas_skill_plugin_test.go`
- [x] `internal/video/canvas_skills.go`
- [x] `internal/video/canvas_store.go`
- [x] `internal/video/canvas_studio_seed_test.go`
- [x] `internal/video/canvas_tasks.go`
- [x] `internal/video/canvas_test.go`
- [x] `internal/video/canvas_timeline.go`
- [x] `internal/video/canvas_tools.go`
- [x] `internal/video/canvas_util.go`
- [x] `internal/video/cast_pack.go`
- [x] `internal/video/cast_pack_test.go`

### `internal/video/cut`

- [x] `internal/video/cut/audio.go`
- [x] `internal/video/cut/audio_test.go`
- [x] `internal/video/cut/captions.go`
- [x] `internal/video/cut/captions_test.go`
- [x] `internal/video/cut/edit.go`
- [x] `internal/video/cut/edit_test.go`
- [x] `internal/video/cut/extract.go`
- [x] `internal/video/cut/grade.go`
- [x] `internal/video/cut/probe.go`
- [x] `internal/video/cut/quiet.go`
- [x] `internal/video/cut/run.go`
- [x] `internal/video/cut/score.go`
- [x] `internal/video/cut/score_test.go`
- [x] `internal/video/cut/types.go`

### `internal/video`

- [x] `internal/video/cut_ops.go`
- [x] `internal/video/cut_project.go`
- [x] `internal/video/cut_test.go`
- [x] `internal/video/cut_tools.go`
- [x] `internal/video/drama.go`
- [x] `internal/video/duration.go`
- [x] `internal/video/engine.go`
- [x] `internal/video/engine_test.go`
- [x] `internal/video/ffmpeg.go`
- [x] `internal/video/ffmpeg_install.go`
- [x] `internal/video/ffmpeg_install_test.go`

### `internal/video/film`

- [x] `internal/video/film/assemble.go`
- [x] `internal/video/film/contract.go`
- [x] `internal/video/film/film_test.go`
- [x] `internal/video/film/font.go`
- [x] `internal/video/film/plan.go`
- [x] `internal/video/film/project.go`
- [x] `internal/video/film/render.go`
- [x] `internal/video/film/words.go`

### `internal/video`

- [x] `internal/video/film_assemble.go`
- [x] `internal/video/film_store.go`
- [x] `internal/video/film_store_test.go`
- [x] `internal/video/gen.go`
- [x] `internal/video/history.go`
- [x] `internal/video/httpx.go`
- [x] `internal/video/id.go`
- [x] `internal/video/import.go`
- [x] `internal/video/job.go`
- [x] `internal/video/matting.go`
- [x] `internal/video/media.go`
- [x] `internal/video/mediaref.go`
- [x] `internal/video/migrate_sqlite.go`
- [x] `internal/video/music_gen.go`
- [x] `internal/video/normalize.go`
- [x] `internal/video/persona.go`
- [x] `internal/video/plan.go`

### `internal/video/protocol`

- [x] `internal/video/protocol/adapter.go`
- [x] `internal/video/protocol/builtin.go`
- [x] `internal/video/protocol/documentation.go`
- [x] `internal/video/protocol/errors.go`
- [x] `internal/video/protocol/expression.go`
- [x] `internal/video/protocol/manifest.go`
- [x] `internal/video/protocol/package.go`
- [x] `internal/video/protocol/payment_rpc.go`
- [x] `internal/video/protocol/registry.go`
- [x] `internal/video/protocol/types.go`

### `internal/video`

- [x] `internal/video/provider.go`
- [x] `internal/video/publish.go`
- [x] `internal/video/quiet.go`
- [x] `internal/video/refs.go`
- [x] `internal/video/refs_test.go`
- [x] `internal/video/remake.go`
- [x] `internal/video/remake_test.go`
- [x] `internal/video/remotion.go`
- [x] `internal/video/remotion_project.go`
- [x] `internal/video/remotion_test.go`
- [x] `internal/video/resolve.go`
- [x] `internal/video/serve.go`
- [x] `internal/video/serve_test.go`
- [x] `internal/video/source.go`
- [x] `internal/video/source_test.go`
- [x] `internal/video/specs.go`
- [x] `internal/video/store.go`
- [x] `internal/video/studio.go`
- [x] `internal/video/studio_auto.go`
- [x] `internal/video/studio_bgm.go`
- [x] `internal/video/studio_catalog.go`
- [x] `internal/video/studio_finish_test.go`
- [x] `internal/video/studio_ops.go`
- [x] `internal/video/studio_pipeline.go`
- [x] `internal/video/studio_produce.go`
- [x] `internal/video/studio_speak.go`
- [x] `internal/video/studio_subs.go`
- [x] `internal/video/studio_test.go`
- [x] `internal/video/studio_tools.go`
- [x] `internal/video/studio_topic_film.go`
- [x] `internal/video/studio_topic_film_test.go`
- [x] `internal/video/style.go`
- [x] `internal/video/style_extra.go`
- [x] `internal/video/templates.go`
- [x] `internal/video/tools.go`
- [x] `internal/video/topic.go`
- [x] `internal/video/trends.go`
- [x] `internal/video/trends_test.go`
- [x] `internal/video/types.go`
- [x] `internal/video/vendor_body_test.go`
- [x] `internal/video/visual.go`
- [x] `internal/video/visual_sign.go`
- [x] `internal/video/visual_test.go`
- [x] `internal/video/wrap.go`
- [x] `internal/video/ytdlp_install.go`
- [x] `internal/video/ytdlp_install_test.go`

### `internal/webcontent`

- [x] `internal/webcontent/antibot.go`
- [x] `internal/webcontent/duckduckgo.go`
- [x] `internal/webcontent/exa.go`
- [x] `internal/webcontent/hits.go`
- [x] `internal/webcontent/hits_test.go`
- [x] `internal/webcontent/http.go`
- [x] `internal/webcontent/jina.go`
- [x] `internal/webcontent/jina_test.go`
- [x] `internal/webcontent/serp.go`
- [x] `internal/webcontent/serp_test.go`
- [x] `internal/webcontent/sites.go`

### `internal/winsize`

- [x] `internal/winsize/size.go`
- [x] `internal/winsize/size_test.go`
- [x] `internal/winsize/workarea_other.go`
- [x] `internal/winsize/workarea_windows.go`

### `(根目录)`

- [x] `main.go`

### `scripts/genicon`

- [x] `scripts/genicon/main.go`

### `scripts/ota`

- [x] `scripts/ota/main.go`
- [x] `scripts/ota/main_test.go`

### `scripts/stampversion`

- [x] `scripts/stampversion/main.go`
- [x] `scripts/stampversion/main_test.go`

### `scripts/yoyo-mark`

- [x] `scripts/yoyo-mark/main.go`

## 自有前端

共 297 个 ts/tsx，不含 vendor。


### `frontend/src`

- [x] `frontend/src/App.tsx`

### `frontend/src/components/ui`

- [x] `frontend/src/components/ui/alert-dialog.tsx`
- [x] `frontend/src/components/ui/badge.tsx`
- [x] `frontend/src/components/ui/button.tsx`
- [x] `frontend/src/components/ui/checkbox.tsx`
- [x] `frontend/src/components/ui/dropdown-menu.tsx`
- [x] `frontend/src/components/ui/empty-state.tsx`
- [x] `frontend/src/components/ui/icon-swap.tsx`
- [x] `frontend/src/components/ui/input.tsx`
- [x] `frontend/src/components/ui/kbd.tsx`
- [x] `frontend/src/components/ui/label.tsx`
- [x] `frontend/src/components/ui/progress-hairline.tsx`
- [x] `frontend/src/components/ui/scroll-fade.tsx`
- [x] `frontend/src/components/ui/select.tsx`
- [x] `frontend/src/components/ui/separator.tsx`
- [x] `frontend/src/components/ui/sheet.tsx`
- [x] `frontend/src/components/ui/slider.tsx`
- [x] `frontend/src/components/ui/switch.tsx`
- [x] `frontend/src/components/ui/tabs.tsx`
- [x] `frontend/src/components/ui/tooltip.tsx`

### `frontend/src/features`

- [x] `frontend/src/features/About.tsx`
- [x] `frontend/src/features/AppErrorBoundary.tsx`
- [x] `frontend/src/features/AskDeck.tsx`
- [x] `frontend/src/features/BootSkeleton.tsx`
- [x] `frontend/src/features/BrowserPane.tsx`
- [x] `frontend/src/features/ChatDock.tsx`
- [x] `frontend/src/features/CommandPalette.tsx`
- [x] `frontend/src/features/Composer.tsx`
- [x] `frontend/src/features/ConfirmDialog.tsx`
- [x] `frontend/src/features/ContextPane.tsx`
- [x] `frontend/src/features/FilePeek.tsx`
- [x] `frontend/src/features/FileTree.tsx`

### `frontend/src/features/harness`

- [x] `frontend/src/features/harness/Changes.tsx`
- [x] `frontend/src/features/harness/ChatNotWorking.tsx`
- [x] `frontend/src/features/harness/HarnessWorkspace.tsx`

### `frontend/src/features`

- [x] `frontend/src/features/HomeStage.tsx`

### `frontend/src/features/idol`

- [x] `frontend/src/features/idol/IdolCall.tsx`
- [x] `frontend/src/features/idol/IdolChat.tsx`
- [x] `frontend/src/features/idol/IdolPanels.tsx`
- [x] `frontend/src/features/idol/IdolPresence.tsx`
- [x] `frontend/src/features/idol/IdolStage.tsx`
- [x] `frontend/src/features/idol/model.test.ts`
- [x] `frontend/src/features/idol/model.ts`
- [x] `frontend/src/features/idol/service-presets.ts`
- [x] `frontend/src/features/idol/stand.test.ts`
- [x] `frontend/src/features/idol/stand.ts`

### `frontend/src/features/idol/stand`

- [x] `frontend/src/features/idol/stand/animations.test.ts`
- [x] `frontend/src/features/idol/stand/animations.ts`
- [x] `frontend/src/features/idol/stand/engine.ts`
- [x] `frontend/src/features/idol/stand/frames.ts`
- [x] `frontend/src/features/idol/stand/spineRuntime.ts`
- [x] `frontend/src/features/idol/stand/StandView.tsx`

### `frontend/src/features/idol`

- [x] `frontend/src/features/idol/useIdolCall.ts`

### `frontend/src/features`

- [x] `frontend/src/features/InboxMenu.tsx`
- [x] `frontend/src/features/Inspector.tsx`

### `frontend/src/features/labs`

- [x] `frontend/src/features/labs/EvolveLab.tsx`
- [x] `frontend/src/features/labs/HarborLab.tsx`
- [x] `frontend/src/features/labs/HarnessLab.tsx`
- [x] `frontend/src/features/labs/LabFrame.tsx`

### `frontend/src/features/learn`

- [ ] `frontend/src/features/learn/LearnWorkspace.tsx`

### `frontend/src/features/pages`

- [x] `frontend/src/features/pages/BrowserShotCard.tsx`
- [x] `frontend/src/features/pages/markdown.test.ts`
- [x] `frontend/src/features/pages/markdown.ts`
- [x] `frontend/src/features/pages/PageEditor.tsx`
- [x] `frontend/src/features/pages/PageReviewCard.tsx`
- [x] `frontend/src/features/pages/PagesWorkspace.tsx`
- [x] `frontend/src/features/pages/review.test.ts`
- [x] `frontend/src/features/pages/review.ts`
- [x] `frontend/src/features/pages/tree.test.ts`
- [x] `frontend/src/features/pages/tree.ts`
- [x] `frontend/src/features/pages/when.test.ts`
- [x] `frontend/src/features/pages/when.ts`

### `frontend/src/features`

- [x] `frontend/src/features/PlanChip.tsx`

### `frontend/src/features/presence`

- [x] `frontend/src/features/presence/busy.ts`
- [x] `frontend/src/features/presence/CompanionApp.tsx`
- [x] `frontend/src/features/presence/companion-call.test.ts`
- [x] `frontend/src/features/presence/companion-call.ts`
- [x] `frontend/src/features/presence/companion-pulse.test.ts`
- [x] `frontend/src/features/presence/companion-pulse.ts`
- [x] `frontend/src/features/presence/director.test.ts`
- [x] `frontend/src/features/presence/director.ts`
- [x] `frontend/src/features/presence/engine.ts`
- [x] `frontend/src/features/presence/flourish.test.ts`
- [x] `frontend/src/features/presence/flourish.ts`
- [x] `frontend/src/features/presence/gaze.test.ts`
- [x] `frontend/src/features/presence/gaze.ts`
- [x] `frontend/src/features/presence/index.tsx`
- [x] `frontend/src/features/presence/PresenceRuntime.tsx`
- [x] `frontend/src/features/presence/stamp.test.ts`
- [x] `frontend/src/features/presence/types.ts`

### `frontend/src/features`

- [x] `frontend/src/features/ProfileChip.tsx`
- [x] `frontend/src/features/ResizeHandle.tsx`
- [x] `frontend/src/features/RunningHub.tsx`

### `frontend/src/features/session-chrome`

- [x] `frontend/src/features/session-chrome/ApprovalTakeover.tsx`
- [x] `frontend/src/features/session-chrome/Attachments.tsx`
- [x] `frontend/src/features/session-chrome/CommandOverlay.tsx`
- [x] `frontend/src/features/session-chrome/ContextMeter.tsx`
- [x] `frontend/src/features/session-chrome/DraftEditor.tsx`
- [x] `frontend/src/features/session-chrome/draft-editor-react19.test.ts`
- [x] `frontend/src/features/session-chrome/icons.tsx`
- [x] `frontend/src/features/session-chrome/mention-chip.ts`
- [x] `frontend/src/features/session-chrome/PermissionSelect.tsx`
- [x] `frontend/src/features/session-chrome/QueueDock.tsx`
- [x] `frontend/src/features/session-chrome/SessionHeader.tsx`
- [x] `frontend/src/features/session-chrome/StatsPills.tsx`
- [x] `frontend/src/features/session-chrome/TodoPanel.tsx`
- [x] `frontend/src/features/session-chrome/WidthControls.tsx`
- [x] `frontend/src/features/session-chrome/WorkspaceChip.tsx`
- [x] `frontend/src/features/session-chrome/WorkspacePicker.tsx`

### `frontend/src/features/settings`

- [x] `frontend/src/features/settings/ExtensionsPanel.tsx`
- [x] `frontend/src/features/settings/GenerationSettings.tsx`
- [x] `frontend/src/features/settings/HandheldPanel.tsx`
- [x] `frontend/src/features/settings/host.ts`
- [x] `frontend/src/features/settings/IdolSettings.tsx`
- [x] `frontend/src/features/settings/pages.tsx`
- [x] `frontend/src/features/settings/PersonalPanel.tsx`
- [x] `frontend/src/features/settings/ProviderMark.tsx`
- [x] `frontend/src/features/settings/ProviderPanel.tsx`
- [x] `frontend/src/features/settings/registry.ts`
- [x] `frontend/src/features/settings/SettingChrome.tsx`
- [x] `frontend/src/features/settings/SettingsPage.tsx`
- [x] `frontend/src/features/settings/SettingsSidebar.tsx`
- [x] `frontend/src/features/settings/SkinStudio.tsx`
- [x] `frontend/src/features/settings/useSettingsHash.ts`

### `frontend/src/features/shell`

- [x] `frontend/src/features/shell/AppFrame.tsx`
- [x] `frontend/src/features/shell/PageHeader.tsx`
- [x] `frontend/src/features/shell/WindowControls.tsx`
- [x] `frontend/src/features/shell/YoyoMark.tsx`

### `frontend/src/features/skills`

- [x] `frontend/src/features/skills/SkillsWorkspace.tsx`

### `frontend/src/features`

- [x] `frontend/src/features/ThreadRail.tsx`
- [x] `frontend/src/features/Titlebar.tsx`
- [x] `frontend/src/features/TracePanel.tsx`
- [x] `frontend/src/features/Transcript.tsx`

### `frontend/src/features/transcript`

- [x] `frontend/src/features/transcript/FilePreview.tsx`
- [x] `frontend/src/features/transcript/ProcessGroup.tsx`
- [x] `frontend/src/features/transcript/SandboxedFrame.tsx`
- [x] `frontend/src/features/transcript/SubagentCard.tsx`
- [x] `frontend/src/features/transcript/TurnStatusBar.tsx`

### `frontend/src/features`

- [x] `frontend/src/features/TurnOutline.tsx`
- [x] `frontend/src/features/UpdateButton.tsx`

### `frontend/src/features/video`

- [x] `frontend/src/features/video/bind.ts`

### `frontend/src/features/video/canvas-host/design-system`

- [x] `frontend/src/features/video/canvas-host/design-system/canvas-theme-host.ts`
- [x] `frontend/src/features/video/canvas-host/design-system/index.ts`
- [x] `frontend/src/features/video/canvas-host/design-system/popup-container.ts`
- [x] `frontend/src/features/video/canvas-host/design-system/surfaces.tsx`
- [x] `frontend/src/features/video/canvas-host/design-system/toolbar.tsx`

### `frontend/src/features/video/canvas-host`

- [x] `frontend/src/features/video/canvas-host/DramaAgentPage.tsx`
- [x] `frontend/src/features/video/canvas-host/host-nav.ts`
- [x] `frontend/src/features/video/canvas-host/hub.ts`
- [x] `frontend/src/features/video/canvas-host/media.ts`
- [x] `frontend/src/features/video/canvas-host/session.ts`
- [x] `frontend/src/features/video/canvas-host/YingceApp.tsx`
- [x] `frontend/src/features/video/canvas-host/yoyo-ant-theme.ts`

### `frontend/src/features/video`

- [x] `frontend/src/features/video/CanvasProjectChip.tsx`
- [x] `frontend/src/features/video/CanvasStudio.tsx`
- [x] `frontend/src/features/video/CreateHome.tsx`
- [x] `frontend/src/features/video/create-mode.ts`
- [x] `frontend/src/features/video/CreateModeTabs.tsx`
- [x] `frontend/src/features/video/cut-desk.ts`
- [x] `frontend/src/features/video/CutStage.tsx`
- [x] `frontend/src/features/video/cut-threads.ts`
- [x] `frontend/src/features/video/DeliveryShelf.tsx`
- [x] `frontend/src/features/video/drama-lib.test.ts`
- [x] `frontend/src/features/video/drama-lib.ts`
- [x] `frontend/src/features/video/DramaMentionField.tsx`
- [x] `frontend/src/features/video/DramaProjectChip.tsx`
- [x] `frontend/src/features/video/drama-stages.tsx`
- [x] `frontend/src/features/video/DramaStudio.tsx`
- [x] `frontend/src/features/video/drama-style-catalog.ts`
- [x] `frontend/src/features/video/drama-style-cover.ts`
- [x] `frontend/src/features/video/drama-style-names-zh.ts`
- [x] `frontend/src/features/video/DramaStylePicker.tsx`
- [x] `frontend/src/features/video/FeaturedCases.tsx`
- [x] `frontend/src/features/video/FilmBoard.tsx`
- [x] `frontend/src/features/video/film-desk.ts`
- [x] `frontend/src/features/video/ingest-clip.ts`
- [x] `frontend/src/features/video/open-cut.ts`
- [x] `frontend/src/features/video/playback-src.test.ts`
- [x] `frontend/src/features/video/playback-src.ts`
- [x] `frontend/src/features/video/remake-desk.ts`
- [x] `frontend/src/features/video/RemakeDraft.tsx`
- [x] `frontend/src/features/video/studio-chrome.tsx`
- [x] `frontend/src/features/video/StudioCopilotHeader.tsx`
- [x] `frontend/src/features/video/StudioDeskChip.tsx`
- [x] `frontend/src/features/video/studio-model.test.ts`
- [x] `frontend/src/features/video/studio-model.ts`
- [x] `frontend/src/features/video/studio-next.test.ts`
- [x] `frontend/src/features/video/studio-next.ts`
- [x] `frontend/src/features/video/StudioProduce.tsx`
- [x] `frontend/src/features/video/studio-session.test.ts`
- [x] `frontend/src/features/video/studio-session.ts`
- [x] `frontend/src/features/video/studio-topic.test.ts`
- [x] `frontend/src/features/video/studio-topic.ts`
- [x] `frontend/src/features/video/studio-topic-card.tsx`
- [x] `frontend/src/features/video/StudioWorkbench.tsx`
- [x] `frontend/src/features/video/VideoHistory.tsx`
- [x] `frontend/src/features/video/video-lanes.test.ts`
- [x] `frontend/src/features/video/video-lanes.ts`
- [x] `frontend/src/features/video/VideoModeSwitch.tsx`
- [x] `frontend/src/features/video/VideoShellNav.tsx`
- [x] `frontend/src/features/video/VideoWorkshop.tsx`
- [x] `frontend/src/features/video/workshop-store.ts`

### `frontend/src/features/workstation`

- [x] `frontend/src/features/workstation/useWorkstation.ts`

### `frontend/src/lib`

- [x] `frontend/src/lib/artifact-preview.test.ts`
- [x] `frontend/src/lib/artifact-preview.ts`
- [x] `frontend/src/lib/ask-form.test.ts`
- [x] `frontend/src/lib/ask-form.ts`
- [x] `frontend/src/lib/assistant-stream.ts`
- [x] `frontend/src/lib/catalog-shape.ts`
- [x] `frontend/src/lib/chrome.ts`
- [x] `frontend/src/lib/client.ts`
- [x] `frontend/src/lib/clipboard.ts`
- [x] `frontend/src/lib/code-plugin.test.ts`
- [x] `frontend/src/lib/code-plugin.ts`
- [x] `frontend/src/lib/context-usage.test.ts`
- [x] `frontend/src/lib/context-usage.ts`
- [x] `frontend/src/lib/control-schema.ts`
- [x] `frontend/src/lib/conversation-fold.ts`
- [x] `frontend/src/lib/copy.ts`
- [x] `frontend/src/lib/copy-zh.ts`
- [x] `frontend/src/lib/diag-bridge.ts`
- [x] `frontend/src/lib/display-title.ts`
- [x] `frontend/src/lib/error.test.ts`
- [x] `frontend/src/lib/error.ts`
- [x] `frontend/src/lib/harness-copy.ts`
- [x] `frontend/src/lib/harness-refs.ts`
- [x] `frontend/src/lib/html-preview.test.ts`
- [x] `frontend/src/lib/html-preview.ts`
- [x] `frontend/src/lib/i18n.ts`
- [x] `frontend/src/lib/incremental-md.test.ts`
- [x] `frontend/src/lib/incremental-md.ts`
- [x] `frontend/src/lib/input-guard.test.ts`
- [x] `frontend/src/lib/input-guard.ts`
- [x] `frontend/src/lib/keymap.ts`
- [x] `frontend/src/lib/layout.ts`
- [x] `frontend/src/lib/local-map.ts`
- [x] `frontend/src/lib/markdown.tsx`
- [x] `frontend/src/lib/mdast-render.tsx`
- [x] `frontend/src/lib/media.ts`
- [x] `frontend/src/lib/mentions.test.ts`
- [x] `frontend/src/lib/mentions.ts`
- [x] `frontend/src/lib/model-catalog.ts`
- [x] `frontend/src/lib/models-dev.snapshot.ts`
- [x] `frontend/src/lib/models-dev.test.ts`
- [x] `frontend/src/lib/models-dev.ts`
- [x] `frontend/src/lib/motion.ts`
- [x] `frontend/src/lib/normalize.ts`
- [x] `frontend/src/lib/notices.test.ts`
- [x] `frontend/src/lib/notices.ts`
- [x] `frontend/src/lib/parse-gfm.ts`
- [x] `frontend/src/lib/pending-auth.ts`
- [x] `frontend/src/lib/plan.ts`
- [x] `frontend/src/lib/popout.ts`
- [x] `frontend/src/lib/presentation-policy.test.ts`
- [x] `frontend/src/lib/presentation-policy.ts`
- [x] `frontend/src/lib/profile-label.ts`
- [x] `frontend/src/lib/protocol.ts`
- [x] `frontend/src/lib/providers.ts`
- [x] `frontend/src/lib/scale.test.ts`
- [x] `frontend/src/lib/scale.ts`
- [x] `frontend/src/lib/scroll-follow.test.ts`
- [x] `frontend/src/lib/scroll-follow.ts`
- [x] `frontend/src/lib/session-stats.test.ts`
- [x] `frontend/src/lib/session-stats.ts`

### `frontend/src/lib/skin`

- [x] `frontend/src/lib/skin/apply.ts`
- [x] `frontend/src/lib/skin/builtin.ts`
- [x] `frontend/src/lib/skin/contrast.ts`
- [x] `frontend/src/lib/skin/extract.ts`
- [x] `frontend/src/lib/skin/index.ts`
- [x] `frontend/src/lib/skin/schema.test.ts`
- [x] `frontend/src/lib/skin/schema.ts`
- [x] `frontend/src/lib/skin/validate.ts`

### `frontend/src/lib`

- [x] `frontend/src/lib/slash.test.ts`
- [x] `frontend/src/lib/slash.ts`
- [x] `frontend/src/lib/split-diff.tsx`
- [x] `frontend/src/lib/store.ts`
- [x] `frontend/src/lib/stream.ts`
- [x] `frontend/src/lib/stream-chunk.test.ts`
- [x] `frontend/src/lib/stream-chunk.ts`
- [x] `frontend/src/lib/stream-cursor.test.ts`
- [x] `frontend/src/lib/stream-cursor.ts`
- [x] `frontend/src/lib/streamdown-highlight-body.tsx`
- [x] `frontend/src/lib/stream-fold.ts`
- [x] `frontend/src/lib/stream-live.ts`
- [x] `frontend/src/lib/surface.ts`
- [x] `frontend/src/lib/theme.tsx`
- [x] `frontend/src/lib/thread.ts`
- [x] `frontend/src/lib/tool-summary.ts`
- [x] `frontend/src/lib/transcript-layout.test.ts`
- [x] `frontend/src/lib/transcript-layout.ts`
- [x] `frontend/src/lib/turn-outline.ts`
- [x] `frontend/src/lib/turn-status.test.ts`
- [x] `frontend/src/lib/turn-status.ts`
- [x] `frontend/src/lib/utils.ts`
- [x] `frontend/src/lib/video-route.test.ts`
- [x] `frontend/src/lib/workspace.ts`

### `frontend/src`

- [x] `frontend/src/main.tsx`
- [x] `frontend/src/vite-env.d.ts`

## 后来补上的入口

这些文件不在最初的 `frontend/src` 扫描里。端到端用例的标题写在 `test()` 上，不再重复一句。

- [x] `frontend/vite.config.ts`
- [x] `frontend/e2e/smoke.spec.ts`
- [x] `frontend/e2e/canvas-mode.spec.ts`
- [x] `frontend/e2e/canvas-spatial.spec.ts`
- [x] `frontend/e2e/error.spec.ts`
- [x] `frontend/e2e/harness.spec.ts`
- [x] `frontend/e2e/harness-refs.spec.ts`
- [x] `frontend/e2e/merge.spec.ts`
- [x] `frontend/e2e/mock.ts`
- [x] `site-src/src/main.tsx`
- [x] `site-src/src/App.tsx`
- [x] `site-src/src/copy.ts`
- [x] `site-src/src/release.ts`
- [x] `site-src/src/workstation.tsx`
- [x] `site-src/src/space-still.tsx`
- [x] `site-src/src/stage-canvas.tsx`
- [x] `site-src/src/vite-env.d.ts`
