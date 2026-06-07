package service

import (
	"context"
	"cwxu-algo/app/core_data/internal/data"
	"cwxu-algo/app/core_data/internal/data/model"
	"cwxu-algo/app/core_data/internal/spider"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SpiderUseCase struct {
	data *data.Data
}

func NewSpiderUseCase(data *data.Data) *SpiderUseCase {
	return &SpiderUseCase{
		data: data,
	}
}

// LoadData 加载数据。targetPlatform 为空时刷新全部绑定平台。
func (uc *SpiderUseCase) LoadData(jobId int64, userId int64, needAll bool, targetPlatform string) error {
	// 无论如何，函数退出前一定删缓存
	defer uc.invalidateCache(userId)

	var platforms []model.Platform
	query := uc.data.DB.Where("user_id = ?", userId)
	if targetPlatform != "" {
		query = query.Where("platform = ?", targetPlatform)
	}
	if err := query.Find(&platforms).Error; err != nil {
		uc.finishJob(jobId, "failed", err.Error())
		return err
	}
	if len(platforms) == 0 {
		err := fmt.Errorf("未绑定平台 %s", targetPlatform)
		if targetPlatform == "" {
			err = fmt.Errorf("未绑定 OJ 平台")
		}
		uc.finishJob(jobId, "failed", err.Error())
		return err
	}

	uc.startJob(jobId, len(platforms))
	var failed []string
	for _, plat := range platforms {
		uc.setCurrentPlatform(jobId, plat.Platform)
		count, err := uc.loadOnePlatform(userId, plat, needAll)
		uc.finishPlatform(jobId, plat.Platform)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", plat.Platform, err))
			uc.markPlatformFailed(userId, plat, err)
			continue
		}
		uc.markPlatformSuccess(userId, plat, count)
	}

	if len(failed) > 0 {
		errText := strings.Join(failed, "; ")
		uc.finishJob(jobId, "failed", errText)
		return fmt.Errorf("%s", errText)
	}

	uc.finishJob(jobId, "success", "")
	return nil
}

func (uc *SpiderUseCase) fetchAndSave(userId int64, plat model.Platform, needAll bool) (int, error) {
	p, ok := spider.Get(plat.Platform)
	if !ok {
		return 0, fmt.Errorf("平台插件不存在")
	}
	sbFetch, ok := p.(spider.SubmitLogFetcher)
	if !ok {
		return 0, fmt.Errorf("平台未实现 SubmitLogFetcher")
	}
	tmp, err := sbFetch.FetchSubmitLog(userId, plat.Username, needAll)
	if err != nil {
		return 0, err
	}
	if len(tmp) == 0 {
		return 0, nil
	}

	err = uc.data.DB.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "platform"}, {Name: "submit_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id",
				"contest",
				"problem",
				"lang",
				"status",
				"time",
			}),
		}).
		Save(&tmp).Error
	return len(tmp), err
}

func (uc *SpiderUseCase) fetchAndSaveContest(userId int64, plat model.Platform, needAll bool) error {
	p, ok := spider.Get(plat.Platform)
	if !ok {
		return fmt.Errorf("平台插件不存在")
	}
	sbFetch, ok := p.(spider.SubmitContestFetcher)
	if !ok {
		return fmt.Errorf("平台未实现 SubmitContestFetcher")
	}
	tmp, err := sbFetch.FetchContestLog(userId, plat.Username, needAll)
	if err != nil {
		return err
	}
	if len(tmp) == 0 {
		return nil
	}

	return uc.data.DB.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "contest_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).
		Save(&tmp).Error
}

func (uc *SpiderUseCase) loadOnePlatform(userId int64, plat model.Platform, needAll bool) (int, error) {
	uc.markPlatformRunning(userId, plat)
	// 限制最大重试次数
	maxRetries := 12
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		count, err := uc.fetchAndSave(userId, plat, needAll)
		if contestErr := uc.fetchAndSaveContest(userId, plat, needAll); contestErr != nil {
			log.Errorf("Spider: fetchAndSaveContest %s %s 失败: %v", plat.Platform, plat.Username, contestErr)
		}
		if err == nil {
			log.Infof("Spider: %s %s 成功", plat.Platform, plat.Username)
			uc.invalidateCache(userId)
			return count, nil
		}
		lastErr = err
		if strings.Contains(err.Error(), "平台") {
			log.Errorf(
				"Spider: %s %s 失败: %v",
				plat.Platform,
				plat.Username,
				err,
			)
			return 0, err
		}
		log.Errorf(
			"Spider: %s %s 失败 (重试 %d/%d): %v",
			plat.Platform,
			plat.Username,
			i+1,
			maxRetries,
			err,
		)
		// needAll=false，不重试
		if !needAll {
			return 0, err
		}
		// needAll=true，重试最多12次
		time.Sleep(5 * time.Second)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("达到最大重试次数 %d", maxRetries)
	}
	log.Errorf("Spider: %s %s 达到最大重试次数 %d", plat.Platform, plat.Username, maxRetries)
	return 0, lastErr
}

func (uc *SpiderUseCase) startJob(jobId int64, totalPlatforms int) {
	if jobId <= 0 {
		return
	}
	now := time.Now()
	_ = uc.data.DB.Model(&model.SpiderRefreshJob{}).Where("id = ?", jobId).Updates(map[string]interface{}{
		"status":             "running",
		"started_at":         &now,
		"total_platforms":    int32(totalPlatforms),
		"finished_platforms": int32(0),
		"error":              "",
	}).Error
}

func (uc *SpiderUseCase) finishJob(jobId int64, status string, errText string) {
	if jobId <= 0 {
		return
	}
	now := time.Now()
	_ = uc.data.DB.Model(&model.SpiderRefreshJob{}).Where("id = ?", jobId).Updates(map[string]interface{}{
		"status":      status,
		"error":       errText,
		"finished_at": &now,
	}).Error
}

func (uc *SpiderUseCase) finishPlatform(jobId int64, platform string) {
	if jobId <= 0 {
		return
	}
	_ = uc.data.DB.Model(&model.SpiderRefreshJob{}).Where("id = ?", jobId).Updates(map[string]interface{}{
		"current_platform":   platform,
		"finished_platforms": gorm.Expr("finished_platforms + 1"),
	}).Error
}

func (uc *SpiderUseCase) setCurrentPlatform(jobId int64, platform string) {
	if jobId <= 0 {
		return
	}
	_ = uc.data.DB.Model(&model.SpiderRefreshJob{}).Where("id = ?", jobId).Update("current_platform", platform).Error
}

func (uc *SpiderUseCase) upsertPlatformStatus(userId int64, plat model.Platform, updates map[string]interface{}) {
	status := model.SpiderSyncStatus{
		UserID:   userId,
		Platform: plat.Platform,
		Username: plat.Username,
	}
	if err := uc.data.DB.Where("user_id = ? AND platform = ?", userId, plat.Platform).FirstOrCreate(&status).Error; err != nil {
		log.Errorf("Spider: sync status firstOrCreate failed: %v", err)
		return
	}
	if err := uc.data.DB.Model(&status).Updates(updates).Error; err != nil {
		log.Errorf("Spider: sync status update failed: %v", err)
	}
}

func (uc *SpiderUseCase) markPlatformRunning(userId int64, plat model.Platform) {
	now := time.Now()
	uc.upsertPlatformStatus(userId, plat, map[string]interface{}{
		"username":        plat.Username,
		"status":          "running",
		"last_started_at": &now,
		"last_error":      "",
	})
}

func (uc *SpiderUseCase) markPlatformSuccess(userId int64, plat model.Platform, count int) {
	now := time.Now()
	uc.upsertPlatformStatus(userId, plat, map[string]interface{}{
		"username":            plat.Username,
		"status":              "success",
		"last_finished_at":    &now,
		"last_success_at":     &now,
		"last_fetched_count":  int64(count),
		"last_error":          "",
		"consecutive_failure": int64(0),
	})
}

func (uc *SpiderUseCase) markPlatformFailed(userId int64, plat model.Platform, err error) {
	now := time.Now()
	uc.upsertPlatformStatus(userId, plat, map[string]interface{}{
		"username":            plat.Username,
		"status":              "failed",
		"last_finished_at":    &now,
		"last_error":          err.Error(),
		"consecutive_failure": gorm.Expr("spider_sync_statuses.consecutive_failure + 1"),
	})
}
func (uc *SpiderUseCase) invalidateCache(userId int64) {
	ctx := context.Background()
	rdb := uc.data.RDB

	// log.Infof("清理缓存")

	// 1. 精确 key，直接删
	_ = rdb.Del(
		ctx,
		fmt.Sprintf("core:submit_log:user:%d", userId),
		"core:submit_log:user:-1",
		fmt.Sprintf("user:%d:lastSubmitTime", userId),
		fmt.Sprintf("statistic:period:%d", userId), // 用户统计缓存
		fmt.Sprintf("statistic:period:-1"),         // 全局统计缓存
		fmt.Sprintf("statistic:platform-period:%d", userId),
		"statistic:platform-period:-1",
		// Contest log 精确 key
		fmt.Sprintf("core:contest_log:user:%d", userId),
	).Err()

	// 2. 模糊前缀，必须 SCAN
	patterns := []string{
		"core:submit_log:detail:*",
		fmt.Sprintf("statistic:heatmap:%d:*:*:*", userId),
		"statistic:heatmap:0:*:*:*",
		// Contest log 相关的模糊 key
		fmt.Sprintf("core:contest_log:user:%d:*", userId),
		"core:contest_log:detail:*",
	}

	for _, pattern := range patterns {
		iter := rdb.Scan(ctx, 0, pattern, 200).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			// 用 UNLINK，异步删除，不阻塞
			_ = rdb.Unlink(ctx, key).Err()
		}
		if err := iter.Err(); err != nil {
			log.Errorf("scan pattern %s failed: %v", pattern, err)
		}
	}
}
