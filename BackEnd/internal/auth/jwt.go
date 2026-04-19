package auth

import (
	"IslaMemory/BackEnd/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Role     uint8  `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	key         []byte
	issuer      string
	subject     string
	expireHours int
}

func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		key:         []byte(cfg.Key),
		issuer:      cfg.Issuer,
		subject:     cfg.Subject,
		expireHours: cfg.ExpireHours,
	}
}

// GenerateToken 生成token
func (s *JWTService) GenerateToken(userID uint64, username string, role uint8) (string, error) {
	now := time.Now()
	expireAt := now.Add(time.Duration(s.expireHours) * time.Hour)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   s.subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.key)
}

// ParseToken 解析token
func (s *JWTService) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return s.key, nil
		},
		jwt.WithIssuer(s.issuer),
		jwt.WithSubject(s.subject),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
