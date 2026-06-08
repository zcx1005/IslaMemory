package video

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler 视频模块 HTTP 处理器
// 负责视频列表、详情、上传、点赞、收藏、评论、播放记录、历史记录、弹幕等接口
type Handler struct {
	service *Service
	hub     *DanmakuHub // 弹幕实时广播中心
}

const (
	// maxUploadBytes 单文件上传大小限制，当前为 200MB
	maxUploadBytes = 200 * 1024 * 1024

	// localStorageRoot 本地文件存储根目录
	localStorageRoot = "./storage"

	// publicIDPrefix 视频公开 ID 前缀
	publicIDPrefix = "IV"

	// publicIDLength 视频公开 ID 总长度，例如：IV2Qw8ErT5Yu
	publicIDLength = 12
)

// NewHandler 创建视频模块 Handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		hub:     NewDanmakuHub(),
	}
}

// CreateCommentRequest 创建评论请求体
type CreateCommentRequest struct {
	ParentID      *uint64 `json:"parent_id"`        // 父评论 ID，nil 表示一级评论
	ReplyToUserID *uint64 `json:"reply_to_user_id"` // 被回复用户 ID
	Content       string  `json:"content" binding:"required"`
}

// UploadInitRequest 初始化分片上传请求体
type UploadInitRequest struct {
	Title        string `json:"title" binding:"required"`         // 视频标题
	Description  string `json:"description"`                      // 视频描述
	CategorySlug string `json:"category_slug" binding:"required"` // 分类标识
	Filename     string `json:"filename" binding:"required"`      // 原始文件名
	TotalSize    uint64 `json:"total_size" binding:"required"`    // 文件总大小
	ChunkSize    uint64 `json:"chunk_size"`                       // 每个分片大小
	TotalChunks  int    `json:"total_chunks" binding:"required"`  // 总分片数
}

// DanmakuRequest 创建弹幕请求体
type DanmakuRequest struct {
	Content string `json:"content" binding:"required"` // 弹幕内容
	TimeMs  uint32 `json:"time_ms"`                    // 弹幕出现时间，单位毫秒
	Color   string `json:"color"`                      // 弹幕颜色
	Mode    string `json:"mode"`                       // 弹幕模式
}

// List 获取公开视频列表
// 支持分页、分类筛选、关键词搜索和排序
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	input := ListInput{
		Page:         page,
		PageSize:     pageSize,
		CategorySlug: c.Query("category_slug"),
		Keyword:      c.Query("keyword"),
		Sort:         c.DefaultQuery("sort", "recommend"),
	}

	list, total, err := h.service.ListPublicVideos(c.Request.Context(), input)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "get videos failed"})
		return
	}

	c.JSON(200, gin.H{
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

// Detail 获取视频详情
func (h *Handler) Detail(c *gin.Context) {
	d, err := h.service.GetPublicVideoDetail(c.Request.Context(), c.Param("public_id"))
	if err != nil {
		if errors.Is(err, ErrVideoNotFound) {
			c.JSON(404, gin.H{"code": 404, "msg": err.Error()})
		} else {
			c.JSON(500, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": d})
}

// Upload 普通视频上传接口
//
// 流程：
// 1. 获取当前登录用户 ID
// 2. 校验标题、分类、文件大小和文件格式
// 3. 将原始视频保存到本地 storage 目录
// 4. 创建视频记录
// 5. 启动异步转码任务
func (h *Handler) Upload(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	categorySlug := strings.TrimSpace(c.PostForm("category_slug"))

	if title == "" || categorySlug == "" {
		c.JSON(400, gin.H{"code": 400, "msg": "title and category are required"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "file is required"})
		return
	}

	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		c.JSON(400, gin.H{"code": 400, "msg": "file too large or empty"})
		return
	}

	// 获取并校验视频文件后缀
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !isAllowedVideoExt(ext) {
		c.JSON(400, gin.H{"code": 400, "msg": "unsupported video format"})
		return
	}

	// 按日期创建存储目录，例如：videos/source/2026/06/08
	now := time.Now()
	dateDir := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	relDir := filepath.ToSlash(filepath.Join("videos", "source", dateDir))
	absDir := filepath.Join(localStorageRoot, relDir)

	if err := os.MkdirAll(absDir, 0755); err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "create upload dir failed"})
		return
	}

	// 生成随机文件名，避免用户上传文件名冲突
	nameToken, err := randomHex(16)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "generate filename failed"})
		return
	}

	filename := nameToken + ext
	absPath := filepath.Join(absDir, filename)

	// 保存上传文件到本地磁盘
	if err := c.SaveUploadedFile(fileHeader, absPath); err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "save file failed"})
		return
	}

	// 生成视频公开 ID
	publicID, err := generateVideoPublicID()
	if err != nil {
		_ = os.Remove(absPath)
		c.JSON(500, gin.H{"code": 500, "msg": "generate public id failed"})
		return
	}

	sourceURL := "/" + filepath.ToSlash(filepath.Join("static", relDir, filename))

	// 创建视频数据库记录
	v, err := h.service.CreateUploadedVideo(c.Request.Context(), CreateVideoInput{
		PublicID:       publicID,
		UserID:         userID,
		CategorySlug:   categorySlug,
		Title:          title,
		Description:    description,
		SourceVideoURL: sourceURL,
		FileSizeBytes:  uint64(fileHeader.Size),
		Status:         2,
	})
	if err != nil {
		_ = os.Remove(absPath)
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 启动异步转码任务
	h.service.StartAsyncTranscode(publicID, absPath)

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "upload success, transcoding started",
		"data": gin.H{
			"public_id":        v.PublicID,
			"status":           v.Status,
			"transcode_status": v.TranscodeStatus,
		},
	})
}

// InitChunkUpload 初始化分片上传任务
// 前端在上传大文件前先调用该接口，后端返回 upload_id
func (h *Handler) InitChunkUpload(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "invalid request"})
		return
	}

	res, err := h.service.InitChunkUpload(c.Request.Context(), UploadInitInput{
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		CategorySlug: req.CategorySlug,
		Filename:     req.Filename,
		TotalSize:    req.TotalSize,
		ChunkSize:    req.ChunkSize,
		TotalChunks:  req.TotalChunks,
	})
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": res})
}

// UploadChunk 上传单个视频分片
func (h *Handler) UploadChunk(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	uploadID := c.Param("upload_id")

	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "invalid chunk index"})
		return
	}

	fh, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "chunk is required"})
		return
	}

	// 分片保存目录：storage/chunks/{upload_id}
	dir := filepath.Join(localStorageRoot, "chunks", uploadID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "create chunk dir failed"})
		return
	}

	// 分片文件名：index.part
	dst := filepath.Join(dir, strconv.Itoa(idx)+".part")

	// 如果分片已存在，则不重复保存，支持断点续传
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := c.SaveUploadedFile(fh, dst); err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "save chunk failed"})
			return
		}
	}

	if err := h.service.UploadChunk(c.Request.Context(), uploadID, userID, idx, uint64(fh.Size)); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "chunk uploaded"})
}

// UploadStatus 查询分片上传状态
// 用于前端断点续传时判断哪些分片已经上传
func (h *Handler) UploadStatus(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	res, err := h.service.UploadStatus(c.Request.Context(), c.Param("upload_id"), userID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": res})
}

// CompleteChunkUpload 完成分片上传
// 后端会合并所有分片，生成完整视频文件，并启动异步转码
func (h *Handler) CompleteChunkUpload(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	res, err := h.service.CompleteChunkUpload(c.Request.Context(), c.Param("upload_id"), userID, localStorageRoot)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "upload complete, transcoding started", "data": res})
}

// InteractionState 获取当前用户对某个视频的点赞和收藏状态
func (h *Handler) InteractionState(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	state, err := h.service.GetInteractionState(c.Request.Context(), c.Param("public_id"), userID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": state})
}

// Like 点赞视频
func (h *Handler) Like(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	liked, err := h.service.LikeVideo(c.Request.Context(), c.Param("public_id"), userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "like failed"})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"liked": true, "changed": liked}})
}

// Unlike 取消点赞视频
func (h *Handler) Unlike(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	unliked, err := h.service.UnlikeVideo(c.Request.Context(), c.Param("public_id"), userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "unlike failed"})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"liked": false, "unliked": unliked}})
}

// Favorite 收藏视频
func (h *Handler) Favorite(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	changed, err := h.service.FavoriteVideo(c.Request.Context(), c.Param("public_id"), userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "favorite failed"})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"favorited": true, "changed": changed}})
}

// Unfavorite 取消收藏视频
func (h *Handler) Unfavorite(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	changed, err := h.service.UnfavoriteVideo(c.Request.Context(), c.Param("public_id"), userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "unfavorite failed"})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"favorited": false, "unfavorited": changed}})
}

// ListComments 获取视频评论列表
func (h *Handler) ListComments(c *gin.Context) {
	comments, err := h.service.ListComments(c.Request.Context(), c.Param("public_id"))
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"list": comments}})
}

// CreateComment 创建视频评论
// 支持一级评论、回复评论、回复用户
func (h *Handler) CreateComment(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "invalid request"})
		return
	}

	item, err := h.service.CreateComment(c.Request.Context(), CreateCommentInput{
		PublicID:      c.Param("public_id"),
		UserID:        userID,
		ParentID:      req.ParentID,
		ReplyToUserID: req.ReplyToUserID,
		Content:       req.Content,
	})
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "comment created", "data": item})
}

// IncreasePlay 记录视频播放行为
// 支持登录用户和游客，用 viewerKey 防止短时间内重复刷播放量
func (h *Handler) IncreasePlay(c *gin.Context) {
	var req struct {
		ProgressSeconds uint32 `json:"progress_seconds"` // 当前播放进度，单位秒
	}

	_ = c.ShouldBindJSON(&req)

	var userID *uint64

	// 如果用户已登录，则记录用户 ID；未登录则按游客处理
	if v, ok := c.Get("user_id"); ok {
		if uid, ok := v.(uint64); ok && uid > 0 {
			userID = &uid
		}
	}

	key := viewerKey(c, userID)

	counted, err := h.service.RegisterPlay(c.Request.Context(), c.Param("public_id"), userID, key, req.ProgressSeconds)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "increase play count failed"})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "play recorded", "data": gin.H{"counted": counted}})
}

// History 获取当前登录用户的视频观看历史
func (h *Handler) History(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.service.ListHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "msg": "get history failed"})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ListDanmaku 获取某个视频的弹幕列表
func (h *Handler) ListDanmaku(c *gin.Context) {
	items, err := h.service.ListDanmaku(c.Request.Context(), c.Param("public_id"))
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "msg": "success", "data": gin.H{"list": items}})
}

// CreateDanmaku 创建弹幕
// 弹幕创建成功后，会通过 DanmakuHub 实时推送给正在观看该视频的用户
func (h *Handler) CreateDanmaku(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req DanmakuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "invalid request"})
		return
	}

	item, err := h.service.CreateDanmaku(c.Request.Context(), CreateDanmakuInput{
		PublicID: c.Param("public_id"),
		UserID:   userID,
		Content:  req.Content,
		TimeMs:   req.TimeMs,
		Color:    req.Color,
		Mode:     req.Mode,
	})
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 实时广播新弹幕
	h.hub.Publish(item)

	c.JSON(200, gin.H{"code": 200, "msg": "danmaku created", "data": item})
}

// DanmakuStream 建立 SSE 弹幕实时推送连接
// 前端可通过 EventSource 监听该接口，实时接收新弹幕
func (h *Handler) DanmakuStream(c *gin.Context) {
	publicID := c.Param("public_id")

	ch, unsubscribe := h.hub.Subscribe(publicID)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(500, gin.H{"code": 500, "msg": "stream unsupported"})
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return

		case item := <-ch:
			b, _ := json.Marshal(item)

			// SSE 格式：event + data
			fmt.Fprintf(c.Writer, "event: danmaku\ndata: %s\n\n", b)
			flusher.Flush()
		}
	}
}

// getUserID 从 Gin Context 中获取当前登录用户 ID
func getUserID(c *gin.Context) (uint64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"code": 401, "msg": "unauthorized"})
		return 0, false
	}

	id, ok := v.(uint64)
	if !ok || id == 0 {
		c.JSON(401, gin.H{"code": 401, "msg": "invalid user id"})
		return 0, false
	}

	return id, true
}

// isAllowedVideoExt 判断视频文件后缀是否合法
func isAllowedVideoExt(ext string) bool {
	return allowedVideoExt(ext)
}

// randomHex 生成指定字节长度的随机十六进制字符串
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

// generateVideoPublicID 生成视频公开 ID
// 例如：IV2Qw8ErT5Yu
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

// randomAlphaNum 生成随机字母数字字符串
func randomAlphaNum(n int) (string, error) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	buf := make([]byte, n)
	rb := make([]byte, n)

	if _, err := rand.Read(rb); err != nil {
		return "", err
	}

	for i := 0; i < n; i++ {
		buf[i] = alphabet[int(rb[i])%len(alphabet)]
	}

	return string(buf), nil
}

// viewerKey 生成观看者唯一标识
//
// 登录用户：u:用户ID
// 游客用户：a:IP + User-Agent 的 SHA1 哈希
func viewerKey(c *gin.Context, userID *uint64) string {
	if userID != nil {
		return fmt.Sprintf("u:%d", *userID)
	}

	h := sha1.Sum([]byte(c.ClientIP() + "|" + c.GetHeader("User-Agent")))
	return "a:" + hex.EncodeToString(h[:])
}

// DanmakuHub 弹幕实时广播中心
// 按 publicID 维护不同视频的订阅者连接
type DanmakuHub struct {
	mu   sync.Mutex
	subs map[string]map[chan DanmakuItem]struct{}
}

// NewDanmakuHub 创建弹幕广播中心
func NewDanmakuHub() *DanmakuHub {
	return &DanmakuHub{
		subs: map[string]map[chan DanmakuItem]struct{}{},
	}
}

// Subscribe 订阅某个视频的弹幕流
// 返回弹幕通道和取消订阅函数
func (h *DanmakuHub) Subscribe(publicID string) (chan DanmakuItem, func()) {
	ch := make(chan DanmakuItem, 16)

	h.mu.Lock()
	if h.subs[publicID] == nil {
		h.subs[publicID] = map[chan DanmakuItem]struct{}{}
	}
	h.subs[publicID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs[publicID], ch)
		close(ch)
		h.mu.Unlock()
	}
}

// Publish 广播新弹幕
// 将弹幕推送给当前视频的所有在线订阅者
func (h *DanmakuHub) Publish(item *DanmakuItem) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subs[item.PublicID] {
		select {
		case ch <- *item:
		default:
			// 如果通道已满，直接跳过，避免阻塞整个广播流程
		}
	}
}
