package service

import (
	"context"

	"cwxu-algo/api/core/v1/statistic"
	"cwxu-algo/app/core_data/internal/biz/service"
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
