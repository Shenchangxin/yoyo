// 本文件拥有市场里列出哪几个技能站，以及市场请求落到哪个站。
// 不拥有站的列表和下载，那些在各站自己的文件。下线的站仍留在 knownHubs 里：已经从它们装好的技能要能按 .market.json 修复和更新。

package skillmarket

// DefaultHub 是市场默认打开的站。
const DefaultHub = HubSkillHubCN

// listedHubIDs 是市场列出的站，顺序就是界面里的顺序。
// 2026-11 逐站实测（每站 3 次拉目录、装 1 个样本）后只留这两个：
// SkillHub 中国目录 1.6s、2000 条、服务端中文搜索、带安全扫描报告；
// Anthropic 官方目录 0.3s、安装 0.7s、质量最高，GitHub 不通时走 jsDelivr。
// Work Buddy 和 ClawHub 拉一次目录要 30 多秒，SkillsMP 会被限流（429），其余是小仓库或二次索引。
var listedHubIDs = []string{HubSkillHubCN, HubAnthropic}

// HubCatalog 返回市场列出的站，顺序就是界面里的顺序。其余站只用来认出已装技能的来源。
func HubCatalog() []HubInfo {
	all := knownHubs()
	out := make([]HubInfo, 0, len(listedHubIDs))
	for _, id := range listedHubIDs {
		for _, h := range all {
			if h.ID == id {
				out = append(out, h)
				break
			}
		}
	}
	return out
}

// MarketHub 把市场请求的站收成列出的站。空串、下线的站和认不出的 id 都回到 DefaultHub。别名按 LookupHub 归一。
func MarketHub(id string) string {
	if info, ok := LookupHub(id); ok {
		for _, listed := range listedHubIDs {
			if info.ID == listed {
				return info.ID
			}
		}
	}
	return DefaultHub
}
