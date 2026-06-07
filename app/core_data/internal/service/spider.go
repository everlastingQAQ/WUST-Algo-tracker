package service

import (
	"context"
	"cwxu-algo/api/core/v1/spider"
	"cwxu-algo/app/common/permission"
	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/core_data/internal/data"
	"cwxu-algo/app/core_data/internal/data/model"
	"cwxu-algo/app/core_data/task"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var (
	SetForbidden    = errors.Forbidden("权限错误", "权限不允许，设置失败")
	InternalError   = errors.InternalServer("内部错误", "内部错误，操作失败")
	UpdateForbidden = errors.Forbidden("权限错误", "权限不允许，不允许手动申请全量更新他人数据")
	RateLimitError  = errors.New(429, "TOO_MANY_REQUESTS", "请求过于频繁，请稍后再试")
)

const staleAfter = 24 * time.Hour

type SpiderService struct {
	spider.UnimplementedSpiderServer
	db         *gorm.DB
	rdb        *redis.Client
	spider     *task.SpiderTask
	limiterMap sync.Map // map[int64]*rate.Limiter
}

func (s *SpiderService) getLimiter(userId int64, interval time.Duration) *rate.Limiter {
	if l, ok := s.limiterMap.Load(userId); ok {
		return l.(*rate.Limiter)
	}
	// 1 request per interval, burst 1
	l := rate.NewLimiter(rate.Every(interval), 1)
	actual, _ := s.limiterMap.LoadOrStore(userId, l)
	return actual.(*rate.Limiter)
}

func (s SpiderService) Update(ctx context.Context, req *spider.UpdateReq) (*spider.UpdateRes, error) {
	//if !auth.VerifyById(ctx, uint(req.UserId)) {
	//	return nil, UpdateForbidden
	//}
	limiter := s.getLimiter(req.UserId, 60*time.Second)
	if !limiter.Allow() {
		return nil, RateLimitError
	}
	current := auth.GetCurrentUser(ctx)
	requesterId := int64(0)
	if current != nil {
		requesterId = int64(current.UserID)
	}
	platform := strings.TrimSpace(req.GetPlatform())
	if platform == "" {
		if header, ok := transport.FromServerContext(ctx); ok {
			platform = strings.TrimSpace(header.RequestHeader().Get("X-Spider-Platform"))
		}
	}
	jobId, err := s.spider.Do(req.UserId, true, "manual", requesterId, platform) // 全量或单平台更新
	if err != nil {
		return nil, InternalError
	}
	message := "更新成功，请稍等片刻，您的全量OJ数据正在更新"
	if platform != "" {
		message = fmt.Sprintf("更新成功，请稍等片刻，%s 数据正在更新", platform)
	}
	return &spider.UpdateRes{
		Code:    0,
		Message: message,
		JobId:   jobId,
	}, nil
}

func (s SpiderService) GetSpider(ctx context.Context, req *spider.GetSpiderReq) (*spider.GetSpiderRep, error) {
	var plats []model.Platform
	err := s.db.Where("user_id = ?", req.UserId).Find(&plats).Error
	if err != nil {
		return nil, InternalError
	}
	res := make([]*spider.GetSpiderRep_Data, 0)
	for _, v := range plats {
		res = append(res, &spider.GetSpiderRep_Data{
			Platform: v.Platform,
			Username: v.Username,
		})
	}
	return &spider.GetSpiderRep{
		Data: res,
	}, nil
}

func toUnix(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

func isCoachOrAdmin(ctx context.Context) bool {
	current := auth.GetCurrentUser(ctx)
	if current == nil {
		return false
	}
	return current.RoleID == permission.RoleAdmin || current.RoleID == permission.RoleCoach
}

func canViewUserDetail(ctx context.Context, userId int64) bool {
	current := auth.GetCurrentUser(ctx)
	if current == nil {
		return false
	}
	return int64(current.UserID) == userId || current.RoleID == permission.RoleAdmin || current.RoleID == permission.RoleCoach
}

func jobToPb(job model.SpiderRefreshJob, canViewError bool) *spider.JobInfo {
	errText := job.Error
	if !canViewError {
		errText = ""
	}
	return &spider.JobInfo{
		JobId:             int64(job.ID),
		UserId:            job.UserID,
		RequesterId:       job.RequesterID,
		Source:            job.Source,
		Status:            job.Status,
		NeedAll:           job.NeedAll,
		CurrentPlatform:   job.CurrentPlatform,
		TotalPlatforms:    job.TotalPlatforms,
		FinishedPlatforms: job.FinishedPlatforms,
		Error:             errText,
		CreatedAt:         job.CreatedAt.Unix(),
		StartedAt:         toUnix(job.StartedAt),
		FinishedAt:        toUnix(job.FinishedAt),
		UpdatedAt:         job.UpdatedAt.Unix(),
	}
}

func (s SpiderService) Job(ctx context.Context, req *spider.JobReq) (*spider.JobRes, error) {
	if req.JobId <= 0 {
		return nil, errors.BadRequest("参数错误", "jobId不能为空")
	}
	var job model.SpiderRefreshJob
	if err := s.db.Where("id = ?", req.JobId).First(&job).Error; err != nil {
		return nil, errors.NotFound("任务不存在", "未找到抓取任务")
	}
	if !canViewUserDetail(ctx, job.UserID) {
		return nil, errors.Forbidden("权限错误", "无权查看该任务")
	}
	return &spider.JobRes{
		Code:    0,
		Message: "获取抓取任务成功",
		Data:    jobToPb(job, true),
	}, nil
}

func (s SpiderService) Jobs(ctx context.Context, req *spider.JobsReq) (*spider.JobsRes, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	current := auth.GetCurrentUser(ctx)
	if current == nil {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}

	query := s.db.Model(&model.SpiderRefreshJob{})
	scope := req.Scope
	if scope == "" {
		scope = "mine"
	}
	if scope == "all" {
		if !isCoachOrAdmin(ctx) {
			return nil, errors.Forbidden("权限错误", "无权查看全站抓取任务")
		}
		if req.UserId > 0 {
			query = query.Where("user_id = ?", req.UserId)
		}
	} else {
		query = query.Where("user_id = ? OR requester_id = ?", current.UserID, current.UserID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, InternalError
	}
	var jobs []model.SpiderRefreshJob
	if err := query.Order("updated_at DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&jobs).Error; err != nil {
		return nil, InternalError
	}
	list := make([]*spider.JobInfo, 0, len(jobs))
	for _, job := range jobs {
		list = append(list, jobToPb(job, canViewUserDetail(ctx, job.UserID)))
	}
	return &spider.JobsRes{
		Code:    0,
		Message: "获取抓取任务列表成功",
		Data:    list,
		Total:   total,
	}, nil
}

func (s SpiderService) Status(ctx context.Context, req *spider.StatusReq) (*spider.StatusRes, error) {
	if req.UserId <= 0 {
		return nil, errors.BadRequest("参数错误", "userId不能为空")
	}
	if !auth.VerifySelfOrAbove(ctx, uint(req.UserId)) && !isCoachOrAdmin(ctx) {
		return nil, errors.Forbidden("权限错误", "无权查看该用户抓取状态")
	}

	var platforms []model.Platform
	if err := s.db.Where("user_id = ?", req.UserId).Find(&platforms).Error; err != nil {
		return nil, InternalError
	}
	var statuses []model.SpiderSyncStatus
	if err := s.db.Where("user_id = ?", req.UserId).Find(&statuses).Error; err != nil {
		return nil, InternalError
	}
	statusByPlatform := make(map[string]model.SpiderSyncStatus, len(statuses))
	for _, item := range statuses {
		statusByPlatform[item.Platform] = item
	}

	now := time.Now()
	canViewError := canViewUserDetail(ctx, req.UserId)
	result := make([]*spider.SyncStatusInfo, 0, len(platforms))
	for _, plat := range platforms {
		item, ok := statusByPlatform[plat.Platform]
		statusText := "never"
		lastStartedAt := int64(0)
		lastFinishedAt := int64(0)
		lastSuccessAt := int64(0)
		lastFetchedCount := int64(0)
		updatedAt := int64(0)
		lastError := ""
		isStale := true
		if ok {
			statusText = item.Status
			lastStartedAt = toUnix(item.LastStartedAt)
			lastFinishedAt = toUnix(item.LastFinishedAt)
			lastSuccessAt = toUnix(item.LastSuccessAt)
			lastFetchedCount = item.LastFetchedCount
			updatedAt = item.UpdatedAt.Unix()
			if canViewError {
				lastError = item.LastError
			}
			isStale = item.LastSuccessAt == nil || now.Sub(*item.LastSuccessAt) > staleAfter
		}
		result = append(result, &spider.SyncStatusInfo{
			Platform:         plat.Platform,
			Username:         plat.Username,
			Status:           statusText,
			LastStartedAt:    lastStartedAt,
			LastFinishedAt:   lastFinishedAt,
			LastSuccessAt:    lastSuccessAt,
			LastError:        lastError,
			LastFetchedCount: lastFetchedCount,
			IsStale:          isStale,
			CanViewError:     canViewError,
			UpdatedAt:        updatedAt,
		})
	}

	return &spider.StatusRes{
		Code:              0,
		Message:           "获取抓取状态成功",
		Data:              result,
		StaleAfterSeconds: int64(staleAfter.Seconds()),
		GeneratedAt:       now.Unix(),
	}, nil
}

func (s SpiderService) SetSpider(ctx context.Context, req *spider.SetSpiderReq) (*spider.SetSpiderRep, error) {
	// 校验JWT：只能设置自己的 spider，或者管理员可以设置任何人
	if !auth.VerifySelfOrAbove(ctx, uint(req.UserId)) {
		return nil, SetForbidden
	}
	// Rate limit
	limiter := s.getLimiter(req.UserId, 30*time.Second)
	if !limiter.Allow() {
		return nil, RateLimitError
	}
	// 直接设置进去 构建Platform
	platform := model.Platform{
		UserID:   req.UserId,
		Platform: req.Platform,
		Username: req.Username,
	}
	if err := s.db.Where("user_id = ? AND platform = ?", req.UserId, req.Platform).Delete(&model.Platform{}).Error; err != nil {
		log.Errorf("SetSpider: delete platform failed: %v", err)
	}
	if err := s.db.Where("user_id = ? AND platform = ?", req.UserId, req.Platform).Delete(&model.SubmitLog{}).Error; err != nil {
		log.Errorf("SetSpider: delete submit_log failed: %v", err)
	}
	if err := s.rdb.Del(ctx, fmt.Sprintf("core:submit_log:user:%d", req.UserId), "core:submit_log:user:-1").Err(); err != nil {
		log.Errorf("SetSpider: redis del failed: %v", err)
	}
	err := s.db.Save(&platform).Error
	if err != nil {
		log.Errorf("SetSpider: save platform failed: %v", err)
		return nil, InternalError
	}
	if _, err := s.spider.Do(req.UserId, true, "bind", int64(auth.GetCurrentUserId(ctx)), req.Platform); err != nil {
		log.Errorf("SetSpider: enqueue spider task failed: %v", err)
	}
	return &spider.SetSpiderRep{
		Code:    0,
		Message: "设置成功，请稍等片刻，您的全量OJ数据正在更新",
	}, nil
}

func NewSpiderService(data *data.Data, spider *task.SpiderTask) *SpiderService {
	return &SpiderService{
		db:     data.DB,
		rdb:    data.RDB,
		spider: spider,
	}
}
