package server

import (
	"context"
	"cwxu-algo/api/user/v1/auth"
	"cwxu-algo/api/user/v1/group"
	"cwxu-algo/api/user/v1/profile"
	"cwxu-algo/api/user/v1/role"
	"cwxu-algo/app/common/conf"
	_const "cwxu-algo/app/common/const"
	"cwxu-algo/app/user/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt2 "github.com/golang-jwt/jwt/v5"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := map[string]string{
		"/api.user.v1.Auth/Login":        "",
		"/api.user.v1.Auth/Register":     "",
		"/api.user.v1.Profile/GetById":   "",
		"/api.user.v1.Profile/GetByName": "",
		"/api.user.v1.Profile/GetList":   "",
		"/api.user.v1.role.Role/List":    "",
		"/api.user.group.Group/Get":      "",
		"/api.user.group.Group/List":     "",
		"/api.user.v1.Team/Detail":       "",
	}
	return func(ctx context.Context, operation string) bool {
		log.Info(operation)
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	authService *service.AuthService,
	profileService *service.ProfileService,
	groupService *service.GroupService,
	roleService *service.RoleService,
	logger log.Logger,

) *http.Server {
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
	auth.RegisterAuthHTTPServer(srv, authService)
	profile.RegisterProfileHTTPServer(srv, profileService)
	group.RegisterGroupHTTPServer(srv, groupService)
	role.RegisterRoleHTTPServer(srv, roleService)
	registerProfileExtraHTTPServer(srv, profileService)
	registerSystemHTTPServer(srv, authService)
	registerTeamHTTPServer(srv, groupService)
	return srv
}

const (
	operationTeamDetail        = "/api.user.v1.Team/Detail"
	operationTeamCreate        = "/api.user.v1.Team/Create"
	operationTeamUpdate        = "/api.user.v1.Team/Update"
	operationTeamInvite        = "/api.user.v1.Team/Invite"
	operationTeamRemoveMember  = "/api.user.v1.Team/RemoveMember"
	operationTeamInviteList    = "/api.user.v1.Team/InviteList"
	operationTeamInviteRespond = "/api.user.v1.Team/InviteRespond"
	operationSystemInviteCode  = "/api.user.v1.System/RegisterInviteCode"
	operationChangePassword    = "/api.user.v1.Profile/ChangePassword"
	operationDeleteUser        = "/api.user.v1.Profile/DeleteUser"
)

func registerSystemHTTPServer(s *http.Server, srv *service.AuthService) {
	r := s.Route("/")
	r.GET("/v1/user/system/register-invite-code", func(ctx http.Context) error {
		http.SetOperation(ctx, operationSystemInviteCode)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetRegisterInviteCode(ctx)
		})
		out, err := h(ctx, nil)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/system/register-invite-code", func(ctx http.Context) error {
		var in service.RegisterInviteCodeRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationSystemInviteCode)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateRegisterInviteCode(ctx, req.(*service.RegisterInviteCodeRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}

func registerProfileExtraHTTPServer(s *http.Server, srv *service.ProfileService) {
	r := s.Route("/")
	r.POST("/v1/user/profile/change-password", func(ctx http.Context) error {
		var in service.ChangePasswordRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationChangePassword)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ChangePassword(ctx, req.(*service.ChangePasswordRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/profile/delete", func(ctx http.Context) error {
		var in service.DeleteUserRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationDeleteUser)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteUser(ctx, req.(*service.DeleteUserRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}

func registerTeamHTTPServer(s *http.Server, srv *service.GroupService) {
	r := s.Route("/")
	r.GET("/v1/user/team/detail", func(ctx http.Context) error {
		var in service.TeamDetailRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamDetail)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.TeamDetail(ctx, req.(*service.TeamDetailRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/team/create", func(ctx http.Context) error {
		var in service.TeamCreateRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamCreate)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.CreateTeam(ctx, req.(*service.TeamCreateRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/team/update", func(ctx http.Context) error {
		var in service.TeamUpdateRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamUpdate)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateTeam(ctx, req.(*service.TeamUpdateRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/team/invite", func(ctx http.Context) error {
		var in service.TeamInviteRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamInvite)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.InviteTeamMember(ctx, req.(*service.TeamInviteRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/team/member/remove", func(ctx http.Context) error {
		var in service.TeamRemoveMemberRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamRemoveMember)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RemoveTeamMember(ctx, req.(*service.TeamRemoveMemberRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.GET("/v1/user/team/invites", func(ctx http.Context) error {
		http.SetOperation(ctx, operationTeamInviteList)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ListTeamInvites(ctx)
		})
		out, err := h(ctx, nil)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
	r.POST("/v1/user/team/invite/respond", func(ctx http.Context) error {
		var in service.TeamRespondInviteRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, operationTeamInviteRespond)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RespondTeamInvite(ctx, req.(*service.TeamRespondInviteRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out)
	})
}
