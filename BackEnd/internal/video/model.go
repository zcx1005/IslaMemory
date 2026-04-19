package video

import (
	"time"

	"gorm.io/gorm"
)

// Video 对应 videos 表
type Video struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	PublicID          string `gorm:"size:32;not null;uniqueIndex" json:"public_id"`
	UserID            uint64 `gorm:"not null;index" json:"user_id"`
	CategoryID        uint64 `gorm:"not null;index" json:"category_id"`
	Title             string `gorm:"size:200;not null" json:"title"`
	Description       string `gorm:"type:text" json:"description"`
	SourceVideoURL    string `gorm:"size:255;not null" json:"-"`
	PlaybackURL       string `gorm:"size:255" json:"-"`
	PlaybackType      uint8  `gorm:"not null;default:0" json:"playback_type"`
	TranscodeStatus   uint8  `gorm:"not null;default:0" json:"transcode_status"`
	TranscodeProgress uint32 `gorm:"not null;default:0" json:"transcode_progress"`
	Status            uint8  `gorm:"not null;default:0" json:"status"`
	TranscodeError    string `gorm:"size:500" json:"-"`
	CoverURL          string `gorm:"size:255" json:"cover_url"`
	DurationSeconds   uint32 `gorm:"not null;default:0" json:"duration_seconds"`
	Width             uint32 `gorm:"not null;default:0" json:"width"`
	Height            uint32 `gorm:"not null;default:0" json:"height"`
	FileSizeBytes     uint64 `gorm:"not null;default:0" json:"file_size_bytes"`

	PlayCount     uint64 `gorm:"not null;default:0" json:"play_count"`
	LikeCount     uint64 `gorm:"not null;default:0" json:"like_count"`
	FavoriteCount uint64 `gorm:"not null;default:0" json:"favorite_count"`
	CommentCount  uint64 `gorm:"not null;default:0" json:"comment_count"`

	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedBy    *uint64        `gorm:"index" json:"-"`
	DeleteReason string         `gorm:"size:255" json:"-"`
	PublishedAt  *time.Time     `json:"published_at"`
	ReviewedAt   *time.Time     `json:"reviewed_at"`
}

func (Video) TableName() string {
	return "videos"
}
