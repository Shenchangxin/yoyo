// remake 是 Hypit 的 Go 翻译，按其许可证保留，不进 Yoyo 安装包，所以单独成模块：
// 主模块的 ./... 不再编译、检查或测试它，它也不能 import 主模块的 internal 包。
module github.com/Shenchangxin/yoyo/remake

go 1.26.0

require (
	golang.org/x/sys v0.48.0
	golang.org/x/text v0.42.0
)
