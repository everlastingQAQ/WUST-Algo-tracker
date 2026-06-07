package platform

import "testing"

func TestNormalizeLuoGuStatus(t *testing.T) {
	cases := map[int]string{
		12: "AC",
		2:  "CE",
		5:  "TLE",
		6:  "WA",
		7:  "RE",
		11: "WA",
		14: "WA",
		0:  "UNKNOWN",
		1:  "UNKNOWN",
		3:  "OLE",
		4:  "MLE",
		99: "LuoGu:99",
	}
	for input, want := range cases {
		if got := normalizeLuoGuStatus(input); got != want {
			t.Fatalf("normalizeLuoGuStatus(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeQOJStatus(t *testing.T) {
	cases := map[string]string{
		"AC":                  "AC",
		"Accepted":            "AC",
		"CE":                  "CE",
		"Compile Error":       "CE",
		"TLE":                 "TLE",
		"Time Limit Exceeded": "TLE",
		"TL":                  "TLE",
		"WA":                  "WA",
		"Wrong Answer":        "WA",
		"Runtime Error":       "RE",
		"Memory Limit":        "MLE",
		"ML":                  "MLE",
		"0":                   "WA",
		"100 ✓":               "AC",
		"":                    "UNKNOWN",
		"Judging":             "Judging",
	}
	for input, want := range cases {
		if got := normalizeQOJStatus(input); got != want {
			t.Fatalf("normalizeQOJStatus(%q) = %q, want %q", input, got, want)
		}
	}
}
