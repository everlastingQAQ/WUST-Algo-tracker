package dal

import "testing"

func TestIsAcceptedStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "short ac", status: "AC", want: true},
		{name: "accepted", status: "Accepted", want: true},
		{name: "ok", status: "OK", want: true},
		{name: "chinese", status: "答案正确", want: true},
		{name: "wrong answer", status: "WA", want: false},
		{name: "time limit", status: "TLE", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAcceptedStatus(tt.status); got != tt.want {
				t.Fatalf("IsAcceptedStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestBuildProblemDistinctKey(t *testing.T) {
	if got := BuildProblemDistinctKey(4, "NowCoder", "A+B", "s1"); got != "4|NowCoder|A+B" {
		t.Fatalf("problem key = %q", got)
	}
	if got := BuildProblemDistinctKey(4, "NowCoder", "  ", "s1"); got != "4|NowCoder|s1" {
		t.Fatalf("fallback key = %q", got)
	}
	if got := BuildProblemDistinctKey(4, "CodeForces", "A+B", "s2"); got == "4|NowCoder|A+B" {
		t.Fatalf("cross-platform problems must not merge")
	}
}
