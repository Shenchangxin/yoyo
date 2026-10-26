// 决定一个版本号要写到哪一份通道清单。

package update

import "strings"

// ChannelsForVersion 去掉空白和前缀 v 后，只要还非空就只返回 DefaultChannel。空版本返回 nil。结果里不会出现 stable、beta 或 nightly。
func ChannelsForVersion(version string) []Channel {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if v == "" {
		return nil
	}
	return []Channel{DefaultChannel}
}
