package video

import (
	"IslaMemory/BackEnd/internal/comment"
	"IslaMemory/BackEnd/internal/favorite"
	"IslaMemory/BackEnd/internal/like"
	"context"
	"time"

	"gorm.io/gorm"
)

// Repository 只负责数据库操作
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetCategoryIDBySlug(ctx context.Context, slug string) (uint64, error) {
	var row struct {
		ID uint64 `gorm:"column:id"`
	}
	err := r.db.WithContext(ctx).
		Table("categories").
		Select("id").
		Where("slug = ?", slug).
		Where("status = ?", 1).
		Where("deleted_at IS NULL").
		First(&row).Error
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateVideo 创建视频记录
func (r *Repository) CreateVideo(ctx context.Context, v *Video) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// ListParams 列表查询参数
type ListParams struct {
	Page         int
	PageSize     int
	CategorySlug string
	Keyword      string
	Sort         string // latest / popular
}

// VideoRow 是列表/详情查询时返回的聚合结果
// 这里额外带上分类名和分类 slug，方便前端直接展示
type VideoRow struct {
	ID                uint64     `gorm:"column:id"`
	PublicID          string     `gorm:"column:public_id"`
	UserID            uint64     `gorm:"column:user_id"`
	CategoryID        uint64     `gorm:"column:category_id"`
	Title             string     `gorm:"column:title"`
	Description       string     `gorm:"column:description"`
	SourceVideoURL    string     `gorm:"column:source_video_url"`
	PlaybackURL       string     `gorm:"column:playback_url"`
	PlaybackType      uint8      `gorm:"column:playback_type"`
	TranscodeStatus   uint8      `gorm:"column:transcode_status"`
	TranscodeProgress uint32     `gorm:"column:transcode_progress"`
	Status            uint8      `gorm:"column:status"`
	TranscodeError    string     `gorm:"column:transcode_error"`
	CoverURL          string     `gorm:"column:cover_url"`
	DurationSeconds   uint32     `gorm:"column:duration_seconds"`
	Width             uint32     `gorm:"column:width"`
	Height            uint32     `gorm:"column:height"`
	FileSizeBytes     uint64     `gorm:"column:file_size_bytes"`
	PlayCount         uint64     `gorm:"column:play_count"`
	LikeCount         uint64     `gorm:"column:like_count"`
	FavoriteCount     uint64     `gorm:"column:favorite_count"`
	CommentCount      uint64     `gorm:"column:comment_count"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	PublishedAt       *time.Time `gorm:"column:published_at"`
	ReviewedAt        *time.Time `gorm:"column:reviewed_at"`

	CategoryName string `gorm:"column:category_name"`
	CategorySlug string `gorm:"column:category_slug"`
}

// basePublicQuery 构建公开视频查询基础条件
// 只返回：未删除、已发布、分类启用的内容
func (r *Repository) basePublicQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("videos AS v").
		Joins("JOIN categories AS c ON c.id = v.category_id").
		Where("v.deleted_at IS NULL").
		Where("c.deleted_at IS NULL").
		Where("c.status = ?", 1).
		Where("v.status = ?", 2)
}

// ListPublicVideos 查询公开视频列表
func (r *Repository) ListPublicVideos(ctx context.Context, params ListParams) ([]VideoRow, int64, error) {
	base := r.basePublicQuery(ctx)

	// 分类筛选
	if params.CategorySlug != "" {
		base = base.Where("c.slug = ?", params.CategorySlug)
	}

	// 关键词搜索：标题和简介都支持
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		base = base.Where("(v.title LIKE ? OR v.description LIKE ?)", like, like)
	}

	// 先查总数
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 再查列表
	query := base.Session(&gorm.Session{}).
		Select(`
			v.id,
			v.public_id,
			v.user_id,
			v.category_id,
			v.title,
			v.description,
			v.source_video_url,
			v.playback_url,
			v.playback_type,
			v.transcode_status,
			v.transcode_progress,
			v.status,
			v.transcode_error,
			v.cover_url,
			v.duration_seconds,
			v.width,
			v.height,
			v.file_size_bytes,
			v.play_count,
			v.like_count,
			v.favorite_count,
			v.comment_count,
			v.created_at,
			v.updated_at,
			v.published_at,
			v.reviewed_at,
			c.name AS category_name,
			c.slug AS category_slug
		`)

	// 排序规则
	switch params.Sort {
	case "popular":
		query = query.Order("v.play_count DESC, v.id DESC")
	default:
		// 默认按最新发布/创建排序
		query = query.Order("COALESCE(v.published_at, v.created_at) DESC, v.id DESC")
	}

	offset := (params.Page - 1) * params.PageSize

	var rows []VideoRow
	if err := query.Limit(params.PageSize).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// GetPublicVideoByPublicID 查询公开视频详情
func (r *Repository) GetPublicVideoByPublicID(ctx context.Context, publicID string) (*VideoRow, error) {
	query := r.basePublicQuery(ctx).
		Select(`
			v.id,
			v.public_id,
			v.user_id,
			v.category_id,
			v.title,
			v.description,
			v.source_video_url,
			v.playback_url,
			v.playback_type,
			v.transcode_status,
			v.transcode_progress,
			v.status,
			v.transcode_error,
			v.cover_url,
			v.duration_seconds,
			v.width,
			v.height,
			v.file_size_bytes,
			v.play_count,
			v.like_count,
			v.favorite_count,
			v.comment_count,
			v.created_at,
			v.updated_at,
			v.published_at,
			v.reviewed_at,
			c.name AS category_name,
			c.slug AS category_slug
		`).
		Where("v.public_id = ?", publicID)

	var row VideoRow
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}

	return &row, nil
}

// IncreasePlayCount 播放量 +1
// 第一版先直接累加，后面你接 Redis 去重时再在 service 层加逻辑
func (r *Repository) IncreasePlayCount(ctx context.Context, publicID string) error {
	return r.db.WithContext(ctx).
		Model(&Video{}).
		Where("public_id = ?", publicID).
		Where("deleted_at IS NULL").
		UpdateColumn("play_count", gorm.Expr("play_count + 1")).
		Error
}

func (r *Repository) IsVideoLikedByUser(ctx context.Context, videoID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_likes").
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) IsVideoFavoritedByUser(ctx context.Context, videoID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_favorites").
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// LikeVideo 视频点赞
func (r *Repository) LikeVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	liked := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&like.VideoLike{
			VideoID: videoID,
			UserID:  userID,
		}).FirstOrCreate(&like.VideoLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		liked = true
		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).
			Error
	})
	return liked, err
}

// UnlikeVideo 取消点赞
func (r *Repository) UnlikeVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	unliked := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("video_id = ? AND user_id = ?", videoID, userID).Delete(&like.VideoLike{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		unliked = true
		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("IF(like_count > 0, like_count - 1, 0)")).
			Error
	})
	return unliked, err
}

// FavoriteVideo 收藏视频
func (r *Repository) FavoriteVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	favorited := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where(&favorite.VideoFavorite{
			VideoID: videoID,
			UserID:  userID,
		}).FirstOrCreate(&favorite.VideoFavorite{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		favorited = true
		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).
			Error
	})
	return favorited, err
}

// UnfavoriteVideo 取消收藏
func (r *Repository) UnfavoriteVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	unfavorited := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("video_id = ? AND user_id = ?", videoID, userID).Delete(&favorite.VideoFavorite{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		unfavorited = true
		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("favorite_count", gorm.Expr("IF(favorite_count > 0, favorite_count - 1, 0)")).
			Error
	})
	return unfavorited, err
}

// GetActiveCommentByID 视频评论
func (r *Repository) GetActiveCommentByID(ctx context.Context, commentID uint64) (*comment.VideoComment, error) {
	var cmt comment.VideoComment
	err := r.db.WithContext(ctx).
		Where("id = ?", commentID).
		Where("deleted_at IS NULL").
		Where("status = ?", 1).
		First(&cmt).Error
	if err != nil {
		return nil, err
	}
	return &cmt, nil
}

type CreateCommentParams struct {
	VideoID       uint64
	UserID        uint64
	ParentID      *uint64
	RootID        *uint64
	ReplyToUserID *uint64
	Content       string
}

// CreateComment 创建评论
func (r *Repository) CreateComment(ctx context.Context, params CreateCommentParams) (*comment.VideoComment, error) {
	cmt := &comment.VideoComment{
		VideoID:       params.VideoID,
		UserID:        params.UserID,
		ParentID:      params.ParentID,
		RootID:        params.RootID,
		ReplyToUserID: params.ReplyToUserID,
		Content:       params.Content,
		Status:        1,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cmt).Error; err != nil {
			return err
		}
		return tx.Model(&Video{}).
			Where("id = ?", params.VideoID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).
			Error
	})
	if err != nil {
		return nil, err
	}
	return cmt, nil
}

type CommentRow struct {
	ID              uint64    `gorm:"column:id"`
	VideoID         uint64    `gorm:"column:video_id"`
	UserID          uint64    `gorm:"column:user_id"`
	ParentID        *uint64   `gorm:"column:parent_id"`
	RootID          *uint64   `gorm:"column:root_id"`
	ReplyToUserID   *uint64   `gorm:"column:reply_to_user_id"`
	Content         string    `gorm:"column:content"`
	LikeCount       uint64    `gorm:"column:like_count"`
	Status          uint8     `gorm:"column:status"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
	Username        string    `gorm:"column:username"`
	AvatarURL       string    `gorm:"column:avatar_url"`
	ReplyToUsername *string   `gorm:"column:reply_to_username"`
	ReplyToAvatar   *string   `gorm:"column:reply_to_avatar_url"`
}

// ListCommentsByVideoID 视频对应评论
func (r *Repository) ListCommentsByVideoID(ctx context.Context, videoID uint64) ([]CommentRow, error) {
	var rows []CommentRow
	err := r.db.WithContext(ctx).
		Table("video_comments AS vc").
		Joins("JOIN users AS u ON u.id = vc.user_id").
		Joins("LEFT JOIN users AS ru ON ru.id = vc.reply_to_user_id").
		Where("vc.video_id = ?", videoID).
		Where("vc.deleted_at IS NULL").
		Where("vc.status = ?", 1).
		Order("vc.created_at ASC, vc.id ASC").
		Select(`
			vc.id,
			vc.video_id,
			vc.user_id,
			vc.parent_id,
			vc.root_id,
			vc.reply_to_user_id,
			vc.content,
			vc.like_count,
			vc.status,
			vc.created_at,
			vc.updated_at,
			u.username,
			u.avatar_url,
			ru.username AS reply_to_username,
			ru.avatar_url AS reply_to_avatar_url
		`).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
