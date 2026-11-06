package remake

// UpstreamCommit is the Hypit commit this tree translates.
// Describe at pin time: v0.3.1-17-ge26d9142.
const UpstreamCommit = "e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16"

// UpstreamDescribe is the git describe of UpstreamCommit.
const UpstreamDescribe = "v0.3.1-17-ge26d9142"

// IncludeInInstaller reports whether yoyo/remake may be packaged into a Yoyo
// build supplied to third parties. The shipping 翻拍 feature is Yoyo's own
// cut flow, not this translation, so this stays false.
func IncludeInInstaller() bool { return false }
