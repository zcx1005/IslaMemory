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
