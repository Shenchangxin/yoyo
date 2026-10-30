package im

import (
	"context"
	"errors"
)

// runners 的键必须和 Catalog 的 id 一一对应。缺一个，设置页能保存，监督循环却只会报未知平台。
var runners = map[string]func(*Runtime, context.Context, Account) error{
	"telegram":                (*Runtime).runTelegram,
	"onebot":                  (*Runtime).runOneBot,
	"qq_official":             (*Runtime).runQQ,
	"qq_official_webhook":     (*Runtime).runWait,
	"lark":                    (*Runtime).runLark,
	"dingtalk":                (*Runtime).runDingTalk,
	"wecom":                   (*Runtime).runWait,
	"wecom_ai_bot":            (*Runtime).runWeComAI,
	"weixin_official_account": (*Runtime).runWait,
	"weixin_personal":         (*Runtime).runWeixinPersonal,
	"slack":                   (*Runtime).runSlack,
	"discord":                 (*Runtime).runDiscord,
	"line":                    (*Runtime).runWait,
	"satori":                  (*Runtime).runSatori,
	"kook":                    (*Runtime).runKook,
	"misskey":                 (*Runtime).runMisskey,
	"mattermost":              (*Runtime).runMattermost,
	"whatsapp":                (*Runtime).runWait,
	"matrix":                  (*Runtime).runMatrix,
	"rocketchat":              (*Runtime).runRocket,
	"vocechat":                (*Runtime).runWait,
}

func (rt *Runtime) runWait(ctx context.Context, acct Account) error {
	if usesHook(acct) {
		if msg := rt.hookProblem(); msg != "" {
			return errors.New(msg)
		}
	}
	<-ctx.Done()
	return nil
}
