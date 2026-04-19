package video

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrVideoNotFound = errors.New("video not found")
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
