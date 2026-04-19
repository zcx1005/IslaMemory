package favorite

import "time"

type VideoFavorite struct {
	ID        uint64 `gorm:"primary_key;autoIncrement"`
	VideoID   uint64 `gorm:"not null;uniqueIndex:uk_video_favorites_video_user"`
	UserID    uint64 `gorm:"not null;uniqueIndex:uk_video_favorites_video_user"`
	CreatedAt time.Time
}

func (VideoFavorite) TableName() string {
	return "video_favorites"
}
