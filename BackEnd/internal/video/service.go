package video

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 业务层统一错误定义
// Handler 层可以根据这些错误返回对应的 HTTP 状态码和提示信息
var (
	ErrVideoNotFound       = errors.New("video not found")          // 视频不存在
	ErrInvalidInput        = errors.New("invalid video input")      // 视频相关输入参数不合法
	ErrCommentNotFound     = errors.New("comment not found")        // 评论不存在
	ErrInvalidCommentInput = errors.New("invalid comment input")    // 评论输入不合法
	ErrUploadNotFound      = errors.New("upload session not found") // 分片上传会话不存在
)

// Service 视频模块业务层
//
// 主要职责：
// 1. 处理视频列表、详情、上传、转码等业务逻辑
// 2. 处理播放统计、点赞、收藏、评论、弹幕等交互逻辑
// 3. 调用 Repository 完成数据库读写
// 4. 对输入参数进行校验，并转换返回给前端的数据结构
type Service struct {
	repo *Repository
	rdb  *redis.Client
}

// NewService 创建视频业务层实例
func NewService(repo *Repository, rdb ...*redis.Client) *Service {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return &Service{repo: repo, rdb: client}
}

// ListInput 视频列表查询输入参数
type ListInput struct {
	Page, PageSize              int    // 分页参数
	CategorySlug, Keyword, Sort string // 分类、关键词、排序方式
}

// VideoListItem 视频列表返回项
// 用于首页、分类页、搜索页等视频列表展示
type VideoListItem struct {
	PublicID         string  `json:"public_id"`         // 视频公开 ID
	Title            string  `json:"title"`             // 视频标题
	Description      string  `json:"description"`       // 视频描述
	CoverURL         string  `json:"cover_url"`         // 封面地址
	DurationSeconds  uint32  `json:"duration_seconds"`  // 视频时长，单位秒
	Width            uint32  `json:"width"`             // 视频宽度
	Height           uint32  `json:"height"`            // 视频高度
	PlayCount        uint64  `json:"play_count"`        // 播放数
	LikeCount        uint64  `json:"like_count"`        // 点赞数
	FavoriteCount    uint64  `json:"favorite_count"`    // 收藏数
	CommentCount     uint64  `json:"comment_count"`     // 评论数
	CategoryID       uint64  `json:"category_id"`       // 分类 ID
	CategoryName     string  `json:"category_name"`     // 分类名称
	CategorySlug     string  `json:"category_slug"`     // 分类标识
	PublishedAt      *string `json:"published_at"`      // 发布时间
	CreatedAt        string  `json:"created_at"`        // 创建时间
	Username         string  `json:"username"`          // 上传者用户名
	UploaderUsername string  `json:"uploader_username"` // 上传者用户名，兼容前端字段
}

// VideoDetail 视频详情返回结构
// 用于视频播放页展示完整视频信息
type VideoDetail struct {
	PublicID          string  `json:"public_id"`          // 视频公开 ID
	Title             string  `json:"title"`              // 视频标题
	Description       string  `json:"description"`        // 视频描述
	CoverURL          string  `json:"cover_url"`          // 封面地址
	DurationSeconds   uint32  `json:"duration_seconds"`   // 视频时长
	Width             uint32  `json:"width"`              // 视频宽度
	Height            uint32  `json:"height"`             // 视频高度
	PlayCount         uint64  `json:"play_count"`         // 播放数
	LikeCount         uint64  `json:"like_count"`         // 点赞数
	FavoriteCount     uint64  `json:"favorite_count"`     // 收藏数
	CommentCount      uint64  `json:"comment_count"`      // 评论数
	CategoryID        uint64  `json:"category_id"`        // 分类 ID
	CategoryName      string  `json:"category_name"`      // 分类名称
	CategorySlug      string  `json:"category_slug"`      // 分类标识
	PlaybackType      uint8   `json:"playback_type"`      // 播放类型：0 原视频，1 HLS
	PlaybackURL       string  `json:"playback_url"`       // 播放地址
	TranscodeStatus   uint8   `json:"transcode_status"`   // 转码状态
	TranscodeProgress uint32  `json:"transcode_progress"` // 转码进度
	PublishedAt       *string `json:"published_at"`       // 发布时间
	CreatedAt         string  `json:"created_at"`         // 创建时间
	Username          string  `json:"username"`           // 上传者用户名
	UploaderUsername  string  `json:"uploader_username"`  // 上传者用户名，兼容前端字段
}

// CreateVideoInput 创建视频记录输入参数
type CreateVideoInput struct {
	PublicID                                         string // 视频公开 ID
	UserID, CategoryID                               uint64 // 上传用户 ID、分类 ID
	CategorySlug, Title, Description, SourceVideoURL string // 分类标识、标题、描述、原始视频地址
	FileSizeBytes                                    uint64 // 文件大小
	Status                                           uint8  // 视频状态
}

// InteractionState 当前用户对视频的交互状态
type InteractionState struct {
	Liked     bool `json:"liked"`     // 是否已点赞
	Favorited bool `json:"favorited"` // 是否已收藏
}

// CommentItem 评论返回结构
// 支持一级评论和回复列表
type CommentItem struct {
	ID              uint64        `json:"id"`                // 评论 ID
	VideoID         uint64        `json:"video_id"`          // 视频 ID
	UserID          uint64        `json:"user_id"`           // 评论用户 ID
	Username        string        `json:"username"`          // 评论用户名
	AvatarURL       string        `json:"avatar_url"`        // 评论用户头像
	ParentID        *uint64       `json:"parent_id"`         // 父评论 ID
	RootID          *uint64       `json:"root_id"`           // 根评论 ID
	ReplyToUserID   *uint64       `json:"reply_to_user_id"`  // 被回复用户 ID
	ReplyToUsername *string       `json:"reply_to_username"` // 被回复用户名
	Content         string        `json:"content"`           // 评论内容
	LikeCount       uint64        `json:"like_count"`        // 评论点赞数
	CreatedAt       string        `json:"created_at"`        // 创建时间
	UpdatedAt       string        `json:"updated_at"`        // 更新时间
	Replies         []CommentItem `json:"replies"`           // 回复列表
}

// CreateCommentInput 创建评论输入参数
type CreateCommentInput struct {
	PublicID                string  // 视频公开 ID
	UserID                  uint64  // 评论用户 ID
	ParentID, ReplyToUserID *uint64 // 父评论 ID、被回复用户 ID
	Content                 string  // 评论内容
}

// UploadInitInput 初始化分片上传输入参数
type UploadInitInput struct {
	UserID                                     uint64 // 上传用户 ID
	Title, Description, CategorySlug, Filename string // 标题、描述、分类标识、文件名
	TotalSize, ChunkSize                       uint64 // 文件总大小、分片大小
	TotalChunks                                int    // 总分片数
}

// UploadInitResult 初始化分片上传返回结果
type UploadInitResult struct {
	UploadID       string `json:"upload_id"`       // 上传会话 ID
	UploadedChunks []int  `json:"uploaded_chunks"` // 已上传分片序号
	ChunkSize      uint64 `json:"chunk_size"`      // 分片大小
	TotalChunks    int    `json:"total_chunks"`    // 总分片数
}

// UploadStatusResult 分片上传状态返回结果
type UploadStatusResult struct {
	UploadID       string `json:"upload_id"`       // 上传会话 ID
	UploadedChunks []int  `json:"uploaded_chunks"` // 已上传分片序号
	TotalChunks    int    `json:"total_chunks"`    // 总分片数
	Completed      bool   `json:"completed"`       // 是否已完成
}

// UploadCompleteResult 分片上传完成返回结果
type UploadCompleteResult struct {
	PublicID        string `json:"public_id"`        // 视频公开 ID
	Status          uint8  `json:"status"`           // 视频状态
	TranscodeStatus uint8  `json:"transcode_status"` // 转码状态
}

// DanmakuItem 弹幕返回结构
type DanmakuItem struct {
	ID        uint64 `json:"id"`         // 弹幕 ID
	PublicID  string `json:"public_id"`  // 视频公开 ID
	UserID    uint64 `json:"user_id"`    // 发送用户 ID
	Content   string `json:"content"`    // 弹幕内容
	TimeMs    uint32 `json:"time_ms"`    // 弹幕出现时间，单位毫秒
	Color     string `json:"color"`      // 弹幕颜色
	Mode      string `json:"mode"`       // 弹幕模式
	CreatedAt string `json:"created_at"` // 创建时间
}

// CreateDanmakuInput 创建弹幕输入参数
type CreateDanmakuInput struct {
	PublicID    string // 视频公开 ID
	UserID      uint64 // 发送用户 ID
	Content     string // 弹幕内容
	TimeMs      uint32 // 出现时间，单位毫秒
	Color, Mode string // 颜色、模式
}

// normalizeListInput 标准化列表查询参数
// 避免非法分页参数影响查询
func (s *Service) normalizeListInput(in ListInput) ListInput {
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	}
	if in.PageSize > 100 {
		in.PageSize = 100
	}
	if in.Sort == "" {
		in.Sort = "recommend"
	}
	return in
}

// ListPublicVideos 获取公开视频列表
// 先标准化分页参数，再调用 Repository 查询，最后转换为前端需要的结构
func (s *Service) ListPublicVideos(ctx context.Context, in ListInput) ([]VideoListItem, int64, error) {
	in = s.normalizeListInput(in)
	if in.Sort == "latest" && in.CategorySlug == "" && in.Keyword == "" {
		if list, total, ok := s.getLatestPublicVideosFromCache(ctx, in.Page, in.PageSize); ok {
			return list, total, nil
		}
	}
	rows, total, err := s.repo.ListPublicVideos(ctx, ListParams{Page: in.Page, PageSize: in.PageSize, CategorySlug: in.CategorySlug, Keyword: in.Keyword, Sort: in.Sort})
	if err != nil {
		return nil, 0, err
	}
	if in.Sort == "latest" && in.CategorySlug == "" && in.Keyword == "" {
		s.cacheLatestPublicVideos(ctx, rows, total)
	}
	return toVideoList(rows), total, nil
}

// toVideoList 将数据库查询结果 VideoRow 转换为接口返回的 VideoListItem
func toVideoList(rows []VideoRow) []VideoListItem {
	list := make([]VideoListItem, 0, len(rows))
	for _, row := range rows {
		item := VideoListItem{PublicID: row.PublicID, Title: row.Title, Description: row.Description, CoverURL: row.CoverURL, DurationSeconds: row.DurationSeconds, Width: row.Width, Height: row.Height, PlayCount: row.PlayCount, LikeCount: row.LikeCount, FavoriteCount: row.FavoriteCount, CommentCount: row.CommentCount, CategoryID: row.CategoryID, CategoryName: row.CategoryName, CategorySlug: row.CategorySlug, CreatedAt: row.CreatedAt.Format(timeFormat), Username: row.Username, UploaderUsername: row.Username}
		if row.PublishedAt != nil {
			t := row.PublishedAt.Format(timeFormat)
			item.PublishedAt = &t
		}
		list = append(list, item)
	}
	return list
}

// GetPublicVideoDetail 根据公开 ID 获取视频详情
// 如果转码播放地址为空，则回退使用原始视频地址播放
func (s *Service) GetPublicVideoDetail(ctx context.Context, publicID string) (*VideoDetail, error) {
	if !s.mightContainVideo(ctx, publicID) {
		s.setCachedVideoMiss(ctx, publicID)
		return nil, ErrVideoNotFound
	}
	if d, ok, err := s.getCachedDetail(ctx, publicID); ok || err != nil {
		return d, err
	}

	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			s.setCachedVideoMiss(ctx, publicID)
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	playURL := row.PlaybackURL
	if playURL == "" {
		playURL = row.SourceVideoURL
	}
	d := &VideoDetail{PublicID: row.PublicID, Title: row.Title, Description: row.Description, CoverURL: row.CoverURL, DurationSeconds: row.DurationSeconds, Width: row.Width, Height: row.Height, PlayCount: row.PlayCount, LikeCount: row.LikeCount, FavoriteCount: row.FavoriteCount, CommentCount: row.CommentCount, CategoryID: row.CategoryID, CategoryName: row.CategoryName, CategorySlug: row.CategorySlug, PlaybackType: row.PlaybackType, PlaybackURL: playURL, TranscodeStatus: row.TranscodeStatus, TranscodeProgress: row.TranscodeProgress, CreatedAt: row.CreatedAt.Format(timeFormat), Username: row.Username, UploaderUsername: row.Username}
	if row.PublishedAt != nil {
		t := row.PublishedAt.Format(timeFormat)
		d.PublishedAt = &t
	}
	s.setCachedDetail(ctx, row, d)
	return d, nil
}

// CreateUploadedVideo 创建已上传视频记录
//
// 主要流程：
// 1. 校验视频公开 ID、用户 ID、标题、原始视频地址
// 2. 如果只传了 CategorySlug，则查询对应 CategoryID
// 3. 设置默认发布状态、转码状态
// 4. 写入 videos 表
func (s *Service) CreateUploadedVideo(ctx context.Context, in CreateVideoInput) (*Video, error) {
	if in.PublicID == "" || in.UserID == 0 || strings.TrimSpace(in.Title) == "" || in.SourceVideoURL == "" {
		return nil, ErrInvalidInput
	}
	if in.CategoryID == 0 && in.CategorySlug != "" {
		id, err := s.repo.GetCategoryIDBySlug(ctx, in.CategorySlug)
		if err != nil {
			if IsNotFound(err) {
				return nil, ErrInvalidInput
			}
			return nil, err
		}
		in.CategoryID = id
	}
	if in.CategoryID == 0 {
		return nil, ErrInvalidInput
	}
	now := time.Now()
	status := in.Status
	if status == 0 {
		status = 2
	}
	video := &Video{PublicID: in.PublicID, UserID: in.UserID, CategoryID: in.CategoryID, Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description), SourceVideoURL: in.SourceVideoURL, PlaybackType: 0, TranscodeStatus: 0, TranscodeProgress: 0, Status: status, FileSizeBytes: in.FileSizeBytes, PublishedAt: &now, ReviewedAt: &now}
	if err := s.repo.CreateVideo(ctx, video); err != nil {
		return nil, err
	}
	s.addVideoBloom(ctx, video.PublicID)
	s.invalidateVideoCache(ctx, video.PublicID)
	return video, nil
}

// StartAsyncTranscode 启动异步转码任务
// 使用 goroutine 后台执行，避免上传接口长时间阻塞
func (s *Service) StartAsyncTranscode(publicID, absoluteSourcePath string) {
	go s.transcodeVideo(context.Background(), publicID, absoluteSourcePath)
}

// transcodeVideo 使用 FFmpeg 进行视频转码
//
// 主要功能：
// 1. 更新转码状态为转码中
// 2. 生成视频封面 cover.jpg
// 3. 将视频转为 HLS 格式
// 4. 更新播放地址、封面地址和转码状态
func (s *Service) transcodeVideo(ctx context.Context, publicID, src string) {
	_ = s.repo.UpdateVideoTranscode(ctx, publicID, map[string]any{"transcode_status": 1, "transcode_progress": 5})

	// 如果服务器没有安装 ffmpeg，则直接使用原始视频播放
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		_ = s.repo.UpdateVideoTranscode(ctx, publicID, map[string]any{"transcode_status": 2, "transcode_progress": 100, "playback_type": 0, "transcode_error": "ffmpeg not found, source playback enabled"})
		s.invalidateVideoCache(ctx, publicID)
		return
	}

	// 输出目录：原视频目录的上级目录 / hls / 文件名
	outDir := filepath.Join(filepath.Dir(filepath.Dir(src)), "hls", strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)))
	_ = os.MkdirAll(outDir, 0755)

	// 截取第 1 秒作为视频封面
	coverPath := filepath.Join(outDir, "cover.jpg")
	_ = exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", "00:00:01", "-i", src, "-frames:v", "1", coverPath).Run()

	_ = s.repo.UpdateVideoTranscode(ctx, publicID, map[string]any{"transcode_progress": 25})

	// 生成 HLS 文件：index.m3u8 + 多个 ts 分片
	m3u8 := filepath.Join(outDir, "index.m3u8")
	err := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", src, "-c:v", "libx264", "-preset", "veryfast", "-c:a", "aac", "-hls_time", "6", "-hls_playlist_type", "vod", "-hls_segment_filename", filepath.Join(outDir, "seg_%03d.ts"), m3u8).Run()

	// 将本地存储路径转换成前端可访问的静态资源路径
	relBase := "/static/" + filepath.ToSlash(strings.TrimPrefix(outDir, "storage"))
	relBase = strings.Replace(relBase, "/./", "/", 1)
	if strings.HasPrefix(relBase, "/static//") {
		relBase = strings.Replace(relBase, "/static//", "/static/", 1)
	}

	values := map[string]any{"transcode_progress": 100}

	// 如果封面生成成功，则更新封面地址
	if _, statErr := os.Stat(coverPath); statErr == nil {
		values["cover_url"] = relBase + "/cover.jpg"
	}

	// 根据 ffmpeg 执行结果更新转码状态
	if err != nil {
		values["transcode_status"] = 3
		values["transcode_error"] = err.Error()
	} else {
		values["transcode_status"] = 2
		values["playback_type"] = 1
		values["playback_url"] = relBase + "/index.m3u8"
	}

	_ = s.repo.UpdateVideoTranscode(ctx, publicID, values)
	s.invalidateVideoCache(ctx, publicID)
}

// RegisterPlay 记录视频播放行为
// 支持登录用户和游客，Repository 层会负责防重复计数和观看历史更新
func (s *Service) RegisterPlay(ctx context.Context, publicID string, userID *uint64, viewerKey string, progress uint32) (bool, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	counted, err := s.repo.CreatePlayAndHistory(ctx, row.ID, userID, viewerKey, time.Now().Format("2006-01-02"), progress)
	if err == nil && counted {
		s.invalidateVideoCache(ctx, publicID)
		if userID != nil {
			s.deleteHistoryCache(ctx, *userID)
		}
	}
	return counted, err
}

// GetInteractionState 获取当前用户对视频的点赞和收藏状态
func (s *Service) GetInteractionState(ctx context.Context, publicID string, userID uint64) (*InteractionState, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	liked, err := s.repo.IsVideoLikedByUser(ctx, row.ID, userID)
	if err != nil {
		return nil, err
	}
	fav, err := s.repo.IsVideoFavoritedByUser(ctx, row.ID, userID)
	if err != nil {
		return nil, err
	}
	return &InteractionState{Liked: liked, Favorited: fav}, nil
}

// LikeVideo 点赞视频
func (s *Service) LikeVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	changed, err := s.repo.LikeVideo(ctx, row.ID, userID)
	if err == nil && changed {
		s.invalidateVideoCache(ctx, publicID)
	}
	return changed, err
}

// UnlikeVideo 取消点赞视频
func (s *Service) UnlikeVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	changed, err := s.repo.UnlikeVideo(ctx, row.ID, userID)
	if err == nil && changed {
		s.invalidateVideoCache(ctx, publicID)
	}
	return changed, err
}

// FavoriteVideo 收藏视频
func (s *Service) FavoriteVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	changed, err := s.repo.FavoriteVideo(ctx, row.ID, userID)
	if err == nil && changed {
		s.invalidateVideoCache(ctx, publicID)
	}
	return changed, err
}

// UnfavoriteVideo 取消收藏视频
func (s *Service) UnfavoriteVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	changed, err := s.repo.UnfavoriteVideo(ctx, row.ID, userID)
	if err == nil && changed {
		s.invalidateVideoCache(ctx, publicID)
	}
	return changed, err
}

// timeFormat 统一时间格式
const timeFormat = "2006-01-02 15:04:05"

// CreateComment 创建评论
//
// 支持：
// 1. 一级评论
// 2. 回复评论
// 3. 自动设置 root_id
// 4. 如果未传 reply_to_user_id，则默认回复父评论作者
func (s *Service) CreateComment(ctx context.Context, in CreateCommentInput) (*CommentItem, error) {
	content := strings.TrimSpace(in.Content)
	if in.PublicID == "" || in.UserID == 0 || content == "" {
		return nil, ErrInvalidCommentInput
	}

	row, err := s.repo.GetPublicVideoByPublicID(ctx, in.PublicID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	var rootID *uint64

	// 如果存在 ParentID，说明当前评论是回复评论
	if in.ParentID != nil {
		parent, err := s.repo.GetActiveCommentByID(ctx, *in.ParentID)
		if err != nil {
			if IsNotFound(err) {
				return nil, ErrCommentNotFound
			}
			return nil, err
		}

		// 防止跨视频回复评论
		if parent.VideoID != row.ID {
			return nil, ErrInvalidCommentInput
		}

		// 如果父评论没有 RootID，说明父评论本身是一级评论
		if parent.RootID == nil {
			id := parent.ID
			rootID = &id
		} else {
			rootID = parent.RootID
		}

		// 如果没有指定回复用户，则默认回复父评论作者
		if in.ReplyToUserID == nil {
			uid := parent.UserID
			in.ReplyToUserID = &uid
		}
	}

	cmt, err := s.repo.CreateComment(ctx, CreateCommentParams{VideoID: row.ID, UserID: in.UserID, ParentID: in.ParentID, RootID: rootID, ReplyToUserID: in.ReplyToUserID, Content: content})
	if err != nil {
		return nil, err
	}

	s.invalidateVideoCache(ctx, in.PublicID)
	return &CommentItem{ID: cmt.ID, VideoID: cmt.VideoID, UserID: cmt.UserID, ParentID: cmt.ParentID, RootID: cmt.RootID, ReplyToUserID: cmt.ReplyToUserID, Content: cmt.Content, LikeCount: cmt.LikeCount, CreatedAt: cmt.CreatedAt.Format(timeFormat), UpdatedAt: cmt.UpdatedAt.Format(timeFormat), Replies: []CommentItem{}}, nil
}

// ListComments 获取视频评论列表
// 将数据库中的扁平评论列表组装成“一级评论 + 回复列表”的结构
func (s *Service) ListComments(ctx context.Context, publicID string) ([]CommentItem, error) {
	var cached []CommentItem
	if s.getJSONCache(ctx, fmt.Sprintf(videoCommentsFmt, publicID), &cached) {
		return cached, nil
	}

	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	rows, err := s.repo.ListCommentsByVideoID(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	roots := []CommentItem{}             // 一级评论列表
	idx := map[uint64]int{}              // root comment id -> roots 下标
	bucket := map[uint64][]CommentItem{} // root comment id -> replies

	for _, r := range rows {
		item := CommentItem{ID: r.ID, VideoID: r.VideoID, UserID: r.UserID, Username: r.Username, AvatarURL: r.AvatarURL, ParentID: r.ParentID, RootID: r.RootID, ReplyToUserID: r.ReplyToUserID, ReplyToUsername: r.ReplyToUsername, Content: r.Content, LikeCount: r.LikeCount, CreatedAt: r.CreatedAt.Format(timeFormat), UpdatedAt: r.UpdatedAt.Format(timeFormat), Replies: []CommentItem{}}
		if r.ParentID == nil {
			// 没有 ParentID，说明是一级评论
			roots = append(roots, item)
			idx[item.ID] = len(roots) - 1
		} else {
			// 有 ParentID，说明是回复评论，先放入对应根评论的 bucket 中
			root := r.RootID
			if root == nil {
				root = r.ParentID
			}
			if root != nil {
				bucket[*root] = append(bucket[*root], item)
			}
		}
	}

	// 将回复评论挂载到对应一级评论下面
	for id, replies := range bucket {
		if i, ok := idx[id]; ok {
			roots[i].Replies = append(roots[i].Replies, replies...)
		}
	}

	s.setJSONCache(ctx, fmt.Sprintf(videoCommentsFmt, publicID), roots, videoItemCacheTTL)
	return roots, nil
}

// InitChunkUpload 初始化分片上传
//
// 主要流程：
// 1. 校验用户、标题、分类、文件大小、分片数量
// 2. 根据分类 slug 查询分类 ID
// 3. 校验视频文件后缀
// 4. 创建上传会话记录
func (s *Service) InitChunkUpload(ctx context.Context, in UploadInitInput) (*UploadInitResult, error) {
	if in.UserID == 0 || strings.TrimSpace(in.Title) == "" || in.CategorySlug == "" || in.TotalSize == 0 || in.TotalChunks <= 0 {
		return nil, ErrInvalidInput
	}
	cat, err := s.repo.GetCategoryIDBySlug(ctx, in.CategorySlug)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if in.ChunkSize == 0 {
		in.ChunkSize = 4 * 1024 * 1024
	}
	ext := strings.ToLower(filepath.Ext(in.Filename))
	if !allowedVideoExt(ext) {
		return nil, ErrInvalidInput
	}

	// 生成上传会话 ID
	id := fmt.Sprintf("UP%d%s", time.Now().UnixNano(), randomSuffix())

	sess := &UploadSession{UploadID: id, UserID: in.UserID, Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description), CategoryID: cat, Filename: filepath.Base(in.Filename), Ext: ext, TotalSize: in.TotalSize, ChunkSize: in.ChunkSize, TotalChunks: in.TotalChunks}
	if err := s.repo.CreateUploadSession(ctx, sess); err != nil {
		return nil, err
	}
	return &UploadInitResult{UploadID: id, UploadedChunks: []int{}, ChunkSize: in.ChunkSize, TotalChunks: in.TotalChunks}, nil
}

// UploadChunk 记录单个分片上传完成
// 实际文件保存由 Handler 层完成，这里只负责校验上传会话和写入分片记录
func (s *Service) UploadChunk(ctx context.Context, uploadID string, userID uint64, idx int, size uint64) error {
	sess, err := s.repo.GetUploadSession(ctx, uploadID, userID)
	if err != nil {
		return ErrUploadNotFound
	}
	if idx < 0 || idx >= sess.TotalChunks {
		return ErrInvalidInput
	}
	_, err = s.repo.AddUploadChunk(ctx, uploadID, idx, size)
	return err
}

// UploadStatus 查询分片上传状态
// 返回已上传分片列表，供前端断点续传使用
func (s *Service) UploadStatus(ctx context.Context, uploadID string, userID uint64) (*UploadStatusResult, error) {
	sess, err := s.repo.GetUploadSession(ctx, uploadID, userID)
	if err != nil {
		return nil, ErrUploadNotFound
	}
	chunks, err := s.repo.ListUploadChunkIndexes(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	return &UploadStatusResult{UploadID: uploadID, UploadedChunks: chunks, TotalChunks: sess.TotalChunks, Completed: sess.Status == 1}, nil
}

// CompleteChunkUpload 完成分片上传
//
// 主要流程：
// 1. 校验上传会话是否存在
// 2. 校验分片数量是否完整
// 3. 按顺序合并所有分片
// 4. 创建视频记录
// 5. 标记上传会话完成
// 6. 删除临时分片目录
// 7. 启动异步转码
func (s *Service) CompleteChunkUpload(ctx context.Context, uploadID string, userID uint64, storageRoot string) (*UploadCompleteResult, error) {
	var result *UploadCompleteResult
	err := s.withRedisLock(ctx, "complete:"+uploadID, videoLockTTL, func() error {
		var innerErr error
		result, innerErr = s.completeChunkUploadLocked(ctx, uploadID, userID, storageRoot)
		return innerErr
	})
	return result, err
}

func (s *Service) completeChunkUploadLocked(ctx context.Context, uploadID string, userID uint64, storageRoot string) (*UploadCompleteResult, error) {
	sess, err := s.repo.GetUploadSession(ctx, uploadID, userID)
	if err != nil {
		return nil, ErrUploadNotFound
	}
	chunks, err := s.repo.ListUploadChunkIndexes(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if len(chunks) != sess.TotalChunks {
		return nil, ErrInvalidInput
	}

	// 创建最终视频存储目录
	now := time.Now()
	dateDir := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	relDir := filepath.ToSlash(filepath.Join("videos", "source", dateDir))
	absDir := filepath.Join(storageRoot, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, err
	}

	fileName := uploadID + sess.Ext
	absPath := filepath.Join(absDir, fileName)

	// 创建最终合并后的视频文件
	out, err := os.Create(absPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	chunkDir := filepath.Join(storageRoot, "chunks", uploadID)

	// 按分片序号从小到大合并文件
	for i := 0; i < sess.TotalChunks; i++ {
		b, err := os.ReadFile(filepath.Join(chunkDir, strconv.Itoa(i)+".part"))
		if err != nil {
			return nil, err
		}
		if _, err := out.Write(b); err != nil {
			return nil, err
		}
	}

	publicID := "IV" + randomAlpha(10)
	sourceURL := "/" + filepath.ToSlash(filepath.Join("static", relDir, fileName))

	v, err := s.CreateUploadedVideo(ctx, CreateVideoInput{PublicID: publicID, UserID: userID, CategoryID: sess.CategoryID, Title: sess.Title, Description: sess.Description, SourceVideoURL: sourceURL, FileSizeBytes: sess.TotalSize, Status: 2})
	if err != nil {
		return nil, err
	}

	if err := s.repo.CompleteUploadSession(ctx, uploadID); err != nil {
		return nil, err
	}

	// 删除临时分片目录
	_ = os.RemoveAll(chunkDir)

	// 上传完成后启动异步转码
	s.StartAsyncTranscode(publicID, absPath)

	return &UploadCompleteResult{PublicID: v.PublicID, Status: v.Status, TranscodeStatus: v.TranscodeStatus}, nil
}

// ListHistory 获取用户观看历史
func (s *Service) ListHistory(ctx context.Context, userID uint64, page, pageSize int) ([]VideoListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	key := fmt.Sprintf(videoHistoryFmt, userID, page, pageSize)
	var cached struct {
		List  []VideoListItem `json:"list"`
		Total int64           `json:"total"`
	}
	if s.getJSONCache(ctx, key, &cached) {
		return cached.List, cached.Total, nil
	}
	rows, total, err := s.repo.ListHistory(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	list := toVideoList(rows)
	s.setJSONCache(ctx, key, struct {
		List  []VideoListItem `json:"list"`
		Total int64           `json:"total"`
	}{List: list, Total: total}, videoListCacheTTL)
	return list, total, nil
}

// ListDanmaku 获取视频弹幕列表
// 根据视频公开 ID 查询视频，再查询对应弹幕数据
func (s *Service) ListDanmaku(ctx context.Context, publicID string) ([]DanmakuItem, error) {
	var cached []DanmakuItem
	if s.getJSONCache(ctx, fmt.Sprintf(videoDanmakuFmt, publicID), &cached) {
		return cached, nil
	}

	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	rows, err := s.repo.ListDanmaku(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	out := make([]DanmakuItem, 0, len(rows))
	for _, d := range rows {
		out = append(out, DanmakuItem{ID: d.ID, PublicID: publicID, UserID: d.UserID, Content: d.Content, TimeMs: d.TimeMs, Color: d.Color, Mode: d.Mode, CreatedAt: d.CreatedAt.Format(timeFormat)})
	}
	s.setJSONCache(ctx, fmt.Sprintf(videoDanmakuFmt, publicID), out, videoItemCacheTTL)
	return out, nil
}

// CreateDanmaku 创建弹幕
//
// 校验规则：
// - 用户必须登录
// - 弹幕内容不能为空
// - 弹幕内容不能超过 120 个字符
// - 默认颜色为白色
// - 默认模式为滚动弹幕
func (s *Service) CreateDanmaku(ctx context.Context, in CreateDanmakuInput) (*DanmakuItem, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, in.PublicID)
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	content := strings.TrimSpace(in.Content)
	if in.UserID == 0 || content == "" || len([]rune(content)) > 120 {
		return nil, ErrInvalidInput
	}
	color := in.Color
	if color == "" {
		color = "#ffffff"
	}
	mode := in.Mode
	if mode == "" {
		mode = "scroll"
	}
	d := &VideoDanmaku{VideoID: row.ID, UserID: in.UserID, Content: content, TimeMs: in.TimeMs, Color: color, Mode: mode, Status: 1}
	if err := s.repo.CreateDanmaku(ctx, d); err != nil {
		return nil, err
	}
	if s.redisEnabled() {
		_ = s.rdb.Del(ctx, fmt.Sprintf(videoDanmakuFmt, in.PublicID)).Err()
	}
	return &DanmakuItem{ID: d.ID, PublicID: in.PublicID, UserID: d.UserID, Content: d.Content, TimeMs: d.TimeMs, Color: d.Color, Mode: d.Mode, CreatedAt: d.CreatedAt.Format(timeFormat)}, nil
}

// allowedVideoExt 判断是否为允许上传的视频格式
func allowedVideoExt(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".webm":
		return true
	default:
		return false
	}
}

// randomSuffix 生成上传 ID 后缀
func randomSuffix() string { return strconv.FormatInt(time.Now().UnixNano()%1e6, 36) }

// randomAlpha 生成指定长度的随机字母数字字符串
func randomAlpha(n int) string {
	const a = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := strings.Builder{}
	seed := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		idx := int(math.Abs(float64((seed + int64(i)*7919) % int64(len(a)))))
		b.WriteByte(a[idx])
	}
	return b.String()
}
