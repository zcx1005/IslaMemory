package category

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category already exists")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// 前台：获取启用分类列表
func (s *Service) ListEnabled(ctx context.Context) ([]Category, error) {
	return s.repo.ListEnabledCategory(ctx)
}

// 后台：获取所有分类（含已删）
func (s *Service) ListAll(ctx context.Context, withDeleted bool) ([]Category, error) {
	if withDeleted {
		return s.repo.ListAllCategoryWithDeleted(ctx)
	}
	return s.repo.ListAllCategory(ctx)
}

// 获取单个分类
func (s *Service) GetByID(ctx context.Context, id uint64) (*Category, error) {
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) GetByName(ctx context.Context, name string) (*Category, error) {
	c, err := s.repo.GetCategoryByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Category, error) {
	c, err := s.repo.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return c, nil
}

// 新增分类（管理员）
func (s *Service) Create(ctx context.Context, name string, slug string, sortOrder int, status uint8) (*Category, error) {
	_, err := s.repo.GetCategoryByName(ctx, name)
	if err == nil {
		return nil, ErrCategoryExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	c := &Category{
		Name:      name,
		SortOrder: sortOrder,
		Status:    status,
		Slug:      slug,
	}

	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// 修改分类（管理员）
func (s *Service) Update(ctx context.Context, id uint64, name string, slug string, sortOrder int, status uint8) (*Category, error) {
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	// 如果改名了，要检查重名
	if c.Name != name {
		_, err := s.repo.GetCategoryByName(ctx, name)
		if err == nil {
			return nil, ErrCategoryExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	c.Name = name
	c.Slug = slug
	c.SortOrder = sortOrder
	c.Status = status

	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

// 删除分类（管理员）
// 这里是软删除
func (s *Service) Delete(ctx context.Context, id uint64) error {
	_, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}

	return s.repo.DeleteCategory(ctx, id)
}

// 恢复分类（管理员）
func (s *Service) Restore(ctx context.Context, id uint64) error {
	return s.repo.RestoreCategory(ctx, id)
}
