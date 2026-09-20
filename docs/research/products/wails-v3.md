# Wails v3 desktop shell

| | |
| --- | --- |
| 文档 | https://v3.wails.io |
| 状态 | **implemented** — 窗口，不是 runtime |

## 主张

Wails v3 用 Go 做桌面壳，前端是普通 Web 资产。Yoyo 用它提供原生菜单、托盘、通知、拖拽区域，以及 `YOYO_WORKER=1` 时把真正的 agent 进程放到旁边。

## 对 Yoyo 的含义

PRODUCT.md：GUI 是 **一个 Go loop 的客户端**。Wails 绑定只做 `desktop.Service` 对 JSON-RPC/HTTP 的转发（harness.*、evolve.run、approvals），不在前端再写一套 tool loop。这是 Codex App Server 模式在 Windows/macOS 上的落地，不是学术贡献。

隔离徽章必须诚实：OS job object / sandbox-exec / bubblewrap 是 opt-in，UI 不得画假沙箱。

## 可偷 / 应拒

- 已用：壳与 core 分离。
- 拒：在 WebView 里跑模型调用。
