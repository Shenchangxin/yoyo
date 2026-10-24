package remote

import "strings"

// PhoneDenied is the operator-only pairing surface. A paired phone can revoke
// a device, and it cannot mint a new pairing or shut the listener.
func PhoneDenied(method string) bool {
	switch strings.TrimSpace(method) {
	case "remote.pair.begin", "remote.pair.approve", "remote.pair.deny", "remote.disable", "remote.enable":
		return true
	default:
		return false
	}
}

// NeedsConfirm is the irreversible or secret-writing set. The phone must send
// confirm:true after its own step-up. Daily work, including switching models,
// does not.
func NeedsConfirm(method string) bool {
	switch strings.TrimSpace(method) {
	case "key.set", "provider.upsert", "provider.delete",
		"workspace.apply_hunks", "workspace.reverse_hunks",
		"harness.checkout", "harness.rollback", "evolve.run", "update.apply",
		"mcp.replace", "thread.delete", "computer.allow", "pages.delete",
		"packs.uninstall", "skills.uninstall",
		"video.providers.upsert", "video.providers.delete",
		"connections.upsert", "connections.delete":
		return true
	default:
		return false
	}
}
