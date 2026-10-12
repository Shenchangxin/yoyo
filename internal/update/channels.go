package update

import "strings"

// ChannelsForVersion picks which feed files a release should overwrite.
// Formal tags update stable+beta+nightly so every client channel moves forward.
// Prereleases skip stable. Dev/workflow builds only move nightly.
func ChannelsForVersion(version string) []Channel {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if v == "" {
		return nil
	}
	if strings.Contains(v, "-dev.") || strings.HasPrefix(v, "0.0.0-dev") {
		return []Channel{ChannelNightly}
	}
	if strings.Contains(v, "-") {
		return []Channel{ChannelBeta, ChannelNightly}
	}
	return []Channel{ChannelStable, ChannelBeta, ChannelNightly}
}
