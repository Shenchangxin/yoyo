package remote

import "testing"

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

func TestStepUpIsExplicit(t *testing.T) {
	if !NeedsConfirm("key.set") || !NeedsConfirm("harness.checkout") || !NeedsConfirm("update.apply") {
		t.Fatal("secrets and promotion need a confirmation")
	}
	if NeedsConfirm("thread.model.set") || NeedsConfirm("thread.start") || NeedsConfirm("config.set") {
		t.Fatal("daily controls stay one tap")
	}
}
