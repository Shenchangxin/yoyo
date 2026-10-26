// 当前二进制对外报出的版本和通道。

package update

import "github.com/Shenchangxin/yoyo/internal/version"

// Channel 是通道名。
type Channel string

const (
	// ChannelStable 是名字 stable。本包发布和轮询不用它。
	ChannelStable Channel = "stable"
	// ChannelBeta 是名字 beta。本包发布和轮询不用它。
	ChannelBeta Channel = "beta"
	// ChannelNightly 是名字 nightly。本包发布和轮询不用它。
	ChannelNightly Channel = "nightly"
	// ChannelLatest 是名字 latest。
	ChannelLatest Channel = "latest"
)

// DefaultChannel 是本包唯一发布和轮询的通道，值为 latest。
const DefaultChannel = ChannelLatest

// Status 是当前二进制报出的版本和通道。
type Status struct {
	Version string  `json:"version"`
	Channel Channel `json:"channel"`
}

// Current 忽略传入的通道，返回 version.Version 和 DefaultChannel。
// ch 赋给空白标识符，不另作使用。
func Current(ch Channel) Status {
	_ = ch
	return Status{Version: version.Version, Channel: DefaultChannel}
}
