# Proactive AI for Superintelligent Agents

| | |
| --- | --- |
| 作者 | Ke Yang 等（UIUC、Google、Stanford、UC Berkeley、CMU） |
| 年份 | **2026**（Google Research 页只标年份，未标月日） |
| 项目 | https://empathyang.github.io/Proactive-AI-for-Superintelligent-Agents/ |
| PDF | https://empathyang.github.io/Proactive-AI-for-Superintelligent-Agents/Proactive-AI-for-Superintelligent-Agents.pdf |
| Google | https://research.google/pubs/proactive-ai-for-superintelligent-agents/ |
| 转述 | https://mp.weixin.qq.com/s/2Ge6Pp2oCAk8fZx7NUE3dg |
| 状态 | **candidate** |

2026-11-03 检索时没有找到 arXiv 号。卡片以项目页和 Google Research 条目为准。

## 主张

主动性是：没有针对这一次行动的明确指令时，自己发起朝向目标的行动。它和智能、对齐并列，也不等于已经超级智能。授权仍由人按可靠性授予。机制收成六种能力：掌握情况、预判、形成议程、判断介入、落实行动、从反馈里调整。预判和判断介入是主动式特有的两段。评估分三层：任务结果、机制是否选对时机、人的负担与控制感。主动帮助的一种失败是事情做对了，但人不希望它发生。只奖励「人接受了建议」，会把系统练成更会劝，而不是更有帮助。

## 对 Yoyo 的含义

六项里，判断介入和议程是当前缺口。人的消息和写成的规则已经走 `policy.Gate`；晨间信和兴趣库变化到点就推。定时任务在 `runIsolatedJob` 里直接开会话。方案见 [../../proactive.md](../../proactive.md)。

## 可偷 / 应拒

- 偷：主动性与授权分开；预判和判断介入先于行动；反馈要能区分忽略、拒绝和事实错误；评估把人的负担算进去。
- 拒：常开截屏、可穿戴和医疗感知；用接受率训练一个更会劝的模型；把综述里的十三个领域都做成产品。
