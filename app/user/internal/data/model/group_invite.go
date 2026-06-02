package model

import "gorm.io/gorm"

type GroupInvite struct {
	gorm.Model
	GroupId   int64  `gorm:"comment:团队ID;index"`
	InviterId int64  `gorm:"comment:邀请人用户ID;index"`
	InviteeId int64  `gorm:"comment:被邀请用户ID;index"`
	Status    string `gorm:"comment:邀请状态;type:varchar(32);index"`

	Group   Group `gorm:"foreignKey:GroupId;references:ID"`
	Inviter User  `gorm:"foreignKey:InviterId;references:ID"`
	Invitee User  `gorm:"foreignKey:InviteeId;references:ID"`
}
