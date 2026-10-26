//go:build ios

// Package main 是 Wails iOS 构建的入口包装。WailsIOSMain 由 WailsAppDelegate 在 didFinishLaunchingWithOptions 之后、后台线程上调用用户的 main，从而不和 UIApplicationMain 并发启动。
// 这段包装不改用户自己的 main.go。同包在 iOS 上会改应用选项，非 iOS 的空实现在另一个文件。
package main

import (
	"C"
)

// WailsIOSMain 调用用户的 main（application.New / app.Run）。
// WailsAppDelegate 在 didFinishLaunchingWithOptions 里调用它，也就是 UIKit 已经启动之后，并且在后台线程上。这样 Go 运行时不会和 UIApplicationMain 一起启动；那次竞态会在真机上间歇破坏 FrontBoard 的启动握手，表现为冷启动空白或 scene-create 看门狗 0x8BADF00D。应用初始化不占操作系统主线程，主线程上的 UIApplicationMain 也不会被挡住。
//
//export WailsIOSMain
func WailsIOSMain() {
	main()
}
