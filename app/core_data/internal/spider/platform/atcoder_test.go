package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestAtCoderFetchSubmitLogPaginatesAndDeduplicatesBoundary(t *testing.T) {
	originalBaseURL := atCoderAPIBaseURL
	originalPageSize := atCoderPageSize
	defer func() {
		atCoderAPIBaseURL = originalBaseURL
		atCoderPageSize = originalPageSize
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromSecond := r.URL.Query().Get("from_second")
		rows := make([]atcJson, 0)
		switch fromSecond {
		case "0":
			for i := 1; i <= 500; i++ {
				rows = append(rows, atcJson{
					ID:          i,
					EpochSecond: int64(999 + i),
					ProblemID:   "abc_" + strconv.Itoa(i),
					ContestID:   "abc",
					UserID:      "tester",
					Language:    "C++",
					Result:      "AC",
				})
			}
		case "1499":
			rows = append(rows,
				atcJson{ID: 500, EpochSecond: 1499, ProblemID: "abc_500", ContestID: "abc", UserID: "tester", Language: "C++", Result: "AC"},
				atcJson{ID: 501, EpochSecond: 1500, ProblemID: "abc_501", ContestID: "abc", UserID: "tester", Language: "C++", Result: "WA"},
			)
		default:
			rows = []atcJson{}
		}
		_ = json.NewEncoder(w).Encode(rows)
	}))
	defer server.Close()

	atCoderAPIBaseURL = server.URL
	atCoderPageSize = 500

	logs, err := NewAtCoder{}.FetchSubmitLog(9, "tester", true)
	if err != nil {
		t.Fatalf("FetchSubmitLog returned error: %v", err)
	}
	if len(logs) != 501 {
		t.Fatalf("logs length = %d, want 501", len(logs))
	}
	if logs[500].SubmitID != "501" || logs[500].Status != "WA" {
		t.Fatalf("last log = (%s,%s), want (501,WA)", logs[500].SubmitID, logs[500].Status)
	}
}
