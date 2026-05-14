package permission

// 角色等级，等级越高权限越大，权限校验：调用者等级 >= 被操作对象等级
const (
	RoleUser   = 0 // 普通用户
	RoleCoach  = 2 // 教练
	RoleAdmin  = 1 // 管理员（最高权限）
)

// RoleName 角色名称映射
var RoleName = map[int]string{
	RoleUser:  "普通用户",
	RoleCoach: "教练",
	RoleAdmin: "管理员",
}

// String 获取角色名称
func String(role int) string {
	if name, ok := RoleName[role]; ok {
		return name
	}
	return "未知角色"
}

// IsValid 检查角色值是否合法
func IsValid(role int) bool {
	switch role {
	case RoleUser, RoleCoach, RoleAdmin:
		return true
	}
	return false
}

// CanManage 判断调用者是否能管理目标用户（调用者角色等级 >= 目标角色等级）
func CanManage(callerRole, targetRole int) bool {
	// 同等级只能管理自己
	return callerRole >= targetRole
}
