package service

import (
	"testing"
	"time"
)

func TestManualRefreshWindowUsesLongerCodeforcesCooldown(t *testing.T) {
	if got := manualRefreshWindow("CodeForces"); got != 30*time.Minute {
		t.Fatalf("CodeForces cooldown = %s, want 30m", got)
	}
	if got := manualRefreshWindow("codeforces"); got != 30*time.Minute {
		t.Fatalf("case-insensitive CodeForces cooldown = %s, want 30m", got)
	}
	if got := manualRefreshWindow("AtCoder"); got != time.Minute {
		t.Fatalf("default cooldown = %s, want 1m", got)
	}
}
