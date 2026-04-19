package comment

import (
	"gorm.io/gorm"
	"time"
)

type VideoComment struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	VideoID       uint64         `gorm:"not null;index" json:"video_id"`
	UserID        uint64         `gorm:"not null;index" json:"user_id"`
	ParentID      *uint64        `gorm:"index"`
	RootID        *uint64        `gorm:"index"`
	ReplyToUserID *uint64        `gorm:"index"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	LikeCount     uint64         `gorm:"not null;default:0" json:"like_count"`
	Status        uint8          `gorm:"not null;default:1" `
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	DeletedBy    *uint64 `gorm:"index"`
	DeleteReason string  `gorm:"size:255"`
}

func (VideoComment) TableName() string {
	return "video_comments"
}
