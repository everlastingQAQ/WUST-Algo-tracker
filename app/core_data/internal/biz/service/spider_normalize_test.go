package service

import (
	"cwxu-algo/app/core_data/internal/data/model"
	"testing"
	"time"
)

func TestNormalizeFetchedSubmitLogs(t *testing.T) {
	logs := []model.SubmitLog{
		{UserID: 9, Platform: "Wrong", SubmitID: " 1 ", Problem: " A ", Status: " AC ", Time: time.Unix(100, 0)},
		{UserID: 9, Platform: "Wrong", SubmitID: "1", Problem: "A2", Status: "WA", Time: time.Unix(101, 0)},
		{SubmitID: "", Time: time.Unix(102, 0)},
		{SubmitID: "2"},
	}

	normalized, skipped, err := normalizeFetchedSubmitLogs(7, "CodeForces", logs)
	if err != nil {
		t.Fatalf("normalizeFetchedSubmitLogs returned error: %v", err)
	}
	if skipped != 3 {
		t.Fatalf("skipped = %d, want 3", skipped)
	}
	if len(normalized) != 1 {
		t.Fatalf("len(normalized) = %d, want 1", len(normalized))
	}
	item := normalized[0]
	if item.UserID != 7 || item.Platform != "CodeForces" || item.SubmitID != "1" {
		t.Fatalf("normalized identity = (%d,%s,%s)", item.UserID, item.Platform, item.SubmitID)
	}
	if item.Problem != "A2" || item.Status != "WA" {
		t.Fatalf("duplicate should keep latest row, got problem=%q status=%q", item.Problem, item.Status)
	}
}

func TestNormalizeFetchedSubmitLogsRejectsAllInvalid(t *testing.T) {
	_, _, err := normalizeFetchedSubmitLogs(7, "CodeForces", []model.SubmitLog{
		{SubmitID: "", Time: time.Unix(100, 0)},
		{SubmitID: "2"},
	})
	if err == nil {
		t.Fatal("expected all-invalid result to be rejected")
	}
}

func TestRejectIncompleteFullFetch(t *testing.T) {
	if err := rejectIncompleteFullFetch("CodeForces", 999, 1000, true); err == nil {
		t.Fatal("expected incomplete full fetch to be rejected")
	}
	if err := rejectIncompleteFullFetch("CodeForces", 1000, 1000, true); err != nil {
		t.Fatalf("same-size full fetch should be accepted: %v", err)
	}
	if err := rejectIncompleteFullFetch("CodeForces", 100, 1000, false); err != nil {
		t.Fatalf("recent incremental fetch should be accepted: %v", err)
	}
	if err := rejectIncompleteFullFetch("CodeForces", 0, 0, true); err != nil {
		t.Fatalf("first full fetch should be accepted: %v", err)
	}
}
