package user

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Account           string         `gorm:"size:64;not null;uniqueIndex" json:"account"`
	Username          string         `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash      string         `gorm:"size:255;not null" json:"-"`
	AvatarURL         string         `gorm:"size:255" json:"avatar_url"`
	Role              uint8          `gorm:"default:0;not null" json:"role"`
	Status            uint8          `gorm:"default:1;not null" json:"status"`
	CanUpload         uint8          `gorm:"default:1;not null" json:"can_upload"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	PasswordChangedAt *time.Time     `json:"-"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedBy         *uint64        `gorm:"index" json:"-"`
	DeleteReason      string         `gorm:"size:255" json:"-"`
}

func (User) TableName() string {
	return "users"
}
