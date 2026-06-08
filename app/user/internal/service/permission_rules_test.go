package service

import (
	"testing"

	"cwxu-algo/app/common/permission"
)

func TestCanSetUserRole(t *testing.T) {
	tests := []struct {
		name       string
		callerID   int64
		callerRole int
		targetID   int64
		targetRole int
		newRole    int
		want       bool
	}{
		{name: "admin can set normal user to coach", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleUser, newRole: permission.RoleCoach, want: true},
		{name: "normal user cannot set role", callerID: 2, callerRole: permission.RoleUser, targetID: 3, targetRole: permission.RoleUser, newRole: permission.RoleCoach, want: false},
		{name: "coach cannot set role", callerID: 2, callerRole: permission.RoleCoach, targetID: 3, targetRole: permission.RoleUser, newRole: permission.RoleUser, want: false},
		{name: "admin cannot edit self", callerID: 1, callerRole: permission.RoleAdmin, targetID: 1, targetRole: permission.RoleAdmin, newRole: permission.RoleCoach, want: false},
		{name: "admin cannot edit another admin", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleAdmin, newRole: permission.RoleCoach, want: false},
		{name: "admin cannot grant admin", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleUser, newRole: permission.RoleAdmin, want: false},
		{name: "invalid role rejected", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleUser, newRole: 99, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canSetUserRole(tt.callerID, tt.callerRole, tt.targetID, tt.targetRole, tt.newRole); got != tt.want {
				t.Fatalf("canSetUserRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanDeleteUser(t *testing.T) {
	tests := []struct {
		name       string
		callerID   int64
		callerRole int
		targetID   int64
		targetRole int
		want       bool
	}{
		{name: "admin can delete normal user", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleUser, want: true},
		{name: "normal user cannot delete", callerID: 2, callerRole: permission.RoleUser, targetID: 3, targetRole: permission.RoleUser, want: false},
		{name: "coach cannot delete", callerID: 2, callerRole: permission.RoleCoach, targetID: 3, targetRole: permission.RoleUser, want: false},
		{name: "admin cannot delete self", callerID: 1, callerRole: permission.RoleAdmin, targetID: 1, targetRole: permission.RoleAdmin, want: false},
		{name: "admin cannot delete another admin", callerID: 1, callerRole: permission.RoleAdmin, targetID: 2, targetRole: permission.RoleAdmin, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canDeleteUser(tt.callerID, tt.callerRole, tt.targetID, tt.targetRole); got != tt.want {
				t.Fatalf("canDeleteUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanBroadcastMessage(t *testing.T) {
	if !canBroadcastMessage(permission.RoleAdmin) {
		t.Fatal("admin should be able to broadcast")
	}
	if !canBroadcastMessage(permission.RoleCoach) {
		t.Fatal("coach should be able to broadcast")
	}
	if canBroadcastMessage(permission.RoleUser) {
		t.Fatal("normal user should not be able to broadcast")
	}
}
