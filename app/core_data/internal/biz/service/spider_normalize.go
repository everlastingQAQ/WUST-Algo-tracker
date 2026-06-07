package service

import (
	"cwxu-algo/app/core_data/internal/data/model"
	"fmt"
	"strings"
)

func normalizeFetchedSubmitLogs(userId int64, platform string, logs []model.SubmitLog) ([]model.SubmitLog, int, error) {
	normalized := make([]model.SubmitLog, 0, len(logs))
	indexBySubmitID := make(map[string]int, len(logs))
	skipped := 0

	for _, item := range logs {
		submitID := strings.TrimSpace(item.SubmitID)
		if submitID == "" || item.Time.IsZero() {
			skipped++
			continue
		}

		item.UserID = userId
		item.Platform = platform
		item.SubmitID = submitID
		item.Contest = strings.TrimSpace(item.Contest)
		item.Problem = strings.TrimSpace(item.Problem)
		item.Lang = strings.TrimSpace(item.Lang)
		item.Status = strings.TrimSpace(item.Status)

		if existingIndex, ok := indexBySubmitID[submitID]; ok {
			normalized[existingIndex] = item
			skipped++
			continue
		}
		indexBySubmitID[submitID] = len(normalized)
		normalized = append(normalized, item)
	}

	if len(logs) > 0 && len(normalized) == 0 {
		return nil, skipped, fmt.Errorf("%s 抓取结果全部无效，已拒绝写入", platform)
	}
	return normalized, skipped, nil
}
