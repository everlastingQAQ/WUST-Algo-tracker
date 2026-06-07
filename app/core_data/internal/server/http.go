package server

import (
	"context"
	"cwxu-algo/api/core/v1/bulletin"
	"cwxu-algo/api/core/v1/contest_log"
	"cwxu-algo/api/core/v1/spider"
	statistic2 "cwxu-algo/api/core/v1/statistic"
	"cwxu-algo/api/core/v1/submit_log"
	"cwxu-algo/app/common/conf"
	_const "cwxu-algo/app/common/const"
	"cwxu-algo/app/core_data/internal/service"
	"encoding/json"
	nethttp "net/http"
	"strconv"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := map[string]string{
		"/api.core.v1.submit_log.Submit/GetSubmitLog":        "",
		"/api.core.v1.contest_log.Contest/GetContestList":    "",
		"/api.core.v1.contest_log.Contest/GetContestRanking": "",
		"/api.core.v1.spider.Spider/GetSpider":               "",
		"/api.core.v1.statistic.Statistic/Heatmap":           "",
		"/api.core.v1.statistic.Statistic/PeriodCount":       "",
		"/v1/core/statistic/platform-period":                 "",
		"/api.core.v1.bulletin.Bulletin/Get":                 "",
		"/api.core.v1.bulletin.Bulletin/List":                "",
	}
	return func(ctx context.Context, operation string) bool {
		//log.Info(operation)
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, logger log.Logger, submitService *service.SubmitLogService, spiderService *service.SpiderService, statisticService *service.StatisticService, contestLogService *service.ContestLogService, bulletinService *service.BulletinService) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			selector.Server(jwt.Server(func(token *jwt2.Token) (interface{}, error) {
				return []byte(_const.JWTSecret), nil
			})).Match(NewWhiteListMatcher()).Build(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	submit_log.RegisterSubmitHTTPServer(srv, submitService)
	spider.RegisterSpiderHTTPServer(srv, spiderService)
	statistic2.RegisterStatisticHTTPServer(srv, statisticService)
	contest_log.RegisterContestHTTPServer(srv, contestLogService)
	bulletin.RegisterBulletinHTTPServer(srv, bulletinService)
	srv.HandleFunc("/v1/core/statistic/platform-period", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		userId := int64(-1)
		if raw := r.URL.Query().Get("userId"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				w.WriteHeader(nethttp.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 400, "message": "userId参数错误"})
				return
			}
			userId = parsed
		}

		data, err := statisticService.PlatformPeriod(r.Context(), userId)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(nethttp.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 500, "message": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	})
	srv.HandleFunc("/v1/core/spider/retry", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}
		var req struct {
			JobId int64 `json:"jobId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(nethttp.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 400, "message": "请求参数错误"})
			return
		}
		res, err := spiderService.Retry(r.Context(), req.JobId)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			statusCode := nethttp.StatusInternalServerError
			if se := kerrors.FromError(err); se != nil {
				statusCode = int(se.Code)
			}
			w.WriteHeader(statusCode)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": statusCode, "message": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})
	return srv
}
