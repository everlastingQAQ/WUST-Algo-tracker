package dal

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// StatisticDal 统计数据访问层
type StatisticDal struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewStatisticDal 创建统计数据访问层
func NewStatisticDal(db *gorm.DB, rdb *redis.Client) *StatisticDal {
	return &StatisticDal{
		db:  db,
		rdb: rdb,
	}
}

// DailyCount 热力图每日统计
type DailyCount struct {
	Day time.Time
	Cnt int64
}

// HeatmapQuery 查询热力图数据
func (d *StatisticDal) HeatmapQuery(ctx context.Context, startDate, endDate string, userId int64, isAc bool) ([]DailyCount, error) {
	sub := d.db.
		Table("submit_logs").
		Select("id, time")
	if isAc {
		sub = sub.Where("status ILIKE ? OR status ILIKE ? OR status ILIKE ?", "%AC%", "%正确%", "%OK%")
	}
	if userId != 0 {
		sub = sub.Where("user_id = ?", userId)
	}

	var result []DailyCount
	err := d.db.Raw(`
		SELECT days.day, COUNT(s.id) AS cnt
		FROM (
			SELECT generate_series(
				?::date,
				?::date,
				INTERVAL '1 day'
			) AS day
		) days
		LEFT JOIN (?) s
		ON s.time >= days.day
		AND s.time < days.day + INTERVAL '1 day'
		GROUP BY days.day
		ORDER BY days.day
	`, startDate, endDate, sub).Scan(&result).Error

	return result, err
}

// PeriodSubmitCount 提交次数统计
type PeriodSubmitCount struct {
	Today     int64
	ThisWeek  int64
	LastWeek  int64
	ThisMonth int64
	LastMonth int64
	ThisYear  int64
	LastYear  int64
	Total     int64
}

// PeriodAcCount AC次数统计
type PeriodAcCount struct {
	Today     int64
	ThisWeek  int64
	LastWeek  int64
	ThisMonth int64
	LastMonth int64
	ThisYear  int64
	LastYear  int64
	Total     int64
}

type PlatformPeriodCount struct {
	Platform string
	Submit   PeriodSubmitCount
	Ac       PeriodAcCount
}

type TeamMemberPeriodCount struct {
	UserID  int64
	Submit  PeriodSubmitCount
	Ac      PeriodAcCount
	WaTotal int64
}

const acCondition = "(status ILIKE '%AC%' OR status ILIKE '%正确%' OR status ILIKE '%OK%')"
const acDistinctKey = "user_id::text || '|' || platform || '|' || COALESCE(NULLIF(BTRIM(problem), ''), submit_id)"

// GetPeriodCount 获取时间段统计数据
func (d *StatisticDal) GetPeriodCount(userId int64) (PeriodSubmitCount, PeriodAcCount, error) {
	now := statisticNow()

	// 日期范围计算
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeekStart := getWeekStart(now)
	lastWeekStart := thisWeekStart.Add(-7 * 24 * time.Hour)
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	thisYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	lastYearStart := thisYearStart.AddDate(-1, 0, 0)

	// 提交次数统计
	submit := PeriodSubmitCount{
		Today:     d.countQuery(userId, todayStart, now),
		ThisWeek:  d.countQuery(userId, thisWeekStart, now),
		LastWeek:  d.countQuery(userId, lastWeekStart, thisWeekStart),
		ThisMonth: d.countQuery(userId, thisMonthStart, now),
		LastMonth: d.countQuery(userId, lastMonthStart, thisMonthStart),
		ThisYear:  d.countQuery(userId, thisYearStart, now),
		LastYear:  d.countQuery(userId, lastYearStart, thisYearStart),
		Total:     d.countQueryTotal(userId),
	}

	// AC 次数统计（去重）
	ac := PeriodAcCount{
		Today:     d.countAcDistinctQuery(userId, todayStart, now),
		ThisWeek:  d.countAcDistinctQuery(userId, thisWeekStart, now),
		LastWeek:  d.countAcDistinctQuery(userId, lastWeekStart, thisWeekStart),
		ThisMonth: d.countAcDistinctQuery(userId, thisMonthStart, now),
		LastMonth: d.countAcDistinctQuery(userId, lastMonthStart, thisMonthStart),
		ThisYear:  d.countAcDistinctQuery(userId, thisYearStart, now),
		LastYear:  d.countAcDistinctQuery(userId, lastYearStart, thisYearStart),
		Total:     d.countAcDistinctTotal(userId),
	}

	return submit, ac, nil
}

func (d *StatisticDal) GetPlatformPeriodCount(userId int64, platforms []string) ([]PlatformPeriodCount, error) {
	now := statisticNow()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeekStart := getWeekStart(now)
	lastWeekStart := thisWeekStart.Add(-7 * 24 * time.Hour)
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	thisYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	lastYearStart := thisYearStart.AddDate(-1, 0, 0)

	type platformPeriodRow struct {
		Platform        string
		SubmitToday     int64
		SubmitThisWeek  int64
		SubmitLastWeek  int64
		SubmitThisMonth int64
		SubmitLastMonth int64
		SubmitThisYear  int64
		SubmitLastYear  int64
		SubmitTotal     int64
		AcToday         int64
		AcThisWeek      int64
		AcLastWeek      int64
		AcThisMonth     int64
		AcLastMonth     int64
		AcThisYear      int64
		AcLastYear      int64
		AcTotal         int64
	}

	query := d.db.Table("submit_logs").Where("platform IN ?", platforms)
	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}

	var rows []platformPeriodRow
	err := query.Select(`
		platform,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_today,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_week,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_week,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_month,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_month,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_year,
		COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_year,
		COUNT(*) AS submit_total,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_today,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_week,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_week,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_month,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_month,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_year,
		COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_year,
		COUNT(DISTINCT CASE WHEN `+acCondition+` THEN `+acDistinctKey+` END) AS ac_total
	`,
		todayStart, now,
		thisWeekStart, now,
		lastWeekStart, thisWeekStart,
		thisMonthStart, now,
		lastMonthStart, thisMonthStart,
		thisYearStart, now,
		lastYearStart, thisYearStart,
		todayStart, now,
		thisWeekStart, now,
		lastWeekStart, thisWeekStart,
		thisMonthStart, now,
		lastMonthStart, thisMonthStart,
		thisYearStart, now,
		lastYearStart, thisYearStart,
	).Group("platform").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	rowByPlatform := make(map[string]platformPeriodRow, len(rows))
	for _, row := range rows {
		rowByPlatform[row.Platform] = row
	}

	result := make([]PlatformPeriodCount, 0, len(platforms))
	for _, platform := range platforms {
		row := rowByPlatform[platform]
		result = append(result, PlatformPeriodCount{
			Platform: platform,
			Submit: PeriodSubmitCount{
				Today:     row.SubmitToday,
				ThisWeek:  row.SubmitThisWeek,
				LastWeek:  row.SubmitLastWeek,
				ThisMonth: row.SubmitThisMonth,
				LastMonth: row.SubmitLastMonth,
				ThisYear:  row.SubmitThisYear,
				LastYear:  row.SubmitLastYear,
				Total:     row.SubmitTotal,
			},
			Ac: PeriodAcCount{
				Today:     row.AcToday,
				ThisWeek:  row.AcThisWeek,
				LastWeek:  row.AcLastWeek,
				ThisMonth: row.AcThisMonth,
				LastMonth: row.AcLastMonth,
				ThisYear:  row.AcThisYear,
				LastYear:  row.AcLastYear,
				Total:     row.AcTotal,
			},
		})
	}
	return result, nil
}

func (d *StatisticDal) GetTeamPeriodCount(userIds []int64) ([]TeamMemberPeriodCount, error) {
	now := statisticNow()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeekStart := getWeekStart(now)
	lastWeekStart := thisWeekStart.Add(-7 * 24 * time.Hour)
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	thisYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	lastYearStart := thisYearStart.AddDate(-1, 0, 0)

	type teamPeriodRow struct {
		UserID          int64
		SubmitToday     int64
		SubmitThisWeek  int64
		SubmitLastWeek  int64
		SubmitThisMonth int64
		SubmitLastMonth int64
		SubmitThisYear  int64
		SubmitLastYear  int64
		SubmitTotal     int64
		AcToday         int64
		AcThisWeek      int64
		AcLastWeek      int64
		AcThisMonth     int64
		AcLastMonth     int64
		AcThisYear      int64
		AcLastYear      int64
		AcTotal         int64
		WaTotal         int64
	}

	var rows []teamPeriodRow
	err := d.db.Table("submit_logs").
		Where("user_id IN ?", userIds).
		Select(`
			user_id,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_today,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_week,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_week,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_month,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_month,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_this_year,
			COUNT(*) FILTER (WHERE time >= ? AND time < ?) AS submit_last_year,
			COUNT(*) AS submit_total,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_today,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_week,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_week,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_month,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_month,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_this_year,
			COUNT(DISTINCT CASE WHEN `+acCondition+` AND time >= ? AND time < ? THEN `+acDistinctKey+` END) AS ac_last_year,
			COUNT(DISTINCT CASE WHEN `+acCondition+` THEN `+acDistinctKey+` END) AS ac_total,
			COUNT(*) FILTER (WHERE status ILIKE '%WA%' OR status ILIKE '%Wrong Answer%' OR status ILIKE '%答案错误%') AS wa_total
		`,
			todayStart, now,
			thisWeekStart, now,
			lastWeekStart, thisWeekStart,
			thisMonthStart, now,
			lastMonthStart, thisMonthStart,
			thisYearStart, now,
			lastYearStart, thisYearStart,
			todayStart, now,
			thisWeekStart, now,
			lastWeekStart, thisWeekStart,
			thisMonthStart, now,
			lastMonthStart, thisMonthStart,
			thisYearStart, now,
			lastYearStart, thisYearStart,
		).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	rowByUserID := make(map[int64]teamPeriodRow, len(rows))
	for _, row := range rows {
		rowByUserID[row.UserID] = row
	}

	result := make([]TeamMemberPeriodCount, 0, len(userIds))
	for _, userId := range userIds {
		row := rowByUserID[userId]
		result = append(result, TeamMemberPeriodCount{
			UserID: userId,
			Submit: PeriodSubmitCount{
				Today:     row.SubmitToday,
				ThisWeek:  row.SubmitThisWeek,
				LastWeek:  row.SubmitLastWeek,
				ThisMonth: row.SubmitThisMonth,
				LastMonth: row.SubmitLastMonth,
				ThisYear:  row.SubmitThisYear,
				LastYear:  row.SubmitLastYear,
				Total:     row.SubmitTotal,
			},
			Ac: PeriodAcCount{
				Today:     row.AcToday,
				ThisWeek:  row.AcThisWeek,
				LastWeek:  row.AcLastWeek,
				ThisMonth: row.AcThisMonth,
				LastMonth: row.AcLastMonth,
				ThisYear:  row.AcThisYear,
				LastYear:  row.AcLastYear,
				Total:     row.AcTotal,
			},
			WaTotal: row.WaTotal,
		})
	}
	return result, nil
}

// RankItem 排行榜项
type RankItem struct {
	Rank   int64
	UserID int64
	Name   string
	Score  int64
}

// GetRank 获取排行榜数据
func (d *StatisticDal) GetRank(ctx context.Context, userId int64, timeType, scoreType string, groupId int64, page, pageSize int64) ([]RankItem, int64, error) {
	now := statisticNow()
	var startTime time.Time
	var endTime = now

	// 时间范围计算
	switch timeType {
	case "日":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "周":
		startTime = getWeekStart(now)
	case "月":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		// 默认全部时间
		startTime = time.Time{}
		endTime = statisticNow().Add(100 * 365 * 24 * time.Hour)
	}

	type RankQueryResult struct {
		Rank   int64
		UserID int64
		Name   string
		Score  int64
	}

	var results []RankQueryResult
	var total int64

	// 基础查询
	baseQuery := d.db.Table("submit_logs").
		Where("time >= ? AND time < ?", startTime, endTime)

	// 按用户ID筛选
	if userId != 0 {
		baseQuery = baseQuery.Where("user_id = ?", userId)
	}

	// 按分组筛选
	if groupId != -1 {
		baseQuery = baseQuery.Where("group_id = ?", groupId)
	}

	// 根据scoreType决定统计方式
	if scoreType == "ac" {
		// AC排行榜，按 user_id, platform, problem 去重。跨平台同名题不能合并。
		baseQuery = baseQuery.Where("status ILIKE ? OR status ILIKE ? OR status ILIKE ?", "%AC%", "%正确%", "%OK%")
	}

	// 获取总数
	countQuery := d.db.Table("(SELECT user_id FROM submit_logs WHERE time >= ? AND time < ?", startTime, endTime)
	if userId != 0 {
		countQuery = countQuery.Where("user_id = ?", userId)
	}
	if groupId != -1 {
		countQuery = countQuery.Where("group_id = ?", groupId)
	}
	if scoreType == "ac" {
		countQuery = countQuery.Where("status ILIKE ? OR status ILIKE ? OR status ILIKE ?", "%AC%", "%正确%", "%OK%")
	}
	countQuery = countQuery.Group("user_id")
	countQuery.Count(&total)

	// 分页
	offset := (page - 1) * pageSize

	// 执行查询
	var selectClause string
	if scoreType == "ac" {
		selectClause = "COUNT(DISTINCT platform || '|' || COALESCE(NULLIF(BTRIM(problem), ''), submit_id))"
	} else {
		selectClause = "COUNT(*)"
	}

	err := d.db.Table("(?)", baseQuery).
		Select("user_id, name, " + selectClause + " as score").
		Group("user_id, name").
		Order("score DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Scan(&results).Error

	if err != nil {
		return nil, 0, err
	}

	// 计算排名
	items := make([]RankItem, len(results))
	for i, r := range results {
		items[i] = RankItem{
			Rank:   int64(offset) + int64(i+1),
			UserID: r.UserID,
			Name:   r.Name,
			Score:  r.Score,
		}
	}

	return items, total, nil
}

// countQuery 统计指定时间范围内的记录数
func (d *StatisticDal) countQuery(userId int64, start, end time.Time) int64 {
	return d.countQueryByPlatform(userId, "", start, end)
}

func (d *StatisticDal) countQueryByPlatform(userId int64, platform string, start, end time.Time) int64 {
	var count int64
	query := d.db.Table("submit_logs").Where("time >= ? AND time < ?", start, end)
	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if err := query.Count(&count).Error; err != nil {
		log.Errorf("countQuery error: %v", err)
	}
	return count
}

// countQueryTotal 统计所有记录数
func (d *StatisticDal) countQueryTotal(userId int64) int64 {
	return d.countQueryTotalByPlatform(userId, "")
}

func (d *StatisticDal) countQueryTotalByPlatform(userId int64, platform string) int64 {
	var count int64
	query := d.db.Table("submit_logs")
	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if err := query.Count(&count).Error; err != nil {
		log.Errorf("countQueryTotal error: %v", err)
	}
	return count
}

// countAcDistinctQuery 统计指定时间范围内的 AC 记录数（按 user_id, platform, problem 去重）
func (d *StatisticDal) countAcDistinctQuery(userId int64, start, end time.Time) int64 {
	return d.countAcDistinctQueryByPlatform(userId, "", start, end)
}

func (d *StatisticDal) countAcDistinctQueryByPlatform(userId int64, platform string, start, end time.Time) int64 {
	var count int64
	query := d.db.Table("submit_logs").
		Where("status ILIKE ? OR status ILIKE ? OR status ILIKE ?", "%AC%", "%正确%", "%OK%").
		Where("time >= ? AND time < ?", start, end)
	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if err := query.Select("COUNT(DISTINCT " + acDistinctKey + ")").Scan(&count).Error; err != nil {
		log.Errorf("countAcDistinctQuery error: %v", err)
	}
	return count
}

// countAcDistinctTotal 统计所有 AC 记录数（按 user_id, platform, problem 去重）
func (d *StatisticDal) countAcDistinctTotal(userId int64) int64 {
	return d.countAcDistinctTotalByPlatform(userId, "")
}

func (d *StatisticDal) countAcDistinctTotalByPlatform(userId int64, platform string) int64 {
	var count int64
	query := d.db.Table("submit_logs").
		Where("status ILIKE ? OR status ILIKE ? OR status ILIKE ?", "%AC%", "%正确%", "%OK%")
	if userId != -1 {
		query = query.Where("user_id = ?", userId)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if err := query.Select("COUNT(DISTINCT " + acDistinctKey + ")").Scan(&count).Error; err != nil {
		log.Errorf("countAcDistinctTotal error: %v", err)
	}
	return count
}

// getWeekStart 获取本周周一 00:00:00
func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	days := int(weekday - time.Monday)
	start := t.AddDate(0, 0, -days)
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
}

func statisticNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return time.Now().In(loc)
}
