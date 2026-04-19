package user

import (
	"context"
	"errors"
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
