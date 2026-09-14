package update

import "github.com/Shenchangxin/yoyo/internal/version"

type Channel string

const (
	ChannelStable  Channel = "stable"
	ChannelBeta    Channel = "beta"
	ChannelNightly Channel = "nightly"
)

// BinaryChannel is independent from harness CAS refs. Agent evolution must
// never rotate updater keys or rewrite this package.
type Status struct {
	Version string  `json:"version"`
	Channel Channel `json:"channel"`
}

func Current(ch Channel) Status {
	if ch == "" {
		ch = ChannelNightly
	}
	return Status{Version: version.Version, Channel: ch}
}
