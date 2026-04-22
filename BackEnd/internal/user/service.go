package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"IslaMemory/BackEnd/internal/auth"

	"gorm.io/gorm"
)

var (
	ErrAccountExists      = errors.New("account already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	repo        *Repository
	passwordSvc *auth.PasswordService
	jwtSvc      *auth.JWTService
}

func NewService(
	repo *Repository,
	passwordSvc *auth.PasswordService,
	jwtSvc *auth.JWTService,
) *Service {
	return &Service{
		repo:        repo,
		passwordSvc: passwordSvc,
		jwtSvc:      jwtSvc,
	}
}

// Register 注册
func (s *Service) Register(ctx context.Context, account, username, password string) (*User, error) {
	_, err := s.repo.GetUserByAccount(ctx, account)
	if err == nil {
		return nil, ErrAccountExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	_, err = s.repo.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, ErrUsernameExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := s.passwordSvc.Hash(password)
	if err != nil {
		return nil, err
	}

	u := &User{
		Account:      account,
		Username:     username,
		PasswordHash: hash,
		Status:       1,
		CanUpload:    1,
		Role:         0,
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

// 登录
func (s *Service) Login(ctx context.Context, account, password string) (string, *User, error) {
	u, err := s.repo.GetUserByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if u.Status == 0 {
		return "", nil, ErrUserDisabled
	}

	if err := s.passwordSvc.Verify(u.PasswordHash, password); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.jwtSvc.GenerateToken(u.ID, u.Username, u.Role)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

// 获取当前用户资料
func (s *Service) GetProfile(ctx context.Context, userID uint64) (*User, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uint64, username, avatarURL *string) (*User, error) {
	updates := make(map[string]any)

	if username != nil {
		trimmed := strings.TrimSpace(*username)
		if trimmed != "" {
			u, err := s.repo.GetUserByUsername(ctx, trimmed)
			if err == nil && u.ID != userID {
				return nil, ErrUsernameExists
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			updates["username"] = trimmed
		}
	}

	if avatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*avatarURL)
	}

	if err := s.repo.UpdateProfile(ctx, userID, updates); err != nil {
		return nil, err
	}

	return s.GetProfile(ctx, userID)
}

// 修改密码
func (s *Service) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := s.passwordSvc.Verify(u.PasswordHash, oldPassword); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := s.passwordSvc.Hash(newPassword)
	if err != nil {
		return err
	}

	now := time.Now()
	return s.repo.UpdatePassword(ctx, userID, newHash, now)
}

type ProfileVideoItem struct {
	PublicID        string `json:"public_id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	CoverURL        string `json:"cover_url"`
	DurationSeconds uint32 `json:"duration_seconds"`
	Width           uint32 `json:"width"`
	Height          uint32 `json:"height"`
	PlayCount       uint64 `json:"play_count"`
	LikeCount       uint64 `json:"like_count"`
	FavoriteCount   uint64 `json:"favorite_count"`
	CommentCount    uint64 `json:"comment_count"`
	CategoryID      uint64 `json:"category_id"`
	CategoryName    string `json:"category_name"`
	CategorySlug    string `json:"category_slug"`
	Username        string `json:"username"`
	CreatedAt       string `json:"created_at"`
}

type ProfileVideoList struct {
	List  []ProfileVideoItem `json:"list"`
	Total int64              `json:"total"`
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toProfileVideoList(rows []UserVideoRow, total int64) *ProfileVideoList {
	list := make([]ProfileVideoItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, ProfileVideoItem{
			PublicID:        row.PublicID,
			Title:           row.Title,
			Description:     row.Description,
			CoverURL:        row.CoverURL,
			DurationSeconds: row.DurationSeconds,
			Width:           row.Width,
			Height:          row.Height,
			PlayCount:       row.PlayCount,
			LikeCount:       row.LikeCount,
			FavoriteCount:   row.FavoriteCount,
			CommentCount:    row.CommentCount,
			CategoryID:      row.CategoryID,
			CategoryName:    row.CategoryName,
			CategorySlug:    row.CategorySlug,
			Username:        row.Username,
			CreatedAt:       row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &ProfileVideoList{List: list, Total: total}
}

func (s *Service) ListMyFavoriteVideos(ctx context.Context, userID uint64, page, pageSize int) (*ProfileVideoList, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, total, err := s.repo.ListMyFavoriteVideos(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return toProfileVideoList(rows, total), nil
}

func (s *Service) ListMyUploadedVideos(ctx context.Context, userID uint64, page, pageSize int) (*ProfileVideoList, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, total, err := s.repo.ListMyUploadedVideos(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return toProfileVideoList(rows, total), nil
}

// 给 JWT 中间件用：判断 token 是否在改密码之后仍然有效
func (s *Service) IsTokenValidAfterPasswordChange(ctx context.Context, userID uint64, issuedAt time.Time) (bool, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrUserNotFound
		}
		return false, err
	}

	if u.PasswordChangedAt == nil {
		return true, nil
	}

	if issuedAt.Before(*u.PasswordChangedAt) {
		return false, nil
	}

	return true, nil
}
