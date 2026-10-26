// Package version 保存用户可见的产品版本。窗口标题、命令行、更新状态和诊断日志都读 Version。
// 源码默认值是 0.6.0。发布构建用 -ldflags -X 覆盖这个字符串变量。
package version

// Version 是用户可见的产品版本，源码默认 0.6.0。
// 发布构建这样覆盖：
//
//	-ldflags "-X github.com/Shenchangxin/yoyo/internal/version.Version=x.y.z"
var Version = "0.6.0"
