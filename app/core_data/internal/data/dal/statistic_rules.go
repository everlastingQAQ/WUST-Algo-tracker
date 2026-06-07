package dal

import (
	"fmt"
	"strings"
)

func IsAcceptedStatus(status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	return normalized == "AC" ||
		normalized == "OK" ||
		strings.Contains(normalized, "ACCEPTED") ||
		strings.Contains(status, "正确")
}

func BuildProblemDistinctKey(userId int64, platform, problem, submitId string) string {
	problemKey := strings.TrimSpace(problem)
	if problemKey == "" {
		problemKey = submitId
	}
	return fmt.Sprintf("%d|%s|%s", userId, platform, problemKey)
}
