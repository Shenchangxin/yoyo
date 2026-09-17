package session

import "time"

// HarnessPolicy controls which snapshot a session executes against.
const (
	FollowActive = "follow_active"
	Pin          = "pin"
)

// TurnStatus is the durable turn state machine.
const (
	StatusIdle              = "idle"
	StatusRunning           = "running"
	StatusAwaitingApproval  = "awaiting_approval"
	StatusCompacting        = "compacting"
	StatusCancelled         = "cancelled"
)

// Meta is the durable session identity. ID never changes.
type Meta struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Workspace        string    `json:"workspace"`
	Harness          string    `json:"harness"`
	HarnessPolicy    string    `json:"harness_policy,omitempty"`
	ModelFingerprint string    `json:"model_fingerprint"`
	Title            string    `json:"title,omitempty"`
	Archived         bool      `json:"archived"`
	Pinned           bool      `json:"pinned"`
	Model            string    `json:"model,omitempty"`
	LoadedSkills     []string  `json:"loaded_skills,omitempty"`
	PlanText         string    `json:"plan_text,omitempty"`
}

// ResolveHarness returns the snapshot hash this session must execute.
// follow_active (default for new chats) tracks refs/active.
// pin (forks, eval, replay) keeps meta.Harness even after promotion.
func ResolveHarness(policy, pinned, active string) string {
	if policy == Pin && pinned != "" {
		return pinned
	}
	if active != "" {
		return active
	}
	return pinned
}

func NormalizePolicy(p string) string {
	switch p {
	case Pin:
		return Pin
	default:
		return FollowActive
	}
}
