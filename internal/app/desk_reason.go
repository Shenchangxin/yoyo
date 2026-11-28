package app

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/initiative"
)

// signalReason 是一条开口的「为什么是这条」。感知自己写了就用它的，否则按种类说一句。
// 这一句只说开口的理由，不复述正文；界面把它放在短讯下面，和依据、三种反馈放在一起。
func signalReason(s initiative.Signal) string {
	if r := strings.TrimSpace(s.Reason); r != "" {
		return r
	}
	switch s.Code {
	case "calendar":
		if s.State == "overdue" {
			return "日程里这一天已经过了，还没标完成"
		}
		return "日程里写着今天到期"
	case "channel":
		return "这个来源出错时，从这里进来的工作会漏掉"
	case "line":
		return "你在做的这条线和上次看到的不一样了"
	case "git":
		return "这个仓库的状态和上次看到的不一样了"
	case "poi":
		return "你常用的东西里有增减"
	case "power":
		return "电量低而且没在充电，正在跑的工作可能被打断"
	case "net":
		return "断网时来源和模型都用不了"
	case "reach":
		return "和你关注的话题对上了"
	case "routine":
		if s.State == "failed" {
			return "你设的例行没跑成"
		}
		return "你设的例行跑完了"
	}
	return ""
}
