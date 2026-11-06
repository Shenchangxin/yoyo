// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import "github.com/Shenchangxin/yoyo/remake/protocol"

func invariant(condition bool, code, message string, subject ...string) {
	if condition {
		return
	}
	var owner *string
	if len(subject) > 0 {
		value := subject[0]
		owner = &value
	}
	panic(&protocol.SvmlError{Code: code, Message: message, Subject: owner})
}
