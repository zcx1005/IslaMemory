package category

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category already exists")
)

type Service struct {
	repo *Repository
	rdb  *redis.Client
}

func NewService(repo *Repository, rdb ...*redis.Client) *Service {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return &Service{repo: repo, rdb: client}
}

// 前台：获取启用分类列表
func (s *Service) ListEnabled(ctx context.Context) ([]Category, error) {
	var cached []Category
	if s.getJSONCache(ctx, categoryEnabledKey, &cached) {
		return cached, nil
	}
	categories, err := s.repo.ListEnabledCategory(ctx)
	if err != nil {
		return nil, err
	}
	s.setJSONCache(ctx, categoryEnabledKey, categories, categoryCacheTTL)
	return categories, nil
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
	var cached Category
	if s.getJSONCache(ctx, fmt.Sprintf(categoryBySlugFmt, slug), &cached) {
		return &cached, nil
	}
	c, err := s.repo.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	if c.DeletedAt.Valid || c.Status == 0 {
		return nil, ErrCategoryNotFound
	}
	s.setJSONCache(ctx, fmt.Sprintf(categoryBySlugFmt, slug), c, categoryCacheTTL)
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
	s.invalidateCategoryCache(ctx, slug)
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

	oldSlug := c.Slug
	c.Name = name
	c.Slug = slug
	c.SortOrder = sortOrder
	c.Status = status

	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		return nil, err
	}

	s.invalidateCategoryCache(ctx, oldSlug, slug)
	return c, nil
}

// 删除分类（管理员）
// 这里是软删除
func (s *Service) Delete(ctx context.Context, id uint64) error {
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}

	err = s.repo.DeleteCategory(ctx, id)
	if err == nil {
		s.invalidateCategoryCache(ctx, c.Slug)
	}
	return err

}

// 恢复分类（管理员）
func (s *Service) Restore(ctx context.Context, id uint64) error {
	err := s.repo.RestoreCategory(ctx, id)
	if err == nil {
		s.invalidateCategoryCache(ctx)
	}
	return err
}
