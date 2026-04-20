package video

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler 负责 video HTTP 请求处理
type Handler struct {
	service *Service
}

const (
	// maxUploadBytes 单文件上传大小限制（200MB）
	maxUploadBytes = 200 * 1024 * 1024
	// localStorageRoot 本地存储根目录
	localStorageRoot = "./storage"
	// publicIDPrefix 视频 public_id 前缀
	publicIDPrefix = "IV"
	// publicIDLength 视频 public_id 总长度，例如：IV2Qw8ErT5Yu
	publicIDLength = 12
)

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List 处理 GET /api/v1/videos
// 支持：分页、分类筛选、关键词搜索、排序
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	input := ListInput{
		Page:         page,
		PageSize:     pageSize,
		CategorySlug: c.Query("category_slug"),
		Keyword:      c.Query("keyword"),
		Sort:         c.DefaultQuery("sort", "latest"),
	}

	list, total, err := h.service.ListPublicVideos(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "get videos failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      input.Page,
			"page_size": input.PageSize,
		},
	})
}

// Detail 处理 GET /api/v1/videos/:public_id
func (h *Handler) Detail(c *gin.Context) {
	publicID := c.Param("public_id")
	if publicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid public_id",
		})
		return
	}

	detail, err := h.service.GetPublicVideoDetail(c.Request.Context(), publicID)
	if err != nil {
		switch err {
		case ErrVideoNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": detail,
	})
}

// TODO: IncreasePlay 处理 POST /api/v1/videos/:public_id/play
func (h *Handler) IncreasePlay(c *gin.Context) {
	publicID := c.Param("public_id")
	if publicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid public_id",
		})
		return
	}

	if err := h.service.IncreasePlayCount(c.Request.Context(), publicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "increase play count failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "play count increased",
	})
}

// Upload 处理 POST /api/v1/videos/upload
// 第一版：视频文件先存本地磁盘，再写入 videos 表
func (h *Handler) Upload(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uint64)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid user id"})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	categoryIDStr := strings.TrimSpace(c.PostForm("category_id"))
	categoryID, _ := strconv.ParseUint(categoryIDStr, 10, 64)

	if title == "" || categoryID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "title and category_id are required"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "file is required"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "file too large or empty"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !isAllowedVideoExt(ext) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "unsupported video format"})
		return
	}

	now := time.Now()
	dateDir := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	relativeDir := filepath.ToSlash(filepath.Join("videos", "source", dateDir))
	absoluteDir := filepath.Join(localStorageRoot, relativeDir)
	if err := os.MkdirAll(absoluteDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "create upload dir failed"})
		return
	}

	nameToken, err := randomHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "generate filename failed"})
		return
	}
	filename := nameToken + ext

	absolutePath := filepath.Join(absoluteDir, filename)
	if err := c.SaveUploadedFile(fileHeader, absolutePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "save file failed"})
		return
	}

	publicID, err := generateVideoPublicID()
	if err != nil {
		_ = os.Remove(absolutePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "generate public id failed"})
		return
	}

	sourceURL := "/" + filepath.ToSlash(filepath.Join("static", relativeDir, filename))

	video, err := h.service.CreateUploadedVideo(c.Request.Context(), CreateVideoInput{
		PublicID:       publicID,
		UserID:         userID,
		CategoryID:     categoryID,
		Title:          title,
		Description:    description,
		SourceVideoURL: sourceURL,
		FileSizeBytes:  uint64(fileHeader.Size),
		Status:         2,
	})
	if err != nil {
		_ = os.Remove(absolutePath)
		if errors.Is(err, ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "create video record failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "upload success",
		"data": gin.H{
			"public_id":        video.PublicID,
			"title":            video.Title,
			"description":      video.Description,
			"source_video_url": video.SourceVideoURL,
			"playback_url":     video.SourceVideoURL,
			"file_size_bytes":  video.FileSizeBytes,
			"status":           video.Status,
		},
	})
}

func isAllowedVideoExt(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".webm":
		return true
	default:
		return false
	}
}

func randomHex(bytesLen int) (string, error) {
	if bytesLen <= 0 {
		return "", errors.New("invalid length")
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// generateVideoPublicID 生成以 "IV" 开头的公开 ID（固定长度 12）
// 示例：IV2Qw8ErT5Yu
func generateVideoPublicID() (string, error) {
	suffixLen := publicIDLength - len(publicIDPrefix)
	if suffixLen <= 0 {
		return "", errors.New("invalid public id config")
	}

	suffix, err := randomAlphaNum(suffixLen)
	if err != nil {
		return "", err
	}
	return publicIDPrefix + suffix, nil
}

func randomAlphaNum(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("invalid length")
	}

	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, n)
	randomBytes := make([]byte, n)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		buf[i] = alphabet[int(randomBytes[i])%len(alphabet)]
	}
	return string(buf), nil
}
