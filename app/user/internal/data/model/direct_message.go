package model

import (
	"time"

	"gorm.io/gorm"
)

type DirectMessageThread struct {
	gorm.Model
	PairKey            string    `gorm:"type:varchar(64);uniqueIndex;comment:会话唯一键"`
	UserAId            int64     `gorm:"comment:较小用户ID;index"`
	UserBId            int64     `gorm:"comment:较大用户ID;index"`
	LastMessageId      uint      `gorm:"comment:最后一条消息ID"`
	LastMessagePreview string    `gorm:"type:varchar(255);comment:最后消息预览"`
	LastSenderId       int64     `gorm:"comment:最后发送者ID"`
	LastSentAt         time.Time `gorm:"comment:最后发送时间;index"`
	UnreadA            int64     `gorm:"comment:A用户未读数;default:0"`
	UnreadB            int64     `gorm:"comment:B用户未读数;default:0"`
}

type DirectMessage struct {
	gorm.Model
	ThreadId   uint       `gorm:"comment:会话ID;index"`
	SenderId   int64      `gorm:"comment:发送者ID;index"`
	ReceiverId int64      `gorm:"comment:接收者ID;index"`
	Content    string     `gorm:"type:text;comment:消息内容"`
	IsRead     bool       `gorm:"comment:是否已读;default:false;index"`
	ReadAt     *time.Time `gorm:"comment:读取时间"`
}
