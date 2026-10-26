// persona.go 给出短片模块的创作者人设页模板。

package video

import "strings"

// PersonaTemplate 返回一份创作者人设 Markdown。名字空着时用「创作者人设」。这是短片人设页，不是会话 Profile；Profile 是能力投影，视频通道不读它。
// name 去掉首尾空白，空串时换成「创作者人设」，再写进返回 Markdown 的一级标题。
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
