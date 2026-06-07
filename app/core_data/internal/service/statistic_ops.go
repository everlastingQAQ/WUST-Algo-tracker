package service

import (
	"context"
	"cwxu-algo/app/common/permission"
	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/core_data/internal/data/model"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
)

type StatisticExplanation struct {
	Code      int64    `json:"code"`
	Message   string   `json:"message"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Bullets   []string `json:"bullets"`
	Generated int64    `json:"generatedAt"`
}

type StatisticPlatformSummary struct {
	RawSubmits        int64 `json:"rawSubmits"`
	AcceptedSubmits   int64 `json:"acceptedSubmits"`
	DistinctSubmitted int64 `json:"distinctSubmitted"`
	DistinctAc        int64 `json:"distinctAc"`
	FilteredDuplicate int64 `json:"filteredDuplicate"`
	FilteredInvalid   int64 `json:"filteredInvalid"`
}

type StatisticDetailRecord struct {
	ID           uint   `json:"id"`
	SubmitID     string `json:"submitId"`
	Platform     string `json:"platform"`
	Problem      string `json:"problem"`
	ProblemKey   string `json:"problemKey"`
	Contest      string `json:"contest"`
	Lang         string `json:"lang"`
	Status       string `json:"status"`
	Time         int64  `json:"time"`
	IncludedInAc bool   `json:"includedInAc"`
	AuditReason  string `json:"auditReason"`
}

type StatisticProblemRecord struct {
	ProblemKey      string `json:"problemKey"`
	Problem         string `json:"problem"`
	FirstAcAt       int64  `json:"firstAcAt"`
	AcceptedSubmits int64  `json:"acceptedSubmits"`
	TotalSubmits    int64  `json:"totalSubmits"`
}

type StatisticPlatformDetail struct {
	Code     int64                    `json:"code"`
	Message  string                   `json:"message"`
	UserID   int64                    `json:"userId"`
	Platform string                   `json:"platform"`
	Mode     string                   `json:"mode"`
	Page     int64                    `json:"page"`
	PageSize int64                    `json:"pageSize"`
	Total    int64                    `json:"total"`
	Summary  StatisticPlatformSummary `json:"summary"`
	Policy   []string                 `json:"policy"`
	Records  []StatisticDetailRecord  `json:"records"`
	Problems []StatisticProblemRecord `json:"problems"`
}

type SpiderAuditItem struct {
	Platform               string   `json:"platform"`
	Username               string   `json:"username"`
	Status                 string   `json:"status"`
	LastStartedAt          int64    `json:"lastStartedAt"`
	LastFinishedAt         int64    `json:"lastFinishedAt"`
	LastSuccessAt          int64    `json:"lastSuccessAt"`
	LastRawFetchedCount    int64    `json:"lastRawFetchedCount"`
	LastFetchedCount       int64    `json:"lastFetchedCount"`
	LastSkippedCount       int64    `json:"lastSkippedCount"`
	LastError              string   `json:"lastError"`
	RawSubmitCount         int64    `json:"rawSubmitCount"`
	DistinctSubmitCount    int64    `json:"distinctSubmitCount"`
	AcceptedSubmitCount    int64    `json:"acceptedSubmitCount"`
	DistinctAcCount        int64    `json:"distinctAcCount"`
	InvalidRowCount        int64    `json:"invalidRowCount"`
	FilteredDuplicateCount int64    `json:"filteredDuplicateCount"`
	FilteredAbnormalCount  int64    `json:"filteredAbnormalCount"`
	CountPolicy            []string `json:"countPolicy"`
	FilterReasons          []string `json:"filterReasons"`
	AuditNotes             []string `json:"auditNotes"`
	IsStale                bool     `json:"isStale"`
}

type SpiderAuditResponse struct {
	Code              int64             `json:"code"`
	Message           string            `json:"message"`
	UserID            int64             `json:"userId"`
	StaleAfterSeconds int64             `json:"staleAfterSeconds"`
	Data              []SpiderAuditItem `json:"data"`
}

type CacheKeyInfo struct {
	Key    string `json:"key"`
	Exists bool   `json:"exists"`
	TTL    int64  `json:"ttl"`
}

type CacheStatusResponse struct {
	Code      int64          `json:"code"`
	Message   string         `json:"message"`
	UserID    int64          `json:"userId"`
	Keys      []CacheKeyInfo `json:"keys"`
	Generated int64          `json:"generatedAt"`
}

type CacheClearResponse struct {
	Code        int64  `json:"code"`
	Message     string `json:"message"`
	UserID      int64  `json:"userId"`
	DeletedKeys int64  `json:"deletedKeys"`
}

const statisticAcSQL = "(status ILIKE '%AC%' OR status ILIKE '%正确%' OR status ILIKE '%OK%')"
const statisticProblemKeySQL = "platform || '|' || COALESCE(NULLIF(BTRIM(problem), ''), submit_id)"
const statisticStaleAfter = 24 * time.Hour

func (s *StatisticService) Explanation() StatisticExplanation {
	return StatisticExplanation{
		Code:    0,
		Message: "获取统计口径说明成功",
		Title:   "统计口径说明",
		Summary: "本站统计基于抓取到的提交日志做统一去重，和各 OJ 个人主页展示口径可能存在差异。",
		Bullets: []string{
			"AC 数按 platform + problem key 去重；problem 为空时回退 submit_id。",
			"洛谷会保留 record/list 中的全部记录，包括 U/T/SP 等题目，因此可能高于洛谷主页公开题库口径。",
			"Codeforces 基于 user.status API，题目标识包含 contestId/index/name，可能与主页 Problems solved 的内部口径略有差异。",
			"更新 OJ 数据、重爬队列和缓存刷新过程中，短时间内可能看到数字变化。",
		},
		Generated: time.Now().Unix(),
	}
}

func problemDistinctExpr() string {
	return "COALESCE(NULLIF(BTRIM(problem), ''), submit_id)"
}

func problemDistinctExprWithAlias(alias string) string {
	return fmt.Sprintf("COALESCE(NULLIF(BTRIM(%s.problem), ''), %s.submit_id)", alias, alias)
}

func statisticCountPolicy(platform string) []string {
	policy := []string{
		"原始提交：数据库中保留的该平台提交记录总数。",
		"去重提交：按 problem key 去重后的提交题目数。",
		"AC 提交：状态识别为 AC/Accepted/正确/OK 的提交次数。",
		"去重 AC：最终展示 AC 数，按 platform + problem key 去重。",
		"problem key：优先使用题目标识，缺失时回退 submit_id；跨平台同名题不合并。",
	}
	switch strings.ToLower(platform) {
	case "luogu":
		policy = append(policy, "洛谷保留抓到的全部记录，可能包含 U/T/SP 等主页统计不展示的题目。")
	case "codeforces":
		policy = append(policy, "Codeforces 基于 user.status API，题目标识由 contestId/index/name 归一化生成。")
	case "nowcoder":
		policy = append(policy, "牛客按提交记录中的题目标识去重，可能与主页不同 tab 的展示口径不同。")
	}
	return policy
}

func auditReason(row model.SubmitLog) string {
	problem := strings.TrimSpace(row.Problem)
	submitID := strings.TrimSpace(row.SubmitID)
	if submitID == "" {
		return "提交 ID 缺失，审计中视为异常记录。"
	}
	keySource := "题目标识"
	if problem == "" {
		keySource = "submit_id 回退"
	}
	if isAcceptedStatus(row.Status) {
		return fmt.Sprintf("AC 状态，进入 AC 候选；最终按 %s 去重。", keySource)
	}
	return fmt.Sprintf("非 AC 状态，仅计入原始提交和提交题，AC 统计不计入；problem key 来源：%s。", keySource)
}

func auditNotes(platform string, audit SpiderAuditItem) []string {
	notes := []string{
		fmt.Sprintf("库内原始提交 %d 条，去重提交题 %d 个。", audit.RawSubmitCount, audit.DistinctSubmitCount),
		fmt.Sprintf("AC 提交 %d 条，最终去重 AC %d 题。", audit.AcceptedSubmitCount, audit.DistinctAcCount),
	}
	if audit.LastStartedAt > 0 {
		notes = append(notes, fmt.Sprintf("最近一次抓取原始返回约 %d 条，有效写入 %d 条，跳过 %d 条。", audit.LastRawFetchedCount, audit.LastFetchedCount, audit.LastSkippedCount))
	}
	if audit.FilteredDuplicateCount > 0 {
		notes = append(notes, fmt.Sprintf("库内有 %d 条重复提交记录会在题目维度统计时被去重。", audit.FilteredDuplicateCount))
	}
	if audit.InvalidRowCount > 0 {
		notes = append(notes, fmt.Sprintf("检测到 %d 条提交 ID 或时间异常记录，需要优先排查爬虫返回。", audit.InvalidRowCount))
	}
	if audit.IsStale {
		notes = append(notes, "最近成功同步已超过 24 小时，数字可能不是最新。")
	}
	if audit.LastError != "" {
		notes = append(notes, "最近一次抓取失败原因已记录，可结合抓取任务日志排查。")
	}
	switch strings.ToLower(platform) {
	case "luogu":
		notes = append(notes, "洛谷主页通过数和本站可能不同：本站以抓取到的提交日志为准。")
	case "codeforces":
		notes = append(notes, "Codeforces 大账号可能包含海量历史提交；本站以 API 返回并成功写入的记录为准。")
	}
	return notes
}

func (s *StatisticService) PlatformDetail(ctx context.Context, userId int64, platform string, mode string, page int64, pageSize int64) (*StatisticPlatformDetail, error) {
	if userId <= 0 {
		return nil, errors.BadRequest("参数错误", "userId不能为空")
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return nil, errors.BadRequest("参数错误", "platform不能为空")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 30
	}
	if mode == "" {
		mode = "ac"
	}

	summary := StatisticPlatformSummary{}
	summarySQL := fmt.Sprintf(`
		SELECT
			COUNT(*) AS raw_submits,
			COUNT(CASE WHEN %s THEN 1 END) AS accepted_submits,
			COUNT(DISTINCT %s) AS distinct_submitted,
			COUNT(DISTINCT CASE WHEN %s THEN %s END) AS distinct_ac,
			GREATEST(COUNT(*) - COUNT(DISTINCT %s), 0) AS filtered_duplicate,
			COUNT(CASE WHEN BTRIM(submit_id) = '' OR time IS NULL THEN 1 END) AS filtered_invalid
		FROM submit_logs
		WHERE user_id = ? AND platform = ?
	`, statisticAcSQL, problemDistinctExpr(), statisticAcSQL, problemDistinctExpr(), problemDistinctExpr())
	if err := s.data.DB.Raw(summarySQL, userId, platform).Scan(&summary).Error; err != nil {
		return nil, err
	}

	resp := &StatisticPlatformDetail{
		Code:     0,
		Message:  "获取平台统计明细成功",
		UserID:   userId,
		Platform: platform,
		Mode:     mode,
		Page:     page,
		PageSize: pageSize,
		Summary:  summary,
		Policy:   statisticCountPolicy(platform),
	}

	if mode == "submit" {
		query := s.data.DB.Model(&model.SubmitLog{}).Where("user_id = ? AND platform = ?", userId, platform)
		if err := query.Count(&resp.Total).Error; err != nil {
			return nil, err
		}
		var rows []model.SubmitLog
		if err := query.Order("time DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
			return nil, err
		}
		resp.Records = make([]StatisticDetailRecord, 0, len(rows))
		for _, row := range rows {
			problemKey := strings.TrimSpace(row.Problem)
			if problemKey == "" {
				problemKey = row.SubmitID
			}
			resp.Records = append(resp.Records, StatisticDetailRecord{
				ID:           row.ID,
				SubmitID:     row.SubmitID,
				Platform:     row.Platform,
				Problem:      row.Problem,
				ProblemKey:   row.Platform + "|" + problemKey,
				Contest:      row.Contest,
				Lang:         row.Lang,
				Status:       row.Status,
				Time:         row.Time.Unix(),
				IncludedInAc: isAcceptedStatus(row.Status),
				AuditReason:  auditReason(row),
			})
		}
		return resp, nil
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (
		SELECT %s AS problem_key
		FROM submit_logs
		WHERE user_id = ? AND platform = ? AND %s
		GROUP BY problem_key
	) t`, problemDistinctExpr(), statisticAcSQL)
	if err := s.data.DB.Raw(countSQL, userId, platform).Scan(&resp.Total).Error; err != nil {
		return nil, err
	}
	type problemRow struct {
		ProblemKey      string
		Problem         string
		FirstAcAt       time.Time
		AcceptedSubmits int64
		TotalSubmits    int64
	}
	var rows []problemRow
	detailSQL := fmt.Sprintf(`
		WITH ac_problems AS (
			SELECT
				%s AS problem_key,
				MIN(problem) AS problem,
				MIN(time) AS first_ac_at,
				COUNT(*) AS accepted_submits
			FROM submit_logs
			WHERE user_id = ? AND platform = ? AND %s
			GROUP BY problem_key
			ORDER BY first_ac_at DESC
			OFFSET ? LIMIT ?
		)
		SELECT
			ac.problem_key,
			ac.problem,
			ac.first_ac_at,
			ac.accepted_submits,
			COUNT(sl.id) AS total_submits
		FROM ac_problems ac
		LEFT JOIN submit_logs sl
			ON sl.user_id = ? AND sl.platform = ? AND %s = ac.problem_key
		GROUP BY ac.problem_key, ac.problem, ac.first_ac_at, ac.accepted_submits
		ORDER BY ac.first_ac_at DESC
	`, problemDistinctExpr(), statisticAcSQL, problemDistinctExprWithAlias("sl"))
	if err := s.data.DB.Raw(detailSQL, userId, platform, (page-1)*pageSize, pageSize, userId, platform).Scan(&rows).Error; err != nil {
		return nil, err
	}
	resp.Problems = make([]StatisticProblemRecord, 0, len(rows))
	for _, row := range rows {
		resp.Problems = append(resp.Problems, StatisticProblemRecord{
			ProblemKey:      platform + "|" + row.ProblemKey,
			Problem:         row.Problem,
			FirstAcAt:       row.FirstAcAt.Unix(),
			AcceptedSubmits: row.AcceptedSubmits,
			TotalSubmits:    row.TotalSubmits,
		})
	}
	return resp, nil
}

func isAcceptedStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(status, "ac") || strings.Contains(status, "accepted") || strings.Contains(status, "正确") || strings.Contains(status, "ok")
}

func (s *StatisticService) SpiderAudit(ctx context.Context, userId int64) (*SpiderAuditResponse, error) {
	if userId <= 0 {
		return nil, errors.BadRequest("参数错误", "userId不能为空")
	}
	if !canViewUserDetail(ctx, userId) {
		return nil, errors.Forbidden("权限错误", "无权查看该用户抓取审计")
	}
	var platforms []model.Platform
	if err := s.data.DB.Where("user_id = ?", userId).Find(&platforms).Error; err != nil {
		return nil, err
	}
	var statuses []model.SpiderSyncStatus
	if err := s.data.DB.Where("user_id = ?", userId).Find(&statuses).Error; err != nil {
		return nil, err
	}
	statusByPlatform := make(map[string]model.SpiderSyncStatus, len(statuses))
	for _, item := range statuses {
		statusByPlatform[item.Platform] = item
	}
	now := time.Now()
	data := make([]SpiderAuditItem, 0, len(platforms))
	for _, platform := range platforms {
		item := statusByPlatform[platform.Platform]
		audit := SpiderAuditItem{
			Platform:            platform.Platform,
			Username:            platform.Username,
			Status:              "never",
			LastStartedAt:       toUnix(item.LastStartedAt),
			LastFinishedAt:      toUnix(item.LastFinishedAt),
			LastSuccessAt:       toUnix(item.LastSuccessAt),
			LastRawFetchedCount: item.LastFetchedCount + item.LastSkippedCount,
			LastFetchedCount:    item.LastFetchedCount,
			LastSkippedCount:    item.LastSkippedCount,
			LastError:           item.LastError,
			CountPolicy:         statisticCountPolicy(platform.Platform),
		}
		if item.Status != "" {
			audit.Status = item.Status
		}
		if item.LastSuccessAt == nil || now.Sub(*item.LastSuccessAt) > statisticStaleAfter {
			audit.IsStale = true
		}
		countSQL := fmt.Sprintf(`
			SELECT
				COUNT(*) AS raw_submit_count,
				COUNT(DISTINCT %s) AS distinct_submit_count,
				COUNT(CASE WHEN %s THEN 1 END) AS accepted_submit_count,
				COUNT(DISTINCT CASE WHEN %s THEN %s END) AS distinct_ac_count,
				COUNT(CASE WHEN BTRIM(submit_id) = '' OR time IS NULL THEN 1 END) AS invalid_row_count,
				GREATEST(COUNT(*) - COUNT(DISTINCT %s), 0) AS filtered_duplicate_count
			FROM submit_logs
			WHERE user_id = ? AND platform = ?
		`, problemDistinctExpr(), statisticAcSQL, statisticAcSQL, problemDistinctExpr(), problemDistinctExpr())
		_ = s.data.DB.Raw(countSQL, userId, platform.Platform).Scan(&audit).Error
		audit.FilteredAbnormalCount = audit.InvalidRowCount + audit.LastSkippedCount
		audit.FilterReasons = []string{
			"重复提交：同一 problem key 的多次提交只在 AC/提交题维度计为 1。",
			"异常记录：submit_id 或时间缺失的记录会标记为异常，需结合最近错误排查。",
			"非 AC 提交：保留在提交记录中，但不会进入最终 AC 数。",
		}
		audit.AuditNotes = auditNotes(platform.Platform, audit)
		data = append(data, audit)
	}
	return &SpiderAuditResponse{
		Code:              0,
		Message:           "获取抓取审计成功",
		UserID:            userId,
		StaleAfterSeconds: int64(statisticStaleAfter.Seconds()),
		Data:              data,
	}, nil
}

func statisticCacheKeys(userId int64) []string {
	return []string{
		fmt.Sprintf("statistic:period:%d", userId),
		fmt.Sprintf("statistic:platform-period:%d", userId),
		fmt.Sprintf("core:submit_log:user:%d", userId),
		fmt.Sprintf("core:contest_log:user:%d", userId),
		"statistic:period:-1",
		"statistic:platform-period:-1",
	}
}

func statisticCachePatterns(userId int64) []string {
	return []string{
		fmt.Sprintf("statistic:heatmap:%d:*", userId),
		"statistic:heatmap:0:*",
		"core:submit_log:detail:*",
		fmt.Sprintf("core:contest_log:user:%d:*", userId),
		"core:contest_log:detail:*",
	}
}

func (s *StatisticService) CacheStatus(ctx context.Context, userId int64) (*CacheStatusResponse, error) {
	if userId <= 0 {
		userId = -1
	}
	if !isCoachOrAdmin(ctx) {
		return nil, errors.Forbidden("权限错误", "无权查看缓存状态")
	}
	keys := statisticCacheKeys(userId)
	result := make([]CacheKeyInfo, 0, len(keys)+8)
	for _, key := range keys {
		ttl, err := s.data.RDB.TTL(ctx, key).Result()
		exists := err == nil && ttl != -2*time.Second
		result = append(result, CacheKeyInfo{
			Key:    key,
			Exists: exists,
			TTL:    int64(ttl.Seconds()),
		})
	}
	for _, pattern := range statisticCachePatterns(userId) {
		iter := s.data.RDB.Scan(ctx, 0, pattern, 20).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			ttl, err := s.data.RDB.TTL(ctx, key).Result()
			result = append(result, CacheKeyInfo{
				Key:    key,
				Exists: err == nil && ttl != -2*time.Second,
				TTL:    int64(ttl.Seconds()),
			})
			if len(result) >= 30 {
				break
			}
		}
	}
	return &CacheStatusResponse{
		Code:      0,
		Message:   "获取缓存状态成功",
		UserID:    userId,
		Keys:      result,
		Generated: time.Now().Unix(),
	}, nil
}

func (s *StatisticService) ClearCache(ctx context.Context, userId int64) (*CacheClearResponse, error) {
	if userId <= 0 {
		userId = -1
	}
	current := auth.GetCurrentUser(ctx)
	if current == nil || (current.RoleID != permission.RoleAdmin && current.RoleID != permission.RoleCoach) {
		return nil, errors.Forbidden("权限错误", "无权清理统计缓存")
	}
	deleted := int64(0)
	keys := statisticCacheKeys(userId)
	if len(keys) > 0 {
		n, _ := s.data.RDB.Del(ctx, keys...).Result()
		deleted += n
	}
	for _, pattern := range statisticCachePatterns(userId) {
		iter := s.data.RDB.Scan(ctx, 0, pattern, 200).Iterator()
		var batch []string
		for iter.Next(ctx) {
			batch = append(batch, iter.Val())
			if len(batch) >= 100 {
				n, _ := s.data.RDB.Unlink(ctx, batch...).Result()
				deleted += n
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			n, _ := s.data.RDB.Unlink(ctx, batch...).Result()
			deleted += n
		}
	}
	return &CacheClearResponse{
		Code:        0,
		Message:     "统计缓存已清理",
		UserID:      userId,
		DeletedKeys: deleted,
	}, nil
}
