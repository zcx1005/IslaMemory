package user

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser 创建用户
func (r *Repository) CreateUser(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetUserByID 按 ID 查用户
func (r *Repository) GetUserByID(ctx context.Context, id uint64) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByAccount 按账号查用户
func (r *Repository) GetUserByAccount(ctx context.Context, account string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Where("account = ?", account).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername 按用户名查用户
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePassword 更新密码
func (r *Repository) UpdatePassword(ctx context.Context, userID uint64, newHash string, changedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"password_hash":       newHash,
			"password_changed_at": changedAt,
		}).Error
}

func (r *Repository) UpdateProfile(ctx context.Context, userID uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

type UserVideoRow struct {
	PublicID        string    `gorm:"column:public_id"`
	Title           string    `gorm:"column:title"`
	Description     string    `gorm:"column:description"`
	CoverURL        string    `gorm:"column:cover_url"`
	DurationSeconds uint32    `gorm:"column:duration_seconds"`
	Width           uint32    `gorm:"column:width"`
	Height          uint32    `gorm:"column:height"`
	PlayCount       uint64    `gorm:"column:play_count"`
	LikeCount       uint64    `gorm:"column:like_count"`
	FavoriteCount   uint64    `gorm:"column:favorite_count"`
	CommentCount    uint64    `gorm:"column:comment_count"`
	CategoryID      uint64    `gorm:"column:category_id"`
	CategoryName    string    `gorm:"column:category_name"`
	CategorySlug    string    `gorm:"column:category_slug"`
	Username        string    `gorm:"column:username"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (r *Repository) ListMyFavoriteVideos(ctx context.Context, userID uint64, page, pageSize int) ([]UserVideoRow, int64, error) {
	base := r.db.WithContext(ctx).
		Table("video_favorites vf").
		Joins("JOIN videos v ON v.id = vf.video_id").
		Joins("JOIN categories c ON c.id = v.category_id").
		Joins("JOIN users u ON u.id = v.user_id").
		Where("vf.user_id = ?", userID).
		Where("v.deleted_at IS NULL").
		Where("c.deleted_at IS NULL").
		Where("v.status = ?", 2)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]UserVideoRow, 0)
	err := base.Select(`
		v.public_id,
		v.title,
		v.description,
		v.cover_url,
		v.duration_seconds,
		v.width,
		v.height,
		v.play_count,
		v.like_count,
		v.favorite_count,
		v.comment_count,
		v.category_id,
		c.name AS category_name,
		c.slug AS category_slug,
		u.username AS username,
		v.created_at
	`).
		Order("vf.created_at DESC, vf.id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) ListMyUploadedVideos(ctx context.Context, userID uint64, page, pageSize int) ([]UserVideoRow, int64, error) {
	base := r.db.WithContext(ctx).
		Table("videos v").
		Joins("JOIN categories c ON c.id = v.category_id").
		Joins("JOIN users u ON u.id = v.user_id").
		Where("v.user_id = ?", userID).
		Where("v.deleted_at IS NULL")

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]UserVideoRow, 0)
	err := base.Select(`
		v.public_id,
		v.title,
		v.description,
		v.cover_url,
		v.duration_seconds,
		v.width,
		v.height,
		v.play_count,
		v.like_count,
		v.favorite_count,
		v.comment_count,
		v.category_id,
		c.name AS category_name,
		c.slug AS category_slug,
		u.username AS username,
		v.created_at
	`).
		Order("v.created_at DESC, v.id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
