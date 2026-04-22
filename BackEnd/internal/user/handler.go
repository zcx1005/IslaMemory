package user

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type RegisterRequest struct {
	Account  string `json:"account" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type UpdateProfileRequest struct {
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

func getCurrentUserID(c *gin.Context) (uint64, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "unauthorized"})
		return 0, false
	}

	userID, ok := userIDVal.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid user id"})
		return 0, false
	}
	return userID, true
}

func buildAvatarPath(ext string) (string, string, error) {
	now := time.Now()
	relDir := filepath.ToSlash(filepath.Join("avatars", fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()), fmt.Sprintf("%02d", now.Day())))
	absDir := filepath.Join("./storage", relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", "", err
	}
	filename := fmt.Sprintf("%d%s", now.UnixNano(), ext)
	relPath := "/" + filepath.ToSlash(filepath.Join("static", relDir, filename))
	absPath := filepath.Join(absDir, filename)
	return relPath, absPath, nil
}

func getAvatarURLFromRequest(c *gin.Context) (*string, error) {
	fileHeader, err := c.FormFile("avatar")
	if err == nil {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		default:
			return nil, fmt.Errorf("unsupported avatar format")
		}
		relPath, absPath, err := buildAvatarPath(ext)
		if err != nil {
			return nil, err
		}
		if err := c.SaveUploadedFile(fileHeader, absPath); err != nil {
			return nil, err
		}
		return &relPath, nil
	}

	avatarURL := strings.TrimSpace(c.PostForm("avatar_url"))
	if avatarURL != "" {
		return &avatarURL, nil
	}
	return nil, nil
}

// 注册
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request",
		})
		return
	}

	u, err := h.service.Register(c.Request.Context(), req.Account, req.Username, req.Password)
	if err != nil {
		switch err {
		case ErrAccountExists, ErrUsernameExists:
			c.JSON(http.StatusConflict, gin.H{
				"code": 409,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "register success",
		"data": gin.H{
			"id":       u.ID,
			"account":  u.Account,
			"username": u.Username,
		},
	})
}

// 登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request",
		})
		return
	}

	token, u, err := h.service.Login(c.Request.Context(), req.Account, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidCredentials, ErrUserDisabled:
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "login success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":         u.ID,
				"account":    u.Account,
				"username":   u.Username,
				"avatar_url": u.AvatarURL,
				"role":       u.Role,
			},
		},
	})
}

// 当前用户信息
func (h *Handler) Me(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	u, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"code": 404,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"id":                  u.ID,
			"account":             u.Account,
			"username":            u.Username,
			"avatar_url":          u.AvatarURL,
			"role":                u.Role,
			"status":              u.Status,
			"can_upload":          u.CanUpload,
			"password_changed_at": u.PasswordChangedAt,
		},
	})
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	var username *string
	var avatarURL *string

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		name := strings.TrimSpace(c.PostForm("username"))
		if name != "" {
			username = &name
		}
		avatar, err := getAvatarURLFromRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		avatarURL = avatar
	} else {
		var req UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid request"})
			return
		}
		username = req.Username
		avatarURL = req.AvatarURL
	}

	u, err := h.service.UpdateProfile(c.Request.Context(), userID, username, avatarURL)
	if err != nil {
		switch err {
		case ErrUsernameExists:
			c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": u})
}

func (h *Handler) MyFavorites(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, err := h.service.ListMyFavoriteVideos(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "get favorites failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list})
}

func (h *Handler) MyUploads(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, err := h.service.ListMyUploadedVideos(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "get uploads failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": list})
}

// 修改密码
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request",
		})
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch err {
		case ErrInvalidCredentials, ErrUserNotFound:
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "password changed",
	})
}
