// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

func assert(cond bool, message string) {
	if !cond {
		panic(errors.New(message))
	}
}

// OrderedBuildId creates the one public, chronologically sortable Build identity.
// The nonce keeps independent submissions in the same millisecond distinct.
// It is not a content digest.
func OrderedBuildId(createdAt int64, nonce string) string {
	if !safeTime(createdAt) {
		assert(false, "Build time must be a non-negative safe integer")
	}
	if !nonceOK(nonce) {
		assert(false, "Build nonce must be ten upper-case alphanumeric characters")
	}
	return "bld_" + timestampText(createdAt) + "_" + nonce
}

// BuildIdCreatedAt is the submission time carried by an ordered Build id.
// It reports ok=false for a non-public or internal id.
func BuildIdCreatedAt(value string) (int64, bool) {
	if len(value) != len("bld_20260902T102030123Z_0000000001") {
		return 0, false
	}
	if value[:4] != "bld_" || value[23] != '_' {
		return 0, false
	}
	raw := value[4:23]
	nonce := value[24:]
	if !nonceOK(nonce) || !timestampShape(raw) {
		return 0, false
	}
	iso := raw[0:4] + "-" + raw[4:6] + "-" + raw[6:8] + "T" + raw[9:11] + ":" + raw[11:13] + ":" + raw[13:15] + "." + raw[15:18] + "Z"
	created, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return 0, false
	}
	ms := created.UnixMilli()
	if timestampText(ms) != raw {
		return 0, false
	}
	return ms, true
}

func AssertOrderedBuildId(value string) {
	AssertBuildId(value)
	if _, ok := BuildIdCreatedAt(value); !ok {
		assert(false, "Build id must contain its UTC submission time and a ten-character nonce")
	}
}

// AssertBuildId checks the filesystem boundary of a Build id.
// An id is never a path, and the two dot directory entries are never Builds.
func AssertBuildId(value string) {
	assert(len(value) > 0 && value == strings.TrimSpace(value), "Build id must be a non-empty unpadded name")
	assert(value != "." && value != "..", "Build id must not be a dot directory")
	for _, r := range value {
		assert(r != '/' && r != '\\' && r != 0, "Build id must be one path-free name")
	}
}

func nonceOK(nonce string) bool {
	if len(nonce) != 10 {
		return false
	}
	for _, r := range nonce {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	return true
}

func timestampShape(raw string) bool {
	if len(raw) != 19 || raw[8] != 'T' || raw[18] != 'Z' {
		return false
	}
	for i, r := range raw {
		if i == 8 || i == 18 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func safeTime(value int64) bool {
	return value >= 0 && value <= math.MaxInt64 && value <= 9007199254740991
}

func timestampText(value int64) string {
	t := time.UnixMilli(value).UTC()
	return fmt.Sprintf("%s%03dZ", t.Format("20060102T150405"), t.Nanosecond()/1e6)
}
