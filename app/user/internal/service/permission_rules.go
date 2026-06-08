package service

import "cwxu-algo/app/common/permission"

func canSetUserRole(callerID int64, callerRole int, targetID int64, targetRole int, newRole int) bool {
	if callerRole != permission.RoleAdmin {
		return false
	}
	if callerID == targetID {
		return false
	}
	if targetRole == permission.RoleAdmin {
		return false
	}
	if newRole == permission.RoleAdmin {
		return false
	}
	return permission.IsValid(newRole)
}

func canDeleteUser(callerID int64, callerRole int, targetID int64, targetRole int) bool {
	if callerRole != permission.RoleAdmin {
		return false
	}
	if callerID == targetID {
		return false
	}
	return targetRole != permission.RoleAdmin
}

func canBroadcastMessage(callerRole int) bool {
	return callerRole == permission.RoleAdmin || callerRole == permission.RoleCoach
}
