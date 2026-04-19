package video

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrVideoNotFound = errors.New("video not found")
	ErrInvalidInput  = errors.New("invalid video input")
)

// Service 负责 video 业务逻辑
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListInput 列表接口输入
type ListInput struct {
	Page         int
	PageSize     int
	CategorySlug string
	Keyword      string
	Sort         string
}

// VideoListItem 列表页返回给前端的结构
type VideoListItem struct {
	PublicID        string  `json:"public_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CoverURL        string  `json:"cover_url"`
	DurationSeconds uint32  `json:"duration_seconds"`
	Width           uint32  `json:"width"`
	Height          uint32  `json:"height"`
	PlayCount       uint64  `json:"play_count"`
	LikeCount       uint64  `json:"like_count"`
	FavoriteCount   uint64  `json:"favorite_count"`
	CommentCount    uint64  `json:"comment_count"`
	CategoryID      uint64  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	CategorySlug    string  `json:"category_slug"`
	PublishedAt     *string `json:"published_at"`
	CreatedAt       string  `json:"created_at"`
}

// VideoDetail 视频详情返回结构
type VideoDetail struct {
	PublicID        string  `json:"public_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CoverURL        string  `json:"cover_url"`
	DurationSeconds uint32  `json:"duration_seconds"`
	Width           uint32  `json:"width"`
	Height          uint32  `json:"height"`
	PlayCount       uint64  `json:"play_count"`
	LikeCount       uint64  `json:"like_count"`
	FavoriteCount   uint64  `json:"favorite_count"`
	CommentCount    uint64  `json:"comment_count"`
	CategoryID      uint64  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	CategorySlug    string  `json:"category_slug"`
	PlaybackType    uint8   `json:"playback_type"`
	PlaybackURL     string  `json:"playback_url"`
	PublishedAt     *string `json:"published_at"`
	CreatedAt       string  `json:"created_at"`
}

// CreateVideoInput 创建视频入参
type CreateVideoInput struct {
	PublicID       string
	UserID         uint64
	CategoryID     uint64
	Title          string
	Description    string
	SourceVideoURL string
	FileSizeBytes  uint64
	Status         uint8
}

var (
	ErrCommentNotFound     = errors.New("comment not found")
	ErrInvalidCommentInput = errors.New("invalid comment input")
)

// 评论结构体
type CommentItem struct {
	ID              uint64        `json:"id"`
	VideoID         uint64        `json:"video_id"`
	UserID          uint64        `json:"user_id"`
	Username        string        `json:"username"`
	AvatarURL       string        `json:"avatar_url"`
	ParentID        *uint64       `json:"parent_id"`
	RootID          *uint64       `json:"root_id"`
	ReplyToUserID   *uint64       `json:"reply_to_user_id"`
	ReplyToUsername *string       `json:"reply_to_username"`
	Content         string        `json:"content"`
	LikeCount       uint64        `json:"like_count"`
	CreatedAt       string        `json:"created_at"`
	UpdatedAt       string        `json:"updated_at"`
	Replies         []CommentItem `json:"replies"`
}

type CreateCommentInput struct {
	PublicID      string
	UserID        uint64
	ParentID      *uint64
	ReplyToUserID *uint64
	Content       string
}

// normalizeListInput 统一处理分页和排序默认值
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
		in.Sort = "latest"
	}
	return in
}

// ListPublicVideos 获取公开视频列表
func (s *Service) ListPublicVideos(ctx context.Context, in ListInput) ([]VideoListItem, int64, error) {
	in = s.normalizeListInput(in)

	rows, total, err := s.repo.ListPublicVideos(ctx, ListParams{
		Page:         in.Page,
		PageSize:     in.PageSize,
		CategorySlug: in.CategorySlug,
		Keyword:      in.Keyword,
		Sort:         in.Sort,
	})
	if err != nil {
		return nil, 0, err
	}

	list := make([]VideoListItem, 0, len(rows))
	for _, row := range rows {
		item := VideoListItem{
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
			CreatedAt:       row.CreatedAt.Format(timeFormat),
		}
		if row.PublishedAt != nil {
			t := row.PublishedAt.Format(timeFormat)
			item.PublishedAt = &t
		}
		list = append(list, item)
	}

	return list, total, nil
}

// GetPublicVideoDetail 获取视频详情
func (s *Service) GetPublicVideoDetail(ctx context.Context, publicID string) (*VideoDetail, error) {
	row, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	// TODO: 第一版播放地址策略：
	// 1. 如果 playback_url 有值，就优先用它
	// 2. 否则退回 source_video_url
	playURL := row.PlaybackURL
	if playURL == "" {
		playURL = row.SourceVideoURL
	}

	detail := &VideoDetail{
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
		PlaybackType:    row.PlaybackType,
		PlaybackURL:     playURL,
		CreatedAt:       row.CreatedAt.Format(timeFormat),
	}
	if row.PublishedAt != nil {
		t := row.PublishedAt.Format(timeFormat)
		detail.PublishedAt = &t
	}

	return detail, nil
}

// IncreasePlayCount 播放量 +1
// TODO: 第一版先直接加；后面你接 Redis 后，可以在这里加播放去重逻辑
func (s *Service) IncreasePlayCount(ctx context.Context, publicID string) error {
	return s.repo.IncreasePlayCount(ctx, publicID)
}

const timeFormat = "2006-01-02 15:04:05"

// CreateUploadedVideo 创建上传视频记录
// 第一版本地上传：
// - playback_url 先留空
// - playback_type=0 表示直接播放原文件
// - 状态先默认已发布（2），便于快速联调
func (s *Service) CreateUploadedVideo(ctx context.Context, in CreateVideoInput) (*Video, error) {
	if in.PublicID == "" || in.UserID == 0 || in.CategoryID == 0 || in.Title == "" || in.SourceVideoURL == "" {
		return nil, ErrInvalidInput
	}

	status := in.Status
	if status == 0 {
		status = 2
	}

	video := &Video{
		PublicID:       in.PublicID,
		UserID:         in.UserID,
		CategoryID:     in.CategoryID,
		Title:          in.Title,
		Description:    in.Description,
		SourceVideoURL: in.SourceVideoURL,
		PlaybackType:   0,
		Status:         status,
		FileSizeBytes:  in.FileSizeBytes,
	}

	if err := s.repo.CreateVideo(ctx, video); err != nil {
		return nil, err
	}

	return video, nil
}

// LikeVideo 视频点赞
func (s *Service) LikeVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	video, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	return s.repo.LikeVideo(ctx, video.ID, userID)
}

// UnlikeVideo 取消点赞
func (s *Service) UnlikeVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	video, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	return s.repo.UnlikeVideo(ctx, video.ID, userID)
}

// FavoriteVideo 收藏视频
func (s *Service) FavoriteVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	video, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	return s.repo.FavoriteVideo(ctx, video.ID, userID)
}

// UnfavoriteVideo 取消收藏
func (s *Service) UnfavoriteVideo(ctx context.Context, publicID string, userID uint64) (bool, error) {
	video, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrVideoNotFound
		}
		return false, err
	}
	return s.repo.UnfavoriteVideo(ctx, video.ID, userID)
}

// CreateComment 创建评论
func (s *Service) CreateComment(ctx context.Context, in CreateCommentInput) (*CommentItem, error) {
	content := strings.TrimSpace(in.Content)
	if in.PublicID == "" || in.UserID == 0 || content == "" {
		return nil, ErrInvalidCommentInput
	}

	video, err := s.repo.GetPublicVideoByPublicID(ctx, in.PublicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	var rootID *uint64
	if in.ParentID != nil {
		parent, err := s.repo.GetActiveCommentByID(ctx, *in.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCommentNotFound
			}
			return nil, err
		}
		if parent.VideoID != video.ID {
			return nil, ErrInvalidCommentInput
		}

		if parent.RootID == nil {
			id := parent.ID
			rootID = &id
		} else {
			rootID = parent.RootID
		}

		if in.ReplyToUserID == nil {
			replyUID := parent.UserID
			in.ReplyToUserID = &replyUID
		}
	}

	cmt, err := s.repo.CreateComment(ctx, CreateCommentParams{
		VideoID:       video.ID,
		UserID:        in.UserID,
		ParentID:      in.ParentID,
		RootID:        rootID,
		ReplyToUserID: in.ReplyToUserID,
		Content:       content,
	})
	if err != nil {
		return nil, err
	}

	item := &CommentItem{
		ID:            cmt.ID,
		VideoID:       cmt.VideoID,
		UserID:        cmt.UserID,
		ParentID:      cmt.ParentID,
		RootID:        cmt.RootID,
		ReplyToUserID: cmt.ReplyToUserID,
		Content:       cmt.Content,
		LikeCount:     cmt.LikeCount,
		CreatedAt:     cmt.CreatedAt.Format(timeFormat),
		UpdatedAt:     cmt.UpdatedAt.Format(timeFormat),
		Replies:       make([]CommentItem, 0),
	}
	return item, nil
}

// ListComments 评论列表
func (s *Service) ListComments(ctx context.Context, publicID string) ([]CommentItem, error) {
	video, err := s.repo.GetPublicVideoByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	rows, err := s.repo.ListCommentsByVideoID(ctx, video.ID)
	if err != nil {
		return nil, err
	}

	rootList := make([]CommentItem, 0)
	rootMap := make(map[uint64]*CommentItem)
	replyBucket := make(map[uint64][]CommentItem)

	for _, row := range rows {
		item := CommentItem{
			ID:              row.ID,
			VideoID:         row.VideoID,
			UserID:          row.UserID,
			Username:        row.Username,
			AvatarURL:       row.AvatarURL,
			ParentID:        row.ParentID,
			RootID:          row.RootID,
			ReplyToUserID:   row.ReplyToUserID,
			ReplyToUsername: row.ReplyToUsername,
			Content:         row.Content,
			LikeCount:       row.LikeCount,
			CreatedAt:       row.CreatedAt.Format(timeFormat),
			UpdatedAt:       row.UpdatedAt.Format(timeFormat),
			Replies:         make([]CommentItem, 0),
		}

		if row.ParentID == nil {
			rootList = append(rootList, item)
			rootMap[item.ID] = &rootList[len(rootList)-1]
			continue
		}

		root := row.RootID
		if root == nil {
			root = row.ParentID
		}
		if root != nil {
			replyBucket[*root] = append(replyBucket[*root], item)
		}
	}

	for rootID, replies := range replyBucket {
		if root, ok := rootMap[rootID]; ok {
			root.Replies = append(root.Replies, replies...)
		}
	}

	return rootList, nil
}
