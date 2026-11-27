//go:build !ios

// 本文件是非 iOS 上的应用选项。modifyOptionsForIOS 不改 opts。包注释在 main_ios.go。

//lint:file-ignore U1000 Wails iOS 模板的空实现，与 app_options_ios.go 成对；iOS 工具链用 build/ios 覆盖编译，不在这里删。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS 在非 iOS 上什么都不改。
// opts 没有被读取，也没有被写入。
func modifyOptionsForIOS(opts *application.Options) {
	// 非 iOS 不需要改选项。
}
