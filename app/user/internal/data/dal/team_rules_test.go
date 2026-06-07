package dal

import "testing"

func TestCanManageTeam(t *testing.T) {
	if !CanManageTeam(7, 7) {
		t.Fatalf("owner should manage team")
	}
	if CanManageTeam(6, 7) {
		t.Fatalf("non-owner must not manage team")
	}
	if CanManageTeam(0, 7) || CanManageTeam(7, 0) {
		t.Fatalf("zero ids must not manage team")
	}
}
