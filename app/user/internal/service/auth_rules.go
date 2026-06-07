package service

import "strings"

func validateRegisterInviteCode(input, expected string) (bool, string) {
	input = strings.TrimSpace(input)
	expected = strings.TrimSpace(expected)
	if input == "" {
		return false, "请输入邀请码"
	}
	if expected == "" {
		expected = defaultRegisterInviteCode
	}
	if input != expected {
		return false, "邀请码错误"
	}
	return true, ""
}
