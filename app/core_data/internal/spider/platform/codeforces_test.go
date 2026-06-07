package platform

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCodeforcesFetchSubmitLogPaginatesNeedAll(t *testing.T) {
	oldBaseURL := codeforcesAPIBaseURL
	oldPageSize := codeforcesPageSize
	defer func() {
		codeforcesAPIBaseURL = oldBaseURL
		codeforcesPageSize = oldPageSize
	}()

	codeforcesPageSize = 2
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("handle") != "jiangly" {
			t.Fatalf("handle = %q", r.URL.Query().Get("handle"))
		}
		switch r.URL.Query().Get("from") {
		case "1":
			_, _ = fmt.Fprint(w, `{"status":"OK","result":[
				{"id":101,"contestId":1,"problem":{"index":"A","name":"One"},"programmingLanguage":"GNU C++20","verdict":"OK","creationTimeSeconds":1000},
				{"id":102,"contestId":1,"problem":{"index":"B","name":"Two"},"programmingLanguage":"GNU C++20","verdict":"WRONG_ANSWER","creationTimeSeconds":1001}
			]}`)
		case "3":
			_, _ = fmt.Fprint(w, `{"status":"OK","result":[
				{"id":103,"contestId":2,"problem":{"index":"C","name":"Three"},"programmingLanguage":"GNU C++20","verdict":"OK","creationTimeSeconds":1002}
			]}`)
		default:
			t.Fatalf("unexpected from=%s", r.URL.Query().Get("from"))
		}
	}))
	defer server.Close()
	codeforcesAPIBaseURL = server.URL

	logs, err := NewCodeforces{}.FetchSubmitLog(7, "jiangly", true)
	if err != nil {
		t.Fatalf("FetchSubmitLog returned error: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("len(logs) = %d, want 3", len(logs))
	}
	if logs[0].UserID != 7 || logs[0].Platform != "CodeForces" || logs[2].SubmitID != "103" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestCodeforcesFetchSubmitLogRecentOnlyUsesSinglePage(t *testing.T) {
	oldBaseURL := codeforcesAPIBaseURL
	defer func() { codeforcesAPIBaseURL = oldBaseURL }()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = fmt.Fprint(w, `{"status":"OK","result":[
			{"id":201,"contestId":1,"problem":{"index":"A","name":"One"},"programmingLanguage":"GNU C++20","verdict":"OK","creationTimeSeconds":1000}
		]}`)
	}))
	defer server.Close()
	codeforcesAPIBaseURL = server.URL

	logs, err := NewCodeforces{}.FetchSubmitLog(7, "tourist", false)
	if err != nil {
		t.Fatalf("FetchSubmitLog returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
}

func TestBuildCodeforcesProblemKeyIncludesContestIdentity(t *testing.T) {
	first := cfJson{ContestID: 1}
	first.Problem.Index = "A"
	first.Problem.Name = "Game"

	second := cfJson{ContestID: 2}
	second.Problem.Index = "A"
	second.Problem.Name = "Game"

	if buildCodeforcesProblemKey(first) == buildCodeforcesProblemKey(second) {
		t.Fatalf("same index/name from different contests should not collapse: %q", buildCodeforcesProblemKey(first))
	}
	if got := buildCodeforcesProblemKey(first); got != "1-A Game" {
		t.Fatalf("buildCodeforcesProblemKey(first) = %q, want %q", got, "1-A Game")
	}
}

func TestBuildCodeforcesProblemKeyUsesProblemsetName(t *testing.T) {
	row := cfJson{}
	row.Problem.ProblemsetName = "acmsguru"
	row.Problem.Index = "108"
	row.Problem.Name = "Self-numbers II"

	if got := buildCodeforcesProblemKey(row); got != "acmsguru-108 Self-numbers II" {
		t.Fatalf("buildCodeforcesProblemKey(row) = %q", got)
	}
}
