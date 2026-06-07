package service

import "testing"

func TestValidateRegisterInviteCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantOK   bool
		wantMsg  string
	}{
		{name: "empty", input: " ", expected: "wustacm666", wantOK: false, wantMsg: "请输入邀请码"},
		{name: "wrong", input: "abc", expected: "wustacm666", wantOK: false, wantMsg: "邀请码错误"},
		{name: "trimmed", input: " wustacm666 ", expected: "wustacm666", wantOK: true},
		{name: "fallback default", input: "wustacm666", expected: " ", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK, gotMsg := validateRegisterInviteCode(tt.input, tt.expected)
			if gotOK != tt.wantOK || gotMsg != tt.wantMsg {
				t.Fatalf("validateRegisterInviteCode() = (%v, %q), want (%v, %q)", gotOK, gotMsg, tt.wantOK, tt.wantMsg)
			}
		})
	}
}
