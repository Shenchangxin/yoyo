// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

// SvmlError is the typed failure thrown by protocol and kernel checks.
type SvmlError struct {
	Code    string
	Message string
	Subject *string
}

func (e *SvmlError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
