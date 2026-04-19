package category

import (
	"context"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ListEnabledCategory 查看所有启用且未删除的分类
func (r *Repository) ListEnabledCategory(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := r.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// ListAllCategory 后台：查询所有分类（包含停用，默认不包含已软删）
func (r *Repository) ListAllCategory(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// 后台：查看包含已删除分类
func (r *Repository) ListAllCategoryWithDeleted(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := r.db.WithContext(ctx).
		Unscoped().
		Order("sort_order ASC, id ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *Repository) GetCategoryByID(ctx context.Context, id uint64) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetCategoryByName(ctx context.Context, name string) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	var c Category
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// 创建分类
func (r *Repository) CreateCategory(ctx context.Context, c *Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// 更新分类
func (r *Repository) UpdateCategory(ctx context.Context, c *Category) error {
	return r.db.WithContext(ctx).Save(c).Error
}

// 软删除
func (r *Repository) DeleteCategory(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Category{}, id).Error
}

// 恢复软删除
func (r *Repository) RestoreCategory(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&Category{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}
