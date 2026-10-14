package desktop

import (
	"runtime"
	"strings"
)

func nativeCopy(locale string) nativeStrings {
	l := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"))
	if l == "zh" || strings.HasPrefix(l, "zh-") {
		return nativeZH
	}
	return nativeEN
}

type nativeStrings struct {
	File, NewSession, OpenWorkspace, CloseWindow, Quit                                       string
	View, ToggleReview, ToggleSidebar, ToggleDock, Palette, Settings, OpenHarness, OpenVideo string
	Thread, Compact, Export, Rename, Fork, Archive, Stop                                     string
	Harness, RunEval, RunCycle                                                               string
	Help, About, Doctor, Logs                                                                string
	Show, Hide, ShowCompanion, HideCompanion, NewChat, Pause, Resume                         string
	Companion, FocusMode, ResetPosition, OpenChat, VoiceCall, HangUp                         string
	ReviewApproval, StartAtLogin, StopTask, SettingsEllipsis                                 string
	StatusIdle, StatusPaused, StatusApproval, WakeHint                                       string
	ChooseWorkspace, AttachFiles                                                             string
	TurnFinished, Approval, Ask, ErrTitle, Working, UpdateReady                              string
}

var nativeEN = nativeStrings{
	File: "File", NewSession: "New chat", OpenWorkspace: "Open workspace…", CloseWindow: "Close window", Quit: "Quit",
	View: "View", ToggleReview: "Toggle review", ToggleSidebar: "Toggle sidebar", ToggleDock: "Toggle chat", Palette: "Command palette", Settings: "Settings", OpenHarness: "Open Harness", OpenVideo: "Open Video",
	Thread: "Thread", Compact: "Compact context", Export: "Export markdown", Rename: "Rename", Fork: "Fork", Archive: "Archive", Stop: "Stop",
	Harness: "Harness", RunEval: "Run eval suite", RunCycle: "Run evolve cycle",
	Help: "Help", About: "About Yoyo", Doctor: "Doctor", Logs: "Logs",
	Show: "Show Yoyo", Hide: "Hide window", ShowCompanion: "Show pet", HideCompanion: "Hide pet", NewChat: "New chat", Pause: "Pause", Resume: "Resume",
	Companion: "Desktop pet", FocusMode: "Focus mode", ResetPosition: "Reset position", OpenChat: "Open chat", VoiceCall: "Voice call", HangUp: "Hang up",
	ReviewApproval: "Review approval…", StartAtLogin: "Start at login", StopTask: "Stop current task", SettingsEllipsis: "Settings…",
	StatusIdle: "Idle", StatusPaused: "Paused", StatusApproval: "Needs approval", WakeHint: "Ctrl+Shift+Y to show",
	ChooseWorkspace: "Choose workspace", AttachFiles: "Attach files",
	TurnFinished: "Turn finished", Approval: "Yoyo approval", Ask: "Yoyo question", ErrTitle: "Yoyo error", Working: "Working", UpdateReady: "Update ready",
}

var nativeZH = nativeStrings{
	File: "文件", NewSession: "新对话", OpenWorkspace: "打开工作区…", CloseWindow: "关闭窗口", Quit: "退出",
	View: "查看", ToggleReview: "切换审阅", ToggleSidebar: "切换侧栏", ToggleDock: "切换对话栏", Palette: "命令面板", Settings: "设置", OpenHarness: "打开 Harness", OpenVideo: "打开视频",
	Thread: "会话", Compact: "压缩上下文", Export: "导出 Markdown", Rename: "重命名", Fork: "复制会话", Archive: "归档", Stop: "停止",
	Harness: "Harness", RunEval: "运行评测套件", RunCycle: "跑一轮进化",
	Help: "帮助", About: "关于 Yoyo", Doctor: "Doctor", Logs: "日志",
	Show: "显示 Yoyo", Hide: "隐藏窗口", ShowCompanion: "显示宠物", HideCompanion: "隐藏宠物", NewChat: "新对话", Pause: "暂停", Resume: "继续",
	Companion: "桌面宠物", FocusMode: "专注模式", ResetPosition: "重置位置", OpenChat: "打开对话", VoiceCall: "语音通话", HangUp: "挂断",
	ReviewApproval: "处理审批…", StartAtLogin: "开机启动", StopTask: "停止当前任务", SettingsEllipsis: "设置…",
	StatusIdle: "空闲", StatusPaused: "已暂停", StatusApproval: "待审批", WakeHint: "Ctrl+Shift+Y 显示窗口",
	ChooseWorkspace: "选择工作区", AttachFiles: "附加文件",
	TurnFinished: "轮次结束", Approval: "Yoyo 审批", Ask: "Yoyo 提问", ErrTitle: "Yoyo 错误", Working: "处理中", UpdateReady: "更新已就绪",
}

func (n nativeStrings) chromeWord(st chromeStatus) string {
	switch st {
	case chromePaused:
		return n.StatusPaused
	case chromeApproval:
		return n.StatusApproval
	case chromeWorking:
		return n.Working
	default:
		return n.StatusIdle
	}
}

func (n nativeStrings) chromeLine(st chromeStatus) string {
	return "Yoyo · " + n.chromeWord(st)
}

func (n nativeStrings) trayTooltip(st chromeStatus, wake bool) string {
	line := n.chromeLine(st)
	if !wake || n.WakeHint == "" {
		return line
	}
	return line + "\n" + n.WakeHint
}

func trayWakeHint() bool {
	return runtime.GOOS == "windows"
}
