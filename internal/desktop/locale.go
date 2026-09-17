package desktop

import "strings"

func nativeCopy(locale string) nativeStrings {
	l := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"))
	if l == "zh" || strings.HasPrefix(l, "zh-") {
		return nativeZH
	}
	return nativeEN
}

type nativeStrings struct {
	File, NewSession, OpenWorkspace, CloseWindow, Quit string
	View, ToggleReview, ToggleSidebar, ToggleDock, Palette, Settings, OpenHarness string
	Thread, Compact, Export, Rename, Fork, Archive     string
	Harness, RunEval, RunCycle                         string
	Help, About, Doctor, Logs                          string
	Show, Hide, NewChat                                string
	ChooseWorkspace, AttachFiles                       string
	TurnFinished, Approval, ErrTitle                   string
}

var nativeEN = nativeStrings{
	File: "File", NewSession: "New chat", OpenWorkspace: "Open workspace…", CloseWindow: "Close window", Quit: "Quit",
	View: "View", ToggleReview: "Toggle review", ToggleSidebar: "Toggle sidebar", ToggleDock: "Toggle chat", Palette: "Command palette", Settings: "Settings", OpenHarness: "Open Harness",
	Thread: "Thread", Compact: "Compact context", Export: "Export markdown", Rename: "Rename", Fork: "Fork", Archive: "Archive",
	Harness: "Harness", RunEval: "Run eval suite", RunCycle: "Run evolve cycle",
	Help: "Help", About: "About Yoyo", Doctor: "Doctor", Logs: "Logs",
	Show: "Show Yoyo", Hide: "Hide", NewChat: "New chat",
	ChooseWorkspace: "Choose workspace", AttachFiles: "Attach files",
	TurnFinished: "Turn finished", Approval: "Yoyo approval", ErrTitle: "Yoyo error",
}

var nativeZH = nativeStrings{
	File: "文件", NewSession: "新对话", OpenWorkspace: "打开工作区…", CloseWindow: "关闭窗口", Quit: "退出",
	View: "查看", ToggleReview: "切换审阅", ToggleSidebar: "切换侧栏", ToggleDock: "切换对话栏", Palette: "命令面板", Settings: "设置", OpenHarness: "打开 Harness",
	Thread: "会话", Compact: "压缩上下文", Export: "导出 Markdown", Rename: "重命名", Fork: "复制会话", Archive: "归档",
	Harness: "Harness", RunEval: "运行评测套件", RunCycle: "跑一轮进化",
	Help: "帮助", About: "关于 Yoyo", Doctor: "Doctor", Logs: "日志",
	Show: "显示 Yoyo", Hide: "隐藏", NewChat: "新对话",
	ChooseWorkspace: "选择工作区", AttachFiles: "附加文件",
	TurnFinished: "轮次结束", Approval: "Yoyo 审批", ErrTitle: "Yoyo 错误",
}
