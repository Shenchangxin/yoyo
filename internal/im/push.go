package im

import (
	"context"
	"fmt"
)

func (rt *Runtime) push(ctx context.Context, acct Account, chatID, text string) error {
	text = clipText(text)
	switch acct.Platform {
	case "telegram":
		return rt.pushTelegram(ctx, acct, chatID, text)
	case "onebot":
		return rt.pushOneBot(ctx, acct, chatID, text)
	case "qq_official", "qq_official_webhook":
		return rt.pushQQ(ctx, acct, chatID, text)
	case "lark":
		return rt.pushLark(ctx, acct, chatID, text)
	case "dingtalk":
		return rt.pushDingTalk(ctx, acct, chatID, text)
	case "wecom":
		return rt.pushWeCom(ctx, acct, chatID, text)
	case "wecom_ai_bot":
		return rt.pushWeComAI(ctx, acct, chatID, text)
	case "weixin_official_account":
		return rt.pushWeixin(ctx, acct, chatID, text)
	case "weixin_personal":
		return rt.pushWeixinPersonal(ctx, acct, chatID, text)
	case "slack":
		return rt.pushSlack(ctx, acct, chatID, text)
	case "discord":
		return rt.pushDiscord(ctx, acct, chatID, text)
	case "line":
		return rt.pushLine(ctx, acct, chatID, text)
	case "satori":
		return rt.pushSatori(ctx, acct, chatID, text)
	case "kook":
		return rt.pushKook(ctx, acct, chatID, text)
	case "misskey":
		return rt.pushMisskey(ctx, acct, chatID, text)
	case "mattermost":
		return rt.pushMattermost(ctx, acct, chatID, text)
	case "whatsapp":
		return rt.pushWhatsApp(ctx, acct, chatID, text)
	case "matrix":
		return rt.pushMatrix(ctx, acct, chatID, text)
	case "rocketchat":
		return rt.pushRocket(ctx, acct, chatID, text)
	case "vocechat":
		return rt.pushVoce(ctx, acct, chatID, text)
	default:
		return fmt.Errorf("unsupported platform %s", acct.Platform)
	}
}
