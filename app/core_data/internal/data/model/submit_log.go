package model

import "time"

type SubmitLog struct {
	ID       uint      `gorm:"comment:ID"`
	Platform string    `gorm:"comment:平台;uniqueIndex:idx_submit_logs_platform_submit_id"`
	UserID   int64     `gorm:"comment:用户ID;index"`
	SubmitID string    `gorm:"comment:提交ID;uniqueIndex:idx_submit_logs_platform_submit_id"`
	Contest  string    `gorm:"comment:比赛名称"`
	Problem  string    `gorm:"comment:问题"`
	Lang     string    `gorm:"comment:语言"`
	Status   string    `gorm:"comment:状态"`
	Time     time.Time `gorm:"comment:提交时间;index"`
}
