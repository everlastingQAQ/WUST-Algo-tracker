package service

import (
	"testing"

	"cwxu-algo/app/common/permission"
)

func TestCanManageCoreOps(t *testing.T) {
	tests := []struct {
		name string
		role int
		want bool
	}{
		{name: "admin can manage", role: permission.RoleAdmin, want: true},
		{name: "coach can manage", role: permission.RoleCoach, want: true},
		{name: "normal user cannot manage", role: permission.RoleUser, want: false},
		{name: "unknown role cannot manage", role: 99, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canManageCoreOps(tt.role); got != tt.want {
				t.Fatalf("canManageCoreOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanOperateUserDetail(t *testing.T) {
	tests := []struct {
		name          string
		currentUserID int64
		role          int
		targetUserID  int64
		want          bool
	}{
		{name: "self can view own detail", currentUserID: 4, role: permission.RoleUser, targetUserID: 4, want: true},
		{name: "normal user cannot view others detail", currentUserID: 4, role: permission.RoleUser, targetUserID: 5, want: false},
		{name: "admin can view others detail", currentUserID: 1, role: permission.RoleAdmin, targetUserID: 5, want: true},
		{name: "coach can view others detail", currentUserID: 2, role: permission.RoleCoach, targetUserID: 5, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canOperateUserDetail(tt.currentUserID, tt.role, tt.targetUserID); got != tt.want {
				t.Fatalf("canOperateUserDetail() = %v, want %v", got, tt.want)
			}
		})
	}
}
