// 本文件生成日志和跨度用的随机十六进制 id。
// NewTraceID 读 16 字节，其余三个读 8 字节。crypto/rand 失败时用下标乘 17 填满缓冲区。

package diaglog

import (
	"crypto/rand"
	"encoding/hex"
)

// NewTraceID 返回 16 个随机字节的十六进制，长度是 32。
func NewTraceID() string {
	return randomHex(16)
}

// NewSpanID 返回 8 个随机字节的十六进制，长度是 16。
func NewSpanID() string {
	return randomHex(8)
}

// NewTurnID 返回 8 个随机字节的十六进制，长度是 16。
func NewTurnID() string {
	return randomHex(8)
}

// NewRequestID 返回 8 个随机字节的十六进制，长度是 16。
func NewRequestID() string {
	return randomHex(8)
}

// randomHex 读取 n 个随机字节并编成十六进制。rand 失败时把第 i 个字节设成 byte(i * 17)。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(i * 17)
		}
	}
	return hex.EncodeToString(b)
}
