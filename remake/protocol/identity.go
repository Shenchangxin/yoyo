// Copyright 2026 Hypit.AI
// Translated from @hypit/protocol (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package protocol

// ResourceId is an opaque identity for one admitted byte resource.
// Equal bytes may have different identities.
type ResourceId string

type RecordId = string
type LogicalOutputId = string
type CandidateId = string
type OperationId = string
type NeedId = string
type StepId = string
type CommandId = string

type ModuleRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// TypeRef is its owning Module and its name, and nothing else is compared.
// Two Types whose values carry identical shapes are distinct.
type TypeRef struct {
	Module ModuleRef `json:"module"`
	Name   string    `json:"name"`
}

type ProducerRef struct {
	Module ModuleRef `json:"module"`
	Name   string    `json:"name"`
}

type CapabilityRef struct {
	Module ModuleRef `json:"module"`
	Name   string    `json:"name"`
}

func ModuleKey(ref ModuleRef) string {
	return ref.Name + "@" + ref.Version
}

func TypeKey(ref TypeRef) string {
	return ModuleKey(ref.Module) + "#" + ref.Name
}

func ProducerKey(ref ProducerRef) string {
	return ModuleKey(ref.Module) + "#" + ref.Name
}

func CapabilityKey(ref CapabilityRef) string {
	return ModuleKey(ref.Module) + "#" + ref.Name
}

func SameModule(left, right ModuleRef) bool {
	return left.Name == right.Name && left.Version == right.Version
}

func SameType(left, right TypeRef) bool {
	return SameModule(left.Module, right.Module) && left.Name == right.Name
}
