package model

import "time"

// Bulletin 公告
type Bulletin struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	Title      string    `gorm:"not null;comment:公告标题"`
	Content    string    `gorm:"not null;type:text;comment:公告内容"`
	AuthorID   int64     `gorm:"not null;index;comment:发布者ID"`
	AuthorName string    `gorm:"not null;comment:发布者姓名"`
	IsPinned   bool      `gorm:"default:false;comment:是否置顶"`
	CreatedAt  time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}
