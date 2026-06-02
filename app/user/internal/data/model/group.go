package model

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Name     *string `gorm:"column:name;type:varchar(255);comment:组名称"`
	Describe string  `gorm:"comment:组描述"`
	Avatar   string  `gorm:"comment:团队头像"`
	OwnerId  int64   `gorm:"comment:团队创建者用户ID;default:0"`
	Users    []User  `gorm:"foreignKey:GroupId;references:ID"`
}
