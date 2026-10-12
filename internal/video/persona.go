package video

import "strings"

// PersonaTemplate is a creator-voice page (not a Profile). Profiles are
// capability projections and the video channel ignores them.
func PersonaTemplate(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "创作者人设"
	}
	return `# ` + name + `

这是短片模块的人设页，不是会话 Profile。发短片前对照选题卡打分。

## 定位
- 谁在看：
- 他们为什么停下来：
- 不做什么（禁区）：

## 六维
| 维 | 一句话 |
|---|---|
| 人设 | |
| 口吻 | |
| 领域 | |
| 节奏 | |
| 视觉 | |
| 禁区 | |

## 样本口播
> 贴一段代表作旁白，选题评估时对照 persona 分。

## 平台备注
- 小红书：
- 抖音：
- B站：
`
}
