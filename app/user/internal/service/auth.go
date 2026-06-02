package service

import (
	"context"
	pb "cwxu-algo/api/user/v1/auth"
	_const "cwxu-algo/app/common/const"
	authutil "cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/user/internal/data"
	"cwxu-algo/app/user/internal/data/model"
	"errors"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/golang-jwt/jwt/v5"

	"gorm.io/gorm"
)

const (
	registerInviteCodeKey     = "register_invite_code"
	defaultRegisterInviteCode = "wustacm666"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(d *data.Data) *AuthService {
	return &AuthService{
		db: d.DB,
	}
}

type RegisterInviteCodeRequest struct {
	InviteCode string `json:"inviteCode"`
}

type RegisterInviteCodeReply struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	InviteCode string `json:"inviteCode"`
}

// Login 登录
func (s *AuthService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginRes, error) {
	res := &pb.LoginRes{}
	// 做校验
	u := &model.User{}
	r := s.db.Where("username = ? and password = ?", req.Username, req.Password).First(&u)
	if errors.Is(r.Error, gorm.ErrRecordNotFound) {
		res.Success = false
		res.Message = "用户名或密码错误"
		return res, nil
	}
	// 签发 JWT Token
	expire := time.Now().Add(8640 * time.Hour) // 过期时间8640小时
	_roleIdsJSON := []byte("[0]")
	if u.RoleID == 1 || u.RoleID == 2 {
		_roleIdsJSON = []byte("[1]")
	}
	_jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   u.ID,
		"username": u.Username,
		"name":     u.Name,
		"email":    u.Email,
		"roleId":   u.RoleID,
		"roleIds":  string(_roleIdsJSON),
		"exp":      expire.Unix(),
		"nbf":      time.Now().Unix(),
	}).SignedString([]byte(_const.JWTSecret))
	if err != nil {
		res.Success = false
		res.Message = "身份校验成功，但是jwt生成失败了." + err.Error()
		return res, nil
	}
	res.Success = true
	res.Message = "登录成功"
	res.JwtToken = _jwtToken
	return res, nil
}

// Register 注册
func (s *AuthService) Register(ctx context.Context, req *pb.RegisterReq) (res *pb.RegisterRes, err error) {
	res = &pb.RegisterRes{
		Success: true,
		Message: "注册成功",
	}
	if strings.TrimSpace(req.InviteCode) == "" {
		res.Success = false
		res.Message = "请输入邀请码"
		return
	}
	if strings.TrimSpace(req.InviteCode) != s.getRegisterInviteCode() {
		res.Success = false
		res.Message = "邀请码错误"
		return
	}
	// 是否已经用户名
	users := make([]model.User, 0)
	s.db.Where("username = ?", req.Username).Find(&users)
	if len(users) >= 1 {
		res.Success = false
		res.Message = "用户名已经存在"
		return
	}
	// 尝试去注册
	newUser := &model.User{
		Username: req.Username,
		Password: req.Password,
		Avatar:   "",
		Name:     req.Name,
		Email:    req.Email,
		GroupId:  0,
		RoleID:   0,
	}
	r := s.db.Create(&newUser)
	if r.Error != nil {
		res.Success = false
		res.Message = r.Error.Error()
	}
	return
}

func (s *AuthService) GetRegisterInviteCode(ctx context.Context) (*RegisterInviteCodeReply, error) {
	if !authutil.VerifyAdmin(ctx) {
		return nil, kerrors.Forbidden("权限不足", "只有管理员可以查看邀请码")
	}
	return &RegisterInviteCodeReply{
		Success:    true,
		Message:    "获取邀请码成功",
		InviteCode: s.getRegisterInviteCode(),
	}, nil
}

func (s *AuthService) UpdateRegisterInviteCode(ctx context.Context, req *RegisterInviteCodeRequest) (*RegisterInviteCodeReply, error) {
	if !authutil.VerifyAdmin(ctx) {
		return nil, kerrors.Forbidden("权限不足", "只有管理员可以修改邀请码")
	}
	inviteCode := strings.TrimSpace(req.InviteCode)
	if inviteCode == "" {
		return nil, kerrors.BadRequest("参数错误", "邀请码不能为空")
	}
	if err := s.saveRegisterInviteCode(inviteCode); err != nil {
		return nil, kerrors.InternalServer("保存失败", err.Error())
	}
	return &RegisterInviteCodeReply{
		Success:    true,
		Message:    "邀请码已更新",
		InviteCode: inviteCode,
	}, nil
}

func (s *AuthService) getRegisterInviteCode() string {
	config := &model.SystemConfig{}
	result := s.db.Where("key = ?", registerInviteCodeKey).First(config)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return defaultRegisterInviteCode
	}
	if result.Error != nil || strings.TrimSpace(config.Value) == "" {
		return defaultRegisterInviteCode
	}
	return strings.TrimSpace(config.Value)
}

func (s *AuthService) saveRegisterInviteCode(inviteCode string) error {
	config := &model.SystemConfig{}
	result := s.db.Where("key = ?", registerInviteCodeKey).First(config)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return s.db.Create(&model.SystemConfig{
			Key:   registerInviteCodeKey,
			Value: inviteCode,
		}).Error
	}
	if result.Error != nil {
		return result.Error
	}
	config.Value = inviteCode
	return s.db.Save(config).Error
}
