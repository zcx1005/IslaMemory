package category

import (
	"gorm.io/gorm"
	"time"
)

type Category struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Slug      string         `gorm:"size:100;not null;uniqueIndex" json:"slug"`
	SortOrder int            `gorm:"not null;default:0" json:"sort_order"`
	Status    uint8          `gorm:"not null;default:1" json:"status"` // 1启用，  0停用
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Category) TableName() string {
	return "categories"
}
