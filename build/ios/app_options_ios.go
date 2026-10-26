//go:build ios

// 本文件在 iOS 上关掉默认信号处理。入口包装在 main_ios.go。

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS 把 DisableDefaultSignalHandler 设为 true，避免 iOS 上因默认信号处理崩溃。
// opts 的 DisableDefaultSignalHandler 被写成 true。
func modifyOptionsForIOS(opts *application.Options) {
	// 在 iOS 上关掉默认信号处理，避免崩溃。
	opts.DisableDefaultSignalHandler = true
}
