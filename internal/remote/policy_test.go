// 锁住手机不能签发、允许或关掉远程，但 PhoneDenied 对换模型和 remote.revoke 返回假。
// key.set、harness.checkout、update.apply 要确认。thread.model.set、thread.start、config.set 不要。

package remote

import "testing"

// TestPhoneCannotMintPairing 锁住 remote.pair.begin、remote.pair.approve、remote.disable 被拒绝，换模型和收回不被拒绝。
func TestPhoneCannotMintPairing(t *testing.T) {
	for _, method := range []string{"remote.pair.begin", "remote.pair.approve", "remote.disable"} {
		if !PhoneDenied(method) {
			t.Fatalf("%s must stay on the computer", method)
		}
	}
	if PhoneDenied("thread.model.set") || PhoneDenied("remote.revoke") {
		t.Fatal("model changes and revoke are operator actions")
	}
}

// TestStepUpIsExplicit 锁住写密钥、检出和安装更新要确认，换模型、开会话和 config.set 不用确认。
func TestStepUpIsExplicit(t *testing.T) {
	if !NeedsConfirm("key.set") || !NeedsConfirm("harness.checkout") || !NeedsConfirm("update.apply") {
		t.Fatal("secrets and promotion need a confirmation")
	}
	if NeedsConfirm("thread.model.set") || NeedsConfirm("thread.start") || NeedsConfirm("config.set") {
		t.Fatal("daily controls stay one tap")
	}
}
