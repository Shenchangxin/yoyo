package app

import (
	"testing"
	"time"
)

func TestEvolveTimeoutCapsAtTwoHours(t *testing.T) {
	if got := EvolveTimeout(0); got != 15*time.Minute {
		t.Fatalf("rounds 0: %s", got)
	}
	if got := EvolveTimeout(1); got != 15*time.Minute {
		t.Fatalf("one round: %s", got)
	}
	if got := EvolveTimeout(4); got != time.Hour {
		t.Fatalf("four rounds: %s", got)
	}
	if got := EvolveTimeout(20); got != 2*time.Hour {
		t.Fatalf("twenty rounds: %s", got)
	}
}

func TestOverlayTimeoutMatchesSharedPolicy(t *testing.T) {
	if got := OverlayTimeout(0, 0); got != 15*time.Minute {
		t.Fatalf("k 0: %s", got)
	}
	if got := OverlayTimeout(2, 0); got != 30*time.Minute {
		t.Fatalf("k 2: %s", got)
	}
	if got := OverlayTimeout(10, 0); got != time.Hour {
		t.Fatalf("k 10: %s", got)
	}
	if got := OverlayTimeout(1, 30*time.Minute); got != 30*time.Minute {
		t.Fatalf("wall extends: %s", got)
	}
	if got := OverlayTimeout(1, 2*time.Hour); got != time.Hour {
		t.Fatalf("wall capped: %s", got)
	}
}
