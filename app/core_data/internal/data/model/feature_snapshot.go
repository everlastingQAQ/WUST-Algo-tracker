package model

import "time"

type FeatureSnapshot struct {
	ID          uint      `gorm:"comment:ID"`
	UserID      int64     `gorm:"index:idx_feature_snapshot_user_kind,unique;comment:用户ID"`
	Kind        string    `gorm:"index:idx_feature_snapshot_user_kind,unique;comment:快照类型"`
	SourceHash  string    `gorm:"index;comment:数据源签名"`
	Payload     string    `gorm:"type:jsonb;comment:快照JSON"`
	GeneratedAt time.Time `gorm:"index;comment:生成时间"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
