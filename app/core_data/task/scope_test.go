package task

import "testing"

func TestActiveRefreshConflictCondition(t *testing.T) {
	condition, args := ActiveRefreshConflictCondition("")
	if condition != "current_platform = '' OR total_platforms <> 1" {
		t.Fatalf("full refresh condition = %q", condition)
	}
	if len(args) != 0 {
		t.Fatalf("full refresh args = %v, want empty", args)
	}

	condition, args = ActiveRefreshConflictCondition("AtCoder")
	if condition != "current_platform = ? OR current_platform = '' OR total_platforms <> 1" {
		t.Fatalf("platform refresh condition = %q", condition)
	}
	if len(args) != 1 || args[0] != "AtCoder" {
		t.Fatalf("platform refresh args = %v", args)
	}
}
