package model

import "time"

type OperationLog struct {
	ID           uint      `gorm:"comment:ID"`
	OperatorID   int64     `gorm:"index;comment:操作者用户ID"`
	OperatorRole int       `gorm:"comment:操作者角色"`
	Action       string    `gorm:"index;comment:操作动作"`
	TargetType   string    `gorm:"index;comment:目标类型"`
	TargetID     int64     `gorm:"index;comment:目标ID"`
	Detail       string    `gorm:"type:text;comment:操作详情JSON"`
	CreatedAt    time.Time `gorm:"index;comment:创建时间"`
}
