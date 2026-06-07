package service

import (
	"strings"
	"testing"
)

func TestNormalizeMessageContent(t *testing.T) {
	got, err := normalizeMessageContent("  hello  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("trimmed content = %q", got)
	}

	if _, err := normalizeMessageContent("   "); err == nil {
		t.Fatalf("empty message should fail")
	}

	if _, err := normalizeMessageContent(strings.Repeat("你", 1000)); err != nil {
		t.Fatalf("1000-rune message should pass: %v", err)
	}

	if _, err := normalizeMessageContent(strings.Repeat("你", 1001)); err == nil {
		t.Fatalf("1001-rune message should fail")
	}
}
