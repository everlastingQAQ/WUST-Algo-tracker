package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"cwxu-algo/app/common/permission"

	"github.com/go-kratos/kratos/v2/transport"
)

// JwtPayload JWT载荷结构体
type JwtPayload struct {
	UserID   uint   `json:"userId"`    // 用户ID
	Username string `json:"username"`  // 用户名
	Name     string `json:"name"`      // 姓名
	Email    string `json:"email"`     // 邮箱
	RoleID   int    `json:"roleId"`    // 角色ID：0=普通用户 1=管理员 2=教练
}

func praseJwtToken(ctx context.Context) string {
	header, _ := transport.FromServerContext(ctx)
	auths := strings.SplitN(header.RequestHeader().Get("Authorization"), " ", 2)
	if len(auths) < 2 {
		return ""
	}
	return auths[1]
}

func parsePayload(ctx context.Context) *JwtPayload {
	parts := strings.Split(praseJwtToken(ctx), ".")
	if len(parts) != 3 {
		return nil
	}
	payloadBase64 := parts[1]
	dstLen := base64.RawURLEncoding.DecodedLen(len(payloadBase64))
	dst := make([]byte, dstLen)
	_, err := base64.RawURLEncoding.Decode(dst, []byte(payloadBase64))
	if err != nil {
		return nil
	}
	pd := JwtPayload{}
	if err := json.Unmarshal(dst, &pd); err != nil {
		return nil
	}
	return &pd
}

// VerifyMinRole 校验调用者角色等级是否 >= minRole
// 例如：VerifyMinRole(ctx, permission.RoleCoach) 表示至少是教练
func VerifyMinRole(ctx context.Context, minRole int) bool {
	pd := parsePayload(ctx)
	if pd == nil {
		return false
	}
	return pd.RoleID >= minRole
}

// VerifySelfOrAbove 校验调用者是否能操作目标用户
// - 调用者是管理员（RoleID=1）：可以操作任何人
// - 调用者是教练（RoleID=2）：只能操作普通用户（RoleID=0）或自己
// - 普通用户（RoleID=0）：只能操作自己
func VerifySelfOrAbove(ctx context.Context, targetUserId uint) bool {
	pd := parsePayload(ctx)
	if pd == nil {
		return false
	}
	// 管理员可以操作任何人
	if pd.RoleID == permission.RoleAdmin {
		return true
	}
	// 教练只能操作普通用户或自己
	if pd.RoleID == permission.RoleCoach {
		return targetUserId == pd.UserID
	}
	// 普通用户只能操作自己
	return pd.UserID == targetUserId
}

// GetCurrentUser 获取当前登录用户信息
func GetCurrentUser(ctx context.Context) *JwtPayload {
	return parsePayload(ctx)
}

// VerifyAdmin 校验是否为管理员（RoleID=1）
func VerifyAdmin(ctx context.Context) bool {
	return VerifyMinRole(ctx, permission.RoleAdmin)
}

// VerifyCoach 校验是否为教练（RoleID=2）
func VerifyCoach(ctx context.Context) bool {
	return VerifyMinRole(ctx, permission.RoleCoach)
}
