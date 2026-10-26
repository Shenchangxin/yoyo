//go:build android

// Package main 在 Android 的 c-shared 构建里登记入口。init 调用 application.RegisterAndroidMain(main)，因为这种构建模式不会自动调用 main。
package main

import "github.com/wailsapp/wails/v3/pkg/application"

// init 把 main 登记成 Android 应用初始化时要调用的函数。c-shared 构建不会自动进入 main。
func init() {
	// 登记 main，等 Android 应用初始化时再调用。c-shared 构建模式不会自动调用 main。
	application.RegisterAndroidMain(main)
}
