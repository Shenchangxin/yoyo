// id.go 生成视频文档用的随机标识和 UTC 时间戳。

package video

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewID 读 16 字节随机数，返回其十六进制字符串。
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Now 返回当前 UTC 时间的 RFC3339Nano 文本。
func Now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
