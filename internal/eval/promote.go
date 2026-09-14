package eval

// ShouldPromote implements the Self-Harness non-regression gate:
// held-in and held-out must not drop, and at least one split must improve.
func ShouldPromote(inBase, hoBase, inCand, hoCand int) bool {
	dIn := inCand - inBase
	dHo := hoCand - hoBase
	return dIn >= 0 && dHo >= 0 && (dIn > 0 || dHo > 0)
}

type Metrics struct {
	HeldInPass   int `json:"held_in_pass"`
	HeldInTotal  int `json:"held_in_total"`
	HeldOutPass  int `json:"held_out_pass"`
	HeldOutTotal int `json:"held_out_total"`
	SafetyFail   int `json:"safety_fail"`
}

func Promote(base, cand Metrics) (bool, string) {
	if cand.SafetyFail > 0 {
		return false, "safety suite failed"
	}
	if !ShouldPromote(base.HeldInPass, base.HeldOutPass, cand.HeldInPass, cand.HeldOutPass) {
		return false, "non-regression gate rejected"
	}
	return true, "accepted"
}
