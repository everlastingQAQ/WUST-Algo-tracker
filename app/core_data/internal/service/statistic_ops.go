package service

import (
	"context"
	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/core_data/internal/data/model"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"gorm.io/gorm"
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

type OperationLogItem struct {
	ID           uint            `json:"id"`
	Service      string          `json:"service"`
	OperatorID   int64           `json:"operatorId"`
	OperatorRole int             `json:"operatorRole"`
	Action       string          `json:"action"`
	TargetType   string          `json:"targetType"`
	TargetID     int64           `json:"targetId"`
	Detail       json.RawMessage `json:"detail"`
	CreatedAt    int64           `json:"createdAt"`
}

type OperationLogResponse struct {
	Code    int64              `json:"code"`
	Message string             `json:"message"`
	Data    []OperationLogItem `json:"data"`
	Total   int64              `json:"total"`
}

type FeatureSnapshotResponse struct {
	Code        int64           `json:"code"`
	Message     string          `json:"message"`
	UserID      int64           `json:"userId"`
	Kind        string          `json:"kind"`
	SourceHash  string          `json:"sourceHash"`
	Payload     json.RawMessage `json:"payload"`
	Exists      bool            `json:"exists"`
	Stale       bool            `json:"stale"`
	GeneratedAt int64           `json:"generatedAt"`
}

type SaveFeatureSnapshotRequest struct {
	UserID     int64           `json:"userId"`
	Kind       string          `json:"kind"`
	SourceHash string          `json:"sourceHash"`
	Payload    json.RawMessage `json:"payload"`
}

type AchievementGlobalRate struct {
	Unlocked int64   `json:"unlocked"`
	Total    int64   `json:"total"`
	Rate     float64 `json:"rate"`
}

type AchievementGlobalSnapshot struct {
	Code             int64                            `json:"code"`
	Message          string                           `json:"message"`
	Rates            map[string]AchievementGlobalRate `json:"rates"`
	NightPercentiles map[int64]float64                `json:"nightPercentiles"`
	SiteContexts     map[int64]map[string]any         `json:"siteContexts"`
	GeneratedAt      int64                            `json:"generatedAt"`
}

const statisticAcSQL = "(status ILIKE '%AC%' OR status ILIKE '%正确%' OR status ILIKE '%OK%')"
const statisticProblemKeySQL = "platform || '|' || COALESCE(NULLIF(BTRIM(problem), ''), submit_id)"
const statisticStaleAfter = 24 * time.Hour
const achievementGlobalSnapshotTTL = 30 * time.Minute

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
			"平台明细页可以查看每条提交是否计入 AC、problem key 和去重后的题目列表。",
			"大型账号会分批写入提交日志，抓取任务未完成前统计可能短暂低于 OJ 主页。",
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
		notes = append(notes, "洛谷主页通过数和本站可能不同：主页可能隐藏部分题库/远程题/不可见记录，本站以抓取到的提交日志为准。")
	case "codeforces":
		notes = append(notes, "Codeforces 主页 solved 口径可能包含 problemset/gym/历史可见性差异；本站以 user.status API 返回并成功写入的记录为准。")
	case "nowcoder":
		notes = append(notes, "牛客主页不同栏目可能分别展示练习、比赛或题单数据；本站统一按提交日志题目标识去重。")
	case "qoj":
		notes = append(notes, "QOJ 部分题目来源和比赛可见性会影响主页展示；本站保留抓取到的提交记录。")
	case "atcoder":
		notes = append(notes, "AtCoder 以提交记录题目标识去重，历史 contest 归档和主页统计展示可能存在时间差。")
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
	if current == nil || !canManageCoreOps(current.RoleID) {
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
	recordCoreOperation(ctx, s.data.DB, "statistic.clear_cache", "user", userId, map[string]any{
		"deletedKeys": deleted,
	})
	return &CacheClearResponse{
		Code:        0,
		Message:     "统计缓存已清理",
		UserID:      userId,
		DeletedKeys: deleted,
	}, nil
}

func operationDetailJSON(detail string) json.RawMessage {
	if strings.TrimSpace(detail) == "" {
		return json.RawMessage("{}")
	}
	if json.Valid([]byte(detail)) {
		return json.RawMessage(detail)
	}
	encoded, _ := json.Marshal(map[string]string{"raw": detail})
	return encoded
}

func (s *StatisticService) OperationLogs(ctx context.Context, page int64, pageSize int64, action string) (*OperationLogResponse, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || !canManageCoreOps(current.RoleID) {
		return nil, errors.Forbidden("权限错误", "无权查看操作日志")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 30
	}
	query := s.data.DB.Model(&model.OperationLog{})
	if strings.TrimSpace(action) != "" {
		query = query.Where("action = ?", strings.TrimSpace(action))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.OperationLog
	if err := query.Order("created_at DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OperationLogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, OperationLogItem{
			ID:           row.ID,
			Service:      "core-data",
			OperatorID:   row.OperatorID,
			OperatorRole: row.OperatorRole,
			Action:       row.Action,
			TargetType:   row.TargetType,
			TargetID:     row.TargetID,
			Detail:       operationDetailJSON(row.Detail),
			CreatedAt:    row.CreatedAt.Unix(),
		})
	}
	return &OperationLogResponse{Code: 0, Message: "获取操作日志成功", Data: items, Total: total}, nil
}

type achievementSnapshotPayload struct {
	Achievements []struct {
		Key      string `json:"key"`
		Unlocked bool   `json:"unlocked"`
	} `json:"achievements"`
}

type achievementUserMetric struct {
	UserID         int64
	TotalSubmit    int64
	AcceptedSubmit int64
	NightAcCount   int64
	TodaySubmit    int64
	TodayAc        int64
	MaxDailyWa     int64
	Hourly         [24]float64
}

func achievementPercentile(values []float64, value float64) float64 {
	if len(values) == 0 {
		return 0
	}
	less := 0
	for _, item := range values {
		if item < value {
			less++
		}
	}
	return (float64(less) / float64(len(values))) * 100
}

func achievementMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	items := append([]float64(nil), values...)
	for i := 1; i < len(items); i++ {
		current := items[i]
		j := i - 1
		for j >= 0 && items[j] > current {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = current
	}
	middle := len(items) / 2
	if len(items)%2 == 1 {
		return items[middle]
	}
	return (items[middle-1] + items[middle]) / 2
}

func achievementDistributionDistance(left [24]float64, right [24]float64) float64 {
	var sum float64
	for i := 0; i < 24; i++ {
		sum += math.Abs(left[i] - right[i])
	}
	return sum
}

func (s *StatisticService) achievementUserMetrics() (map[int64]*achievementUserMetric, error) {
	type metricRow struct {
		UserID         int64
		TotalSubmit    int64
		AcceptedSubmit int64
		NightAcCount   int64
		TodaySubmit    int64
		TodayAc        int64
	}
	var rows []metricRow
	if err := s.data.DB.Raw(`
		SELECT
			user_id,
			COUNT(*) AS total_submit,
			SUM(CASE WHEN ` + statisticAcSQL + ` THEN 1 ELSE 0 END) AS accepted_submit,
			SUM(CASE WHEN ` + statisticAcSQL + ` AND EXTRACT(HOUR FROM time) >= 0 AND EXTRACT(HOUR FROM time) < 5 THEN 1 ELSE 0 END) AS night_ac_count,
			SUM(CASE WHEN DATE(time) = CURRENT_DATE THEN 1 ELSE 0 END) AS today_submit,
			SUM(CASE WHEN DATE(time) = CURRENT_DATE AND ` + statisticAcSQL + ` THEN 1 ELSE 0 END) AS today_ac
		FROM submit_logs
		GROUP BY user_id
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	metrics := make(map[int64]*achievementUserMetric, len(rows))
	for _, row := range rows {
		metrics[row.UserID] = &achievementUserMetric{
			UserID:         row.UserID,
			TotalSubmit:    row.TotalSubmit,
			AcceptedSubmit: row.AcceptedSubmit,
			NightAcCount:   row.NightAcCount,
			TodaySubmit:    row.TodaySubmit,
			TodayAc:        row.TodayAc,
		}
	}

	type dailyWaRow struct {
		UserID int64
		Count  int64
	}
	var dailyWaRows []dailyWaRow
	if err := s.data.DB.Raw(`
		SELECT user_id, COUNT(*) AS count
		FROM submit_logs
		WHERE status ILIKE '%WA%' OR status ILIKE '%Wrong Answer%' OR status ILIKE '%答案错误%'
		GROUP BY user_id, DATE(time)
	`).Scan(&dailyWaRows).Error; err != nil {
		return nil, err
	}
	for _, row := range dailyWaRows {
		if metric := metrics[row.UserID]; metric != nil && row.Count > metric.MaxDailyWa {
			metric.MaxDailyWa = row.Count
		}
	}

	type hourRow struct {
		UserID int64
		Hour   int
		Count  int64
	}
	var hourRows []hourRow
	if err := s.data.DB.Raw(`
		SELECT user_id, EXTRACT(HOUR FROM time)::int AS hour, COUNT(*) AS count
		FROM submit_logs
		GROUP BY user_id, EXTRACT(HOUR FROM time)::int
	`).Scan(&hourRows).Error; err != nil {
		return nil, err
	}
	hourTotals := map[int64]float64{}
	for _, row := range hourRows {
		hourTotals[row.UserID] += float64(row.Count)
	}
	for _, row := range hourRows {
		metric := metrics[row.UserID]
		total := hourTotals[row.UserID]
		if metric == nil || total <= 0 || row.Hour < 0 || row.Hour > 23 {
			continue
		}
		metric.Hourly[row.Hour] = float64(row.Count) / total
	}
	return metrics, nil
}

func (s *StatisticService) buildAchievementGlobalSnapshot() (*AchievementGlobalSnapshot, error) {
	var snapshots []model.FeatureSnapshot
	if err := s.data.DB.Where("kind = ? AND user_id > 0", "achievement").Find(&snapshots).Error; err != nil {
		return nil, err
	}
	counts := map[string]int64{}
	total := int64(0)
	for _, snapshot := range snapshots {
		var payload achievementSnapshotPayload
		if err := json.Unmarshal([]byte(snapshot.Payload), &payload); err != nil || len(payload.Achievements) == 0 {
			continue
		}
		total++
		for _, achievement := range payload.Achievements {
			if strings.TrimSpace(achievement.Key) == "" || !achievement.Unlocked {
				continue
			}
			counts[achievement.Key]++
		}
	}
	rates := make(map[string]AchievementGlobalRate, len(counts))
	for key, unlocked := range counts {
		rate := 0.0
		if total > 0 {
			rate = (float64(unlocked) / float64(total)) * 100
		}
		rates[key] = AchievementGlobalRate{Unlocked: unlocked, Total: total, Rate: rate}
	}

	metrics, err := s.achievementUserMetrics()
	if err != nil {
		return nil, err
	}
	totalSubmits := make([]float64, 0, len(metrics))
	acceptRates := make([]float64, 0, len(metrics))
	nightCounts := make([]float64, 0, len(metrics))
	todayLuckyRates := []float64{}
	averageHourly := [24]float64{}
	for _, metric := range metrics {
		totalSubmits = append(totalSubmits, float64(metric.TotalSubmit))
		acceptRate := 0.0
		if metric.TotalSubmit > 0 {
			acceptRate = (float64(metric.AcceptedSubmit) / float64(metric.TotalSubmit)) * 100
		}
		acceptRates = append(acceptRates, acceptRate)
		nightCounts = append(nightCounts, float64(metric.NightAcCount))
		if metric.TodaySubmit >= 10 {
			todayLuckyRates = append(todayLuckyRates, (float64(metric.TodayAc)/float64(metric.TodaySubmit))*100)
		}
		for i := 0; i < 24; i++ {
			averageHourly[i] += metric.Hourly[i] / math.Max(float64(len(metrics)), 1)
		}
	}
	medianSubmit := achievementMedian(totalSubmits)
	medianAcceptRate := achievementMedian(acceptRates)
	maxDailyWaLeader := int64(0)
	todaySubmitLeader := int64(0)
	for _, metric := range metrics {
		if metric.MaxDailyWa > maxDailyWaLeader {
			maxDailyWaLeader = metric.MaxDailyWa
		}
		if metric.TodaySubmit > todaySubmitLeader {
			todaySubmitLeader = metric.TodaySubmit
		}
	}

	type minuteOwnerRow struct {
		UserID int64
		Minute string
	}
	var minuteRows []minuteOwnerRow
	if err := s.data.DB.Raw(`
		SELECT user_id, TO_CHAR(DATE_TRUNC('minute', time), 'YYYY-MM-DD HH24:MI') AS minute
		FROM submit_logs
		WHERE ` + statisticAcSQL + `
	`).Scan(&minuteRows).Error; err != nil {
		return nil, err
	}
	minuteOwners := map[string]map[int64]struct{}{}
	for _, row := range minuteRows {
		if _, ok := minuteOwners[row.Minute]; !ok {
			minuteOwners[row.Minute] = map[int64]struct{}{}
		}
		minuteOwners[row.Minute][row.UserID] = struct{}{}
	}
	uniqueAcMinuteByUser := map[int64]bool{}
	for _, row := range minuteRows {
		if len(minuteOwners[row.Minute]) == 1 {
			uniqueAcMinuteByUser[row.UserID] = true
		}
	}

	offPeakScores := make([]float64, 0, len(metrics))
	offPeakByUser := map[int64]float64{}
	for _, metric := range metrics {
		score := achievementDistributionDistance(metric.Hourly, averageHourly)
		offPeakByUser[metric.UserID] = score
		offPeakScores = append(offPeakScores, score)
	}

	nightPercentiles := map[int64]float64{}
	siteContexts := map[int64]map[string]any{}
	for _, metric := range metrics {
		acceptRate := 0.0
		if metric.TotalSubmit > 0 {
			acceptRate = (float64(metric.AcceptedSubmit) / float64(metric.TotalSubmit)) * 100
		}
		todayAcRate := 0.0
		if metric.TodaySubmit > 0 {
			todayAcRate = (float64(metric.TodayAc) / float64(metric.TodaySubmit)) * 100
		}
		nightPercentile := achievementPercentile(nightCounts, float64(metric.NightAcCount))
		nightPercentiles[metric.UserID] = nightPercentile
		todayAcRatePercentile := 0.0
		if metric.TodaySubmit >= 10 {
			todayAcRatePercentile = achievementPercentile(todayLuckyRates, todayAcRate)
		}
		siteContexts[metric.UserID] = map[string]any{
			"totalSubmitPercentile": achievementPercentile(totalSubmits, float64(metric.TotalSubmit)),
			"acceptRatePercentile":  achievementPercentile(acceptRates, acceptRate),
			"medianSubmit":          medianSubmit,
			"medianAcceptRate":      medianAcceptRate,
			"uniqueAcMinute":        uniqueAcMinuteByUser[metric.UserID],
			"siteDailyWaLeader":     maxDailyWaLeader >= 20 && metric.MaxDailyWa == maxDailyWaLeader,
			"todaySubmitLeader":     todaySubmitLeader > 0 && metric.TodaySubmit == todaySubmitLeader,
			"todaySubmit":           metric.TodaySubmit,
			"todayAcRatePercentile": todayAcRatePercentile,
			"offPeakPercentile":     achievementPercentile(offPeakScores, offPeakByUser[metric.UserID]),
		}
	}
	return &AchievementGlobalSnapshot{
		Code:             0,
		Message:          "获取全站成就快照成功",
		Rates:            rates,
		NightPercentiles: nightPercentiles,
		SiteContexts:     siteContexts,
		GeneratedAt:      time.Now().Unix(),
	}, nil
}

func (s *StatisticService) GetAchievementGlobalSnapshot(ctx context.Context) (*AchievementGlobalSnapshot, error) {
	var snapshot model.FeatureSnapshot
	err := s.data.DB.Where("user_id = ? AND kind = ?", -1, "achievement_global").First(&snapshot).Error
	if err == nil && time.Since(snapshot.GeneratedAt) < achievementGlobalSnapshotTTL && json.Valid([]byte(snapshot.Payload)) {
		var payload AchievementGlobalSnapshot
		if json.Unmarshal([]byte(snapshot.Payload), &payload) == nil {
			payload.Code = 0
			payload.Message = "获取全站成就快照成功"
			return &payload, nil
		}
	}
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	payload, err := s.buildAchievementGlobalSnapshot()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	record := model.FeatureSnapshot{UserID: -1, Kind: "achievement_global"}
	if err := s.data.DB.Where("user_id = ? AND kind = ?", -1, "achievement_global").Assign(model.FeatureSnapshot{
		SourceHash:  fmt.Sprintf("%d", now.Unix()),
		Payload:     string(raw),
		GeneratedAt: now,
	}).FirstOrCreate(&record).Error; err != nil {
		return nil, err
	}
	payload.GeneratedAt = now.Unix()
	return payload, nil
}

func normalizeSnapshotKind(kind string) (string, error) {
	kind = strings.TrimSpace(kind)
	switch kind {
	case "weekly_report", "achievement":
		return kind, nil
	default:
		return "", errors.BadRequest("参数错误", "不支持的快照类型")
	}
}

func (s *StatisticService) GetFeatureSnapshot(ctx context.Context, userId int64, kind string, sourceHash string) (*FeatureSnapshotResponse, error) {
	if userId <= 0 {
		return nil, errors.BadRequest("参数错误", "userId不能为空")
	}
	normalizedKind, err := normalizeSnapshotKind(kind)
	if err != nil {
		return nil, err
	}
	current := auth.GetCurrentUser(ctx)
	if current == nil || !canOperateUserDetail(int64(current.UserID), current.RoleID, userId) {
		return nil, errors.Forbidden("权限错误", "无权查看该用户快照")
	}
	var snapshot model.FeatureSnapshot
	err = s.data.DB.Where("user_id = ? AND kind = ?", userId, normalizedKind).First(&snapshot).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return &FeatureSnapshotResponse{
			Code:    0,
			Message: "暂无快照",
			UserID:  userId,
			Kind:    normalizedKind,
			Payload: json.RawMessage("{}"),
			Exists:  false,
			Stale:   true,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	stale := sourceHash != "" && snapshot.SourceHash != sourceHash
	payload := json.RawMessage(snapshot.Payload)
	if !json.Valid(payload) {
		payload = json.RawMessage("{}")
	}
	return &FeatureSnapshotResponse{
		Code:        0,
		Message:     "获取快照成功",
		UserID:      userId,
		Kind:        normalizedKind,
		SourceHash:  snapshot.SourceHash,
		Payload:     payload,
		Exists:      true,
		Stale:       stale,
		GeneratedAt: snapshot.GeneratedAt.Unix(),
	}, nil
}

func (s *StatisticService) SaveFeatureSnapshot(ctx context.Context, req SaveFeatureSnapshotRequest) (*FeatureSnapshotResponse, error) {
	if req.UserID <= 0 {
		return nil, errors.BadRequest("参数错误", "userId不能为空")
	}
	normalizedKind, err := normalizeSnapshotKind(req.Kind)
	if err != nil {
		return nil, err
	}
	current := auth.GetCurrentUser(ctx)
	if current == nil || !canOperateUserDetail(int64(current.UserID), current.RoleID, req.UserID) {
		return nil, errors.Forbidden("权限错误", "无权保存该用户快照")
	}
	payload := strings.TrimSpace(string(req.Payload))
	if payload == "" || payload == "null" {
		payload = "{}"
	}
	if !json.Valid([]byte(payload)) {
		return nil, errors.BadRequest("参数错误", "payload必须是JSON")
	}
	now := time.Now()
	snapshot := model.FeatureSnapshot{UserID: req.UserID, Kind: normalizedKind}
	if err := s.data.DB.Where("user_id = ? AND kind = ?", req.UserID, normalizedKind).Assign(model.FeatureSnapshot{
		SourceHash:  strings.TrimSpace(req.SourceHash),
		Payload:     payload,
		GeneratedAt: now,
	}).FirstOrCreate(&snapshot).Error; err != nil {
		return nil, err
	}
	recordCoreOperation(ctx, s.data.DB, "snapshot.save", "user", req.UserID, map[string]any{
		"kind":       normalizedKind,
		"sourceHash": req.SourceHash,
	})
	return &FeatureSnapshotResponse{
		Code:        0,
		Message:     "保存快照成功",
		UserID:      req.UserID,
		Kind:        normalizedKind,
		SourceHash:  req.SourceHash,
		Payload:     json.RawMessage(payload),
		Exists:      true,
		Stale:       false,
		GeneratedAt: now.Unix(),
	}, nil
}
