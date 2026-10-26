// errors.go 表示一条协议当前不可用。

package protocol

import "fmt"

// UnavailableError 带上协议标识和原因。
type UnavailableError struct {
	Protocol string
	Reason   string
}

// Error 在原因为空时只说明协议不可用，否则把原因接在后面。
func (e UnavailableError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("protocol %s is unavailable", e.Protocol)
	}
	return fmt.Sprintf("protocol %s is unavailable: %s", e.Protocol, e.Reason)
}

// unavailable 用元数据里的标识和 UnavailableReason 构造 UnavailableError。
func unavailable(metadata Metadata) error {
	return UnavailableError{Protocol: metadata.ID, Reason: metadata.UnavailableReason}
}
