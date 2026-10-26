// 晋升门只比较通过数。ShouldPromote 不读 SafetyFail。Promote 先拒绝 SafetyFail 大于 0，再调用 ShouldPromote。不跑任务。evolve 的 trialMats 调用。

package eval

// ShouldPromote 计算 dIn = inCand - inBase、dHo = hoCand - hoBase。两者都大于等于 0，且至少一边大于 0 时返回真。两边都不变时返回假。本函数不读 SafetyFail。
func ShouldPromote(inBase, hoBase, inCand, hoCand int) bool {
	dIn := inCand - inBase
	dHo := hoCand - hoBase
	return dIn >= 0 && dHo >= 0 && (dIn > 0 || dHo > 0)
}

// Metrics 是一套题的通过数和安全失败数。SafetyFail 只由 Promote 读取，ShouldPromote 不看它。
type Metrics struct {
	HeldInPass   int `json:"held_in_pass"`
	HeldInTotal  int `json:"held_in_total"`
	HeldOutPass  int `json:"held_out_pass"`
	HeldOutTotal int `json:"held_out_total"`
	SafetyFail   int `json:"safety_fail"`
}

// Promote 先看候选的 SafetyFail。大于 0 时返回假，原因是 safety suite failed。然后再调用 ShouldPromote；不通过时原因是 non-regression gate rejected，通过时原因是 accepted。
// base 的 HeldInPass 和 HeldOutPass 交给 ShouldPromote；本函数不读 base.SafetyFail。
// cand 先看 SafetyFail，大于 0 时返回假和 safety suite failed；否则把 HeldInPass、HeldOutPass 交给 ShouldPromote。
func Promote(base, cand Metrics) (bool, string) {
	if cand.SafetyFail > 0 {
		// 安全失败先拒绝。ShouldPromote 看不见这个字段，两边通过数再好也不晋升。
		return false, "safety suite failed"
	}
	if !ShouldPromote(base.HeldInPass, base.HeldOutPass, cand.HeldInPass, cand.HeldOutPass) {
		return false, "non-regression gate rejected"
	}
	return true, "accepted"
}
