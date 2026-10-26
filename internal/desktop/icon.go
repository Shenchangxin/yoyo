package desktop

import _ "embed"

// AppIcon 是托盘和 Linux 窗口图标用的 Yoyo 标记。
//
//go:embed assets/appicon.png
var AppIcon []byte
