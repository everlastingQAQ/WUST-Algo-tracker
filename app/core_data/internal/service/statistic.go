package service

import (
	"context"

	"cwxu-algo/api/core/v1/statistic"
	"cwxu-algo/app/core_data/internal/biz/service"
	"cwxu-algo/app/core_data/internal/data"
	coreDal "cwxu-algo/app/core_data/internal/data/dal"
)

// StatisticService 统计服务
type StatisticService struct {
	statistic.UnimplementedStatisticServer
	uc   *service.StatisticUseCase
	data *data.Data
}

type PlatformPeriodItem struct {
	Platform string                 `json:"platform"`
	Submit   *statistic.SubmitCount `json:"submit"`
	Ac       *statistic.AcCount     `json:"ac"`
}

type TeamPeriodItem struct {
	UserID  int64                  `json:"userId"`
	Submit  *statistic.SubmitCount `json:"submit"`
	Ac      *statistic.AcCount     `json:"ac"`
	WaTotal int64                  `json:"waTotal"`
}

type TeamPeriodSummary struct {
	UserID  int64                  `json:"userId"`
	Submit  *statistic.SubmitCount `json:"submit"`
	Ac      *statistic.AcCount     `json:"ac"`
	WaTotal int64                  `json:"waTotal"`
}

type TeamPeriodResponse struct {
	Code    int64             `json:"code"`
	Message string            `json:"message"`
	Members []TeamPeriodItem  `json:"members"`
	Total   TeamPeriodSummary `json:"total"`
}

// NewStatistic 创建统计服务
func NewStatistic(uc *service.StatisticUseCase, data *data.Data) *StatisticService {
	return &StatisticService{
		UnimplementedStatisticServer: statistic.UnimplementedStatisticServer{},
		uc:                           uc,
		data:                         data,
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

func (s *StatisticService) TeamPeriod(ctx context.Context, userIds []int64) (*TeamPeriodResponse, error) {
	data, err := s.uc.TeamPeriodCount(ctx, userIds)
	if err != nil {
		return nil, err
	}

	result := make([]TeamPeriodItem, 0, len(data))
	totalSubmit := &statistic.SubmitCount{}
	totalAc := &statistic.AcCount{}
	var totalWa int64

	for _, item := range data {
		submit := toSubmitCount(item.Submit)
		ac := toAcCount(item.Ac)
		result = append(result, TeamPeriodItem{
			UserID:  item.UserID,
			Submit:  submit,
			Ac:      ac,
			WaTotal: item.WaTotal,
		})
		addSubmitCount(totalSubmit, submit)
		addAcCount(totalAc, ac)
		totalWa += item.WaTotal
	}

	return &TeamPeriodResponse{
		Code:    0,
		Message: "获取团队统计成功",
		Members: result,
		Total: TeamPeriodSummary{
			UserID:  0,
			Submit:  totalSubmit,
			Ac:      totalAc,
			WaTotal: totalWa,
		},
	}, nil
}

func toSubmitCount(item coreDal.PeriodSubmitCount) *statistic.SubmitCount {
	return &statistic.SubmitCount{
		Today:     item.Today,
		ThisWeek:  item.ThisWeek,
		LastWeek:  item.LastWeek,
		ThisMonth: item.ThisMonth,
		LastMonth: item.LastMonth,
		ThisYear:  item.ThisYear,
		LastYear:  item.LastYear,
		Total:     item.Total,
	}
}

func toAcCount(item coreDal.PeriodAcCount) *statistic.AcCount {
	return &statistic.AcCount{
		Today:     item.Today,
		ThisWeek:  item.ThisWeek,
		LastWeek:  item.LastWeek,
		ThisMonth: item.ThisMonth,
		LastMonth: item.LastMonth,
		ThisYear:  item.ThisYear,
		LastYear:  item.LastYear,
		Total:     item.Total,
	}
}

func addSubmitCount(total *statistic.SubmitCount, item *statistic.SubmitCount) {
	total.Today += item.Today
	total.ThisWeek += item.ThisWeek
	total.LastWeek += item.LastWeek
	total.ThisMonth += item.ThisMonth
	total.LastMonth += item.LastMonth
	total.ThisYear += item.ThisYear
	total.LastYear += item.LastYear
	total.Total += item.Total
}

func addAcCount(total *statistic.AcCount, item *statistic.AcCount) {
	total.Today += item.Today
	total.ThisWeek += item.ThisWeek
	total.LastWeek += item.LastWeek
	total.ThisMonth += item.ThisMonth
	total.LastMonth += item.LastMonth
	total.ThisYear += item.ThisYear
	total.LastYear += item.LastYear
	total.Total += item.Total
}
