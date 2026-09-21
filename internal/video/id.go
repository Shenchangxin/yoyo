package video

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
