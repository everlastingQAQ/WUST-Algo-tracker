package model

import "gorm.io/gorm"

type SystemConfig struct {
	gorm.Model
	Key   string `gorm:"column:key;type:varchar(128);uniqueIndex;comment:配置键"`
	Value string `gorm:"column:value;type:text;comment:配置值"`
}
