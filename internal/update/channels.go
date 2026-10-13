package update

import "strings"

// ChannelsForVersion picks which feed files a release should overwrite.
// Yoyo ships a single latest pointer; older stable/beta/nightly names are not published.
func ChannelsForVersion(version string) []Channel {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if v == "" {
		return nil
	}
	return []Channel{DefaultChannel}
}
