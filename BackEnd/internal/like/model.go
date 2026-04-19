package like

import "time"

type VideoLike struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	VideoID   uint64 `gorm:"not null;uniqueIndex:uk_video_likes_video_user"`
	UserID    uint64 `gorm:"not null;uniqueIndex:uk_video_likes_video_user"`
	CreatedAt time.Time
}

func (VideoLike) TableName() string {
	return "video_likes"
}
