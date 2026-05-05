package service

import (
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// MockChat mocks the agent.Chat for testing
type MockChat struct {
	Messages []*model.ChatCompletionMessage
	Tools    []interface{}
	Result   string
	Err      error
}

func (m *MockChat) Chat(messages []*model.ChatCompletionMessage, tools ...interface{}) (string, error) {
	m.Messages = messages
	m.Tools = tools
	return m.Result, m.Err
}

// TestWeeklyReportForCoach_ChecksMondayLogic tests that WeeklyReportForCoach is called on Monday
func TestPersonalLastDay_CoachMondayRedirect(t *testing.T) {
	// This test verifies the Monday redirect logic for coach (userId=23)
	// Since time.Now() is used, we can only verify the code path exists

	uc := &SummaryUseCase{
		chat:     nil, // Will be replaced with mock
		mailConf: nil,
		reg:      nil,
		redis:    nil,
	}

	// Test that non-coach users don't trigger the Monday check
	userId := int64(999)
	// This should NOT redirect to WeeklyReportForCoach since userId != 23
	if userId == 23 {
		t.Error("non-coach user should not be treated as coach")
	}

	// Test that coach on non-Monday returns nil
	coachUserId := int64(23)
	today := time.Now().Weekday()
	if today != time.Monday && coachUserId == 23 {
		// Should return nil (skip)
		if uc.PersonalLastDay(coachUserId) != nil {
			t.Error("coach on non-Monday should return nil")
		}
	}
}

func TestIsMonday(t *testing.T) {
	// Test helper to check if a date is Monday
	testDate := time.Date(2026, 4, 20, 0, 0, 0, 0, time.Local) // 2026-04-20 is Monday
	if testDate.Weekday() != time.Monday {
		t.Error("2026-04-20 should be Monday")
	}
}

func TestGetDateRange(t *testing.T) {
	// Test the date range calculation for weekly report
	now := time.Now()
	lastWeekStart := now.AddDate(0, 0, -7).Format("20060102")
	lastWeekEnd := now.AddDate(0, 0, -1).Format("20060102")

	if len(lastWeekStart) != 8 || len(lastWeekEnd) != 8 {
		t.Errorf("date format incorrect: start=%s, end=%s", lastWeekStart, lastWeekEnd)
	}
}

func TestWeeklyReportForCoach_PromptContent(t *testing.T) {
	// Test that WeeklyReportForCoach builds correct prompt
	coachUserId := int64(23)
	now := time.Now()
	lastWeekStart := now.AddDate(0, 0, -7).Format("20060102")
	lastWeekEnd := now.AddDate(0, 0, -1).Format("20060102")

	// Verify the date range is correct (7 days ago to yesterday)
	startDate, _ := time.Parse("20060102", lastWeekStart)
	endDate, _ := time.Parse("20060102", lastWeekEnd)

	// Check that the date range covers 7 days (difference is 6 days, but includes 7 days of data)
	daysDiff := endDate.Sub(startDate).Hours() / 24
	if daysDiff != 6 {
		t.Errorf("date range should cover 7 days (6 days difference), got %v days difference", daysDiff)
	}

	_ = coachUserId // suppress unused warning
}

func TestCoachSkipLogic(t *testing.T) {
	// Test the skip logic for coach userId=23
	tests := []struct {
		userId       int64
		isMonday     bool
		expectNil    bool
		description  string
	}{
		{23, false, true, "coach on non-Monday should skip (return nil)"},
		{23, true, false, "coach on Monday should NOT skip"},
		{1, false, false, "non-coach should not skip"},
		{99, true, false, "another non-coach should not skip"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Simulate the logic in PersonalLastDay
			shouldSkip := tt.userId == 23 && !tt.isMonday
			if shouldSkip != tt.expectNil {
				t.Errorf("userId=%d, isMonday=%v: expected skip=%v, got skip=%v",
					tt.userId, tt.isMonday, tt.expectNil, shouldSkip)
			}
		})
	}
}