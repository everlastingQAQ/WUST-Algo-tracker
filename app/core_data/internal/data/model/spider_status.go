package model

import "time"

type SpiderSyncStatus struct {
	ID                 uint       `gorm:"comment:ID"`
	UserID             int64      `gorm:"index:idx_spider_sync_user_platform,unique;comment:用户ID"`
	Platform           string     `gorm:"index:idx_spider_sync_user_platform,unique;comment:平台"`
	Username           string     `gorm:"comment:平台用户名"`
	Status             string     `gorm:"comment:状态"`
	LastStartedAt      *time.Time `gorm:"comment:最近开始抓取时间"`
	LastFinishedAt     *time.Time `gorm:"comment:最近结束抓取时间"`
	LastSuccessAt      *time.Time `gorm:"comment:最近成功抓取时间"`
	LastError          string     `gorm:"type:text;comment:最近失败原因"`
	LastFetchedCount   int64      `gorm:"comment:最近抓取提交数"`
	LastSkippedCount   int64      `gorm:"comment:最近跳过异常/重复提交数"`
	ConsecutiveFailure int64      `gorm:"comment:连续失败次数"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SpiderRefreshJob struct {
	ID                uint       `gorm:"comment:ID"`
	UserID            int64      `gorm:"index;comment:目标用户ID"`
	RequesterID       int64      `gorm:"index;comment:发起用户ID"`
	Source            string     `gorm:"index;comment:来源 manual/cron/bind"`
	Status            string     `gorm:"index;comment:状态 queued/running/success/failed"`
	NeedAll           bool       `gorm:"comment:是否全量刷新"`
	CurrentPlatform   string     `gorm:"comment:当前平台"`
	TotalPlatforms    int32      `gorm:"comment:总平台数"`
	FinishedPlatforms int32      `gorm:"comment:已完成平台数"`
	Error             string     `gorm:"type:text;comment:失败原因"`
	StartedAt         *time.Time `gorm:"comment:开始时间"`
	FinishedAt        *time.Time `gorm:"comment:结束时间"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
