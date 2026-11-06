// Copyright 2026 Hypit.AI
// Translated from @hypit/kernel (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package kernel

import "github.com/Shenchangxin/yoyo/remake/protocol"

func moduleKey(ref protocol.ModuleRef) string { return protocol.ModuleKey(ref) }

func typeKey(ref protocol.TypeRef) string { return protocol.TypeKey(ref) }

func producerKey(ref protocol.ProducerRef) string { return protocol.ProducerKey(ref) }

func capabilityKey(ref protocol.CapabilityRef) string { return protocol.CapabilityKey(ref) }

func sameModule(left, right protocol.ModuleRef) bool { return protocol.SameModule(left, right) }

func sameType(left, right protocol.TypeRef) bool { return protocol.SameType(left, right) }
