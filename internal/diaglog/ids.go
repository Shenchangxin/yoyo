package diaglog

import (
	"crypto/rand"
	"encoding/hex"
)

func NewTraceID() string {
	return randomHex(16)
}

func NewSpanID() string {
	return randomHex(8)
}

func NewTurnID() string {
	return randomHex(8)
}

func NewRequestID() string {
	return randomHex(8)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(i * 17)
		}
	}
	return hex.EncodeToString(b)
}
