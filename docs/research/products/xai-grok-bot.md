# xAI Grok Bot

| | |
| --- | --- |
| 来源 | 官方公告与文档，不是论文 |
| 公告 | https://x.ai/news/introducing-grok-bot ；设计 https://x.ai/news/designing-grok-bot ；入口 https://x.ai/bot |
| 文档 | https://docs.x.ai/grok-bot/skills-routines-and-automations |
| 状态 | **watch** |

与 [xai-grok-build.md](xai-grok-build.md) 不是同一个产品。Build 是开源 coding harness。Bot 是带例行的云端助手。

## 主张

技能规定怎么做，例行规定什么时候做。例行可以由日程启动，也可以由 Slack、GitHub、Linear、Sentry、PagerDuty、邮件或 webhook 启动。做法是先跟人做一遍，再自动化。文档要求监听窄：「每条新消息」这种过宽的触发会被拒绝。试跑做的是真实工作。文档片段里的限额包括账号每小时 300、单条例行每小时 200、日程间隔大约不少于 5 分钟、每个 Bot 最多 50 条例行。结果和异常回到对话记录。企业说明里还有审计和网络边界。

介绍页写每个 Bot 有自己的电脑；对照博客里有人写成共享电脑。以设计说明和文档为准，并保留这个不一致，不采用博客里的数字。

## 对 Yoyo 的含义

一项长期职责是一条规则，不是隐藏的 cron。先有一次人接受的做法，再考虑启用 auto。事件源就是来源。Yoyo 已有 off / observe / auto 和定时规则。缺口是：规则还是模板，不是从一次成功的演示里长出来的；晨间信还不是评估之后的去向。

方案见 [../../proactive.md](../../proactive.md)。

## 可偷 / 应拒

- 偷：窄触发；来源缺失就失败，不用旧结果顶上；先试跑再启用。
- 拒：云电脑、许多 Bot 共用登录、把 Slack 当主会话。
