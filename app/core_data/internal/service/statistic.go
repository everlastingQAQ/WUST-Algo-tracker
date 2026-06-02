package service

import (
	"context"

	"cwxu-algo/api/core/v1/statistic"
	"cwxu-algo/app/core_data/internal/biz/service"
	"cwxu-algo/app/core_data/internal/data/dal"
	"cwxu-algo/app/core_data/internal/data/model"
)

// StatisticService 统计服务
type StatisticService struct {
	statistic.UnimplementedStatisticServer
	uc *service.StatisticUseCase
}

type PlatformPeriodItem struct {
	Platform string                 `json:"platform"`
	Submit   *statistic.SubmitCount `json:"submit"`
	Ac       *statistic.AcCount     `json:"ac"`
}

type CompareDailyItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type CompareSubmitLogItem struct {
	Id       uint   `json:"id"`
	Platform string `json:"platform"`
	UserId   int64  `json:"userId"`
	SubmitId string `json:"submitId"`
	Contest  string `json:"contest"`
	Problem  string `json:"problem"`
	Lang     string `json:"lang"`
	Status   string `json:"status"`
	Time     int64  `json:"time"`
}

type CompareSideItem struct {
	UserId        int64                  `json:"userId"`
	Period        *statistic.PeriodData  `json:"period"`
	Platform      []PlatformPeriodItem   `json:"platformPeriod"`
	HeatmapSubmit []CompareDailyItem     `json:"heatmapSubmit"`
	HeatmapAc     []CompareDailyItem     `json:"heatmapAc"`
	RecentSubmits []CompareSubmitLogItem `json:"recentSubmits"`
}

type CompareOverlapItem struct {
	CommonAcCount    int64 `json:"commonAcCount"`
	LeftOnlyAcCount  int64 `json:"leftOnlyAcCount"`
	RightOnlyAcCount int64 `json:"rightOnlyAcCount"`
}

type CompareResponse struct {
	Left    CompareSideItem    `json:"left"`
	Right   CompareSideItem    `json:"right"`
	Overlap CompareOverlapItem `json:"overlap"`
}

// NewStatistic 创建统计服务
func NewStatistic(uc *service.StatisticUseCase) *StatisticService {
	return &StatisticService{
		UnimplementedStatisticServer: statistic.UnimplementedStatisticServer{},
		uc:                           uc,
	}
}

// Heatmap 获取热力图数据
func (s *StatisticService) Heatmap(ctx context.Context, req *statistic.HeatmapReq) (*statistic.HeatmapResp, error) {
	return s.uc.Heatmap(ctx, req)
}

// PeriodCount 获取时间段统计数据
func (s *StatisticService) PeriodCount(ctx context.Context, req *statistic.PeriodCountReq) (*statistic.PeriodCountResp, error) {
	return s.uc.PeriodCount(ctx, req)
}

func (s *StatisticService) PlatformPeriod(ctx context.Context, userId int64) ([]PlatformPeriodItem, error) {
	data, err := s.uc.PlatformPeriodCount(ctx, userId)
	if err != nil {
		return nil, err
	}

	result := make([]PlatformPeriodItem, 0, len(data))
	for _, item := range data {
		result = append(result, PlatformPeriodItem{
			Platform: item.Platform,
			Submit: &statistic.SubmitCount{
				Today:     item.Submit.Today,
				ThisWeek:  item.Submit.ThisWeek,
				LastWeek:  item.Submit.LastWeek,
				ThisMonth: item.Submit.ThisMonth,
				LastMonth: item.Submit.LastMonth,
				ThisYear:  item.Submit.ThisYear,
				LastYear:  item.Submit.LastYear,
				Total:     item.Submit.Total,
			},
			Ac: &statistic.AcCount{
				Today:     item.Ac.Today,
				ThisWeek:  item.Ac.ThisWeek,
				LastWeek:  item.Ac.LastWeek,
				ThisMonth: item.Ac.ThisMonth,
				LastMonth: item.Ac.LastMonth,
				ThisYear:  item.Ac.ThisYear,
				LastYear:  item.Ac.LastYear,
				Total:     item.Ac.Total,
			},
		})
	}

	return result, nil
}

func (s *StatisticService) Compare(ctx context.Context, leftUserId int64, rightUserId int64, startDate string, endDate string) (*CompareResponse, error) {
	data, err := s.uc.Compare(ctx, leftUserId, rightUserId, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return &CompareResponse{
		Left:  toCompareSideItem(data.Left),
		Right: toCompareSideItem(data.Right),
		Overlap: CompareOverlapItem{
			CommonAcCount:    data.Overlap.CommonAcCount,
			LeftOnlyAcCount:  data.Overlap.LeftOnlyAcCount,
			RightOnlyAcCount: data.Overlap.RightOnlyAcCount,
		},
	}, nil
}

func toCompareSideItem(data service.CompareSideData) CompareSideItem {
	return CompareSideItem{
		UserId:        data.UserId,
		Period:        toPeriodData(data.Submit, data.Ac),
		Platform:      toPlatformPeriodItems(data.Platform),
		HeatmapSubmit: toDailyItems(data.HeatmapSubmit),
		HeatmapAc:     toDailyItems(data.HeatmapAc),
		RecentSubmits: toSubmitLogItems(data.RecentSubmits),
	}
}

func toPeriodData(submit dal.PeriodSubmitCount, ac dal.PeriodAcCount) *statistic.PeriodData {
	return &statistic.PeriodData{
		Submit: &statistic.SubmitCount{
			Today:     submit.Today,
			ThisWeek:  submit.ThisWeek,
			LastWeek:  submit.LastWeek,
			ThisMonth: submit.ThisMonth,
			LastMonth: submit.LastMonth,
			ThisYear:  submit.ThisYear,
			LastYear:  submit.LastYear,
			Total:     submit.Total,
		},
		Ac: &statistic.AcCount{
			Today:     ac.Today,
			ThisWeek:  ac.ThisWeek,
			LastWeek:  ac.LastWeek,
			ThisMonth: ac.ThisMonth,
			LastMonth: ac.LastMonth,
			ThisYear:  ac.ThisYear,
			LastYear:  ac.LastYear,
			Total:     ac.Total,
		},
	}
}

func toPlatformPeriodItems(data []dal.PlatformPeriodCount) []PlatformPeriodItem {
	result := make([]PlatformPeriodItem, 0, len(data))
	for _, item := range data {
		period := toPeriodData(item.Submit, item.Ac)
		result = append(result, PlatformPeriodItem{
			Platform: item.Platform,
			Submit:   period.Submit,
			Ac:       period.Ac,
		})
	}
	return result
}

func toDailyItems(data []dal.DailyCount) []CompareDailyItem {
	result := make([]CompareDailyItem, 0, len(data))
	for _, item := range data {
		result = append(result, CompareDailyItem{
			Date:  item.Day.Format("2006-01-02"),
			Count: item.Cnt,
		})
	}
	return result
}

func toSubmitLogItems(data []model.SubmitLog) []CompareSubmitLogItem {
	result := make([]CompareSubmitLogItem, 0, len(data))
	for _, item := range data {
		result = append(result, CompareSubmitLogItem{
			Id:       item.ID,
			Platform: item.Platform,
			UserId:   item.UserID,
			SubmitId: item.SubmitID,
			Contest:  item.Contest,
			Problem:  item.Problem,
			Lang:     item.Lang,
			Status:   item.Status,
			Time:     item.Time.Unix(),
		})
	}
	return result
}
