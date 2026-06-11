package video

import (
	"IslaMemory/BackEnd/internal/comment"
	"IslaMemory/BackEnd/internal/favorite"
	"IslaMemory/BackEnd/internal/like"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository 视频模块数据访问层
type Repository struct{ db *gorm.DB }

// NewRepository 创建视频 Repository
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// GetCategoryIDBySlug 根据分类 slug 获取分类 ID
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

// ListParams 视频列表查询参数
type ListParams struct {
	Page, PageSize              int
	CategorySlug, Keyword, Sort string
}

// VideoRow 视频查询结果
// 用于返回视频信息、分类信息和作者信息
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
	CategoryName      string     `gorm:"column:category_name"`
	CategorySlug      string     `gorm:"column:category_slug"`
	Username          string     `gorm:"column:username"`
}

// basePublicQuery 构建公开视频查询基础条件
func (r *Repository) basePublicQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("videos AS v").
		Joins("JOIN categories AS c ON c.id = v.category_id").
		Joins("JOIN users AS u ON u.id = v.user_id").
		Where("v.deleted_at IS NULL").
		Where("c.deleted_at IS NULL").
		Where("c.status = ?", 1).
		Where("v.status = ?", 2)
}

// selectVideoRows 统一选择视频查询字段
func selectVideoRows(q *gorm.DB) *gorm.DB {
	return q.Select(`
v.id, v.public_id, v.user_id, v.category_id, v.title, v.description,
v.source_video_url, v.playback_url, v.playback_type, v.transcode_status,
v.transcode_progress, v.status, v.transcode_error, v.cover_url,
v.duration_seconds, v.width, v.height, v.file_size_bytes,
v.play_count, v.like_count, v.favorite_count, v.comment_count,
v.created_at, v.updated_at, v.published_at, v.reviewed_at,
c.name AS category_name, c.slug AS category_slug, u.username AS username`)
}

// ListPublicVideos 获取公开视频列表
func (r *Repository) ListPublicVideos(ctx context.Context, params ListParams) ([]VideoRow, int64, error) {
	base := r.basePublicQuery(ctx)

	if params.CategorySlug != "" {
		base = base.Where("c.slug = ?", params.CategorySlug)
	}

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		base = base.Where("(v.title LIKE ? OR v.description LIKE ?)", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := selectVideoRows(base.Session(&gorm.Session{}))

	switch params.Sort {
	case "popular":
		query = query.Order("v.play_count DESC, v.like_count DESC, v.favorite_count DESC, v.id DESC")
	case "latest":
		query = query.Order("COALESCE(v.published_at, v.created_at) DESC, v.id DESC")
	default:
		query = query.Order("(v.play_count + v.like_count * 5 + v.favorite_count * 8 + v.comment_count * 3 + GREATEST(0, 1000 - TIMESTAMPDIFF(HOUR, COALESCE(v.published_at, v.created_at), NOW())) ) DESC, v.id DESC")
	}

	var rows []VideoRow
	if err := query.Limit(params.PageSize).Offset((params.Page - 1) * params.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// GetPublicVideoByPublicID 根据公开 ID 获取视频详情
func (r *Repository) GetPublicVideoByPublicID(ctx context.Context, publicID string) (*VideoRow, error) {
	var row VideoRow

	err := selectVideoRows(r.basePublicQuery(ctx)).
		Where("v.public_id = ?", publicID).
		First(&row).Error

	if err != nil {
		return nil, err
	}

	return &row, nil
}

// UpdateVideoTranscode 更新视频转码信息
func (r *Repository) UpdateVideoTranscode(ctx context.Context, publicID string, values map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&Video{}).
		Where("public_id = ?", publicID).
		Updates(values).Error
}

// CreatePlayAndHistory 创建播放记录并更新观看历史
func (r *Repository) CreatePlayAndHistory(ctx context.Context, videoID uint64, userID *uint64, viewerKey, viewedOn string, progress uint32) (bool, error) {
	counted := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		event := &VideoPlayEvent{
			VideoID:   videoID,
			UserID:    userID,
			ViewerKey: viewerKey,
			ViewedOn:  viewedOn,
		}

		// 同一视频、同一观看者、同一天只记录一次播放量
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			counted = true

			// 新播放事件创建成功后，播放量 +1
			if err := tx.Model(&Video{}).
				Where("id = ?", videoID).
				UpdateColumn("play_count", gorm.Expr("play_count + 1")).Error; err != nil {
				return err
			}
		}

		// 登录用户需要更新观看历史
		if userID != nil && *userID > 0 {
			h := &VideoWatchHistory{
				VideoID:         videoID,
				UserID:          *userID,
				ProgressSeconds: progress,
				LastWatchedAt:   time.Now(),
			}

			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "video_id"},
				},
				DoUpdates: clause.Assignments(map[string]any{
					"progress_seconds": progress,
					"last_watched_at":  time.Now(),
					"updated_at":       time.Now(),
				}),
			}).Create(h).Error
		}

		return nil
	})

	return counted, err
}

// IsVideoLikedByUser 判断用户是否点赞过视频
func (r *Repository) IsVideoLikedByUser(ctx context.Context, videoID, userID uint64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("video_likes").
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Count(&count).Error

	return count > 0, err
}

// IsVideoFavoritedByUser 判断用户是否收藏过视频
func (r *Repository) IsVideoFavoritedByUser(ctx context.Context, videoID, userID uint64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("video_favorites").
		Where("video_id = ? AND user_id = ?", videoID, userID).
		Count(&count).Error

	return count > 0, err
}

// LikeVideo 点赞视频
func (r *Repository) LikeVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	liked := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&like.VideoLike{VideoID: videoID, UserID: userID})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return nil
		}

		liked = true

		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	})

	return liked, err
}

// UnlikeVideo 取消点赞
func (r *Repository) UnlikeVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	unliked := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("video_id = ? AND user_id = ?", videoID, userID).
			Delete(&like.VideoLike{})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return nil
		}

		unliked = true

		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("IF(like_count > 0, like_count - 1, 0)")).Error
	})

	return unliked, err
}

// FavoriteVideo 收藏视频
func (r *Repository) FavoriteVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	favorited := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&favorite.VideoFavorite{VideoID: videoID, UserID: userID})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return nil
		}

		favorited = true

		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error
	})

	return favorited, err
}

// UnfavoriteVideo 取消收藏
func (r *Repository) UnfavoriteVideo(ctx context.Context, videoID, userID uint64) (bool, error) {
	unfavorited := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("video_id = ? AND user_id = ?", videoID, userID).
			Delete(&favorite.VideoFavorite{})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return nil
		}

		unfavorited = true

		return tx.Model(&Video{}).
			Where("id = ?", videoID).
			UpdateColumn("favorite_count", gorm.Expr("IF(favorite_count > 0, favorite_count - 1, 0)")).Error
	})

	return unfavorited, err
}

// GetActiveCommentByID 获取有效评论
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

// CreateCommentParams 创建评论参数
type CreateCommentParams struct {
	VideoID, UserID                 uint64
	ParentID, RootID, ReplyToUserID *uint64
	Content                         string
}

// CreateComment 创建评论并增加视频评论数
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
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})

	return cmt, err
}

// CommentRow 评论查询结果
type CommentRow struct {
	ID, VideoID, UserID             uint64
	ParentID, RootID, ReplyToUserID *uint64
	Content                         string
	LikeCount                       uint64
	Status                          uint8
	CreatedAt, UpdatedAt            time.Time
	Username, AvatarURL             string
	ReplyToUsername, ReplyToAvatar  *string
}

// ListCommentsByVideoID 获取视频评论列表
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
vc.id, vc.video_id, vc.user_id, vc.parent_id, vc.root_id,
vc.reply_to_user_id, vc.content, vc.like_count, vc.status,
vc.created_at, vc.updated_at,
u.username, u.avatar_url,
ru.username AS reply_to_username,
ru.avatar_url AS reply_to_avatar_url`).
		Find(&rows).Error

	return rows, err
}

// CreateUploadSession 创建分片上传会话
func (r *Repository) CreateUploadSession(ctx context.Context, s *UploadSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetUploadSession 获取分片上传会话
func (r *Repository) GetUploadSession(ctx context.Context, uploadID string, userID uint64) (*UploadSession, error) {
	var s UploadSession

	err := r.db.WithContext(ctx).
		Where("upload_id = ? AND user_id = ?", uploadID, userID).
		First(&s).Error

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// AddUploadChunk 添加上传分片记录
func (r *Repository) AddUploadChunk(ctx context.Context, uploadID string, idx int, size uint64) (bool, error) {
	added := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&UploadChunk{UploadID: uploadID, Index: idx, Size: size})

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected > 0 {
			added = true

			return tx.Model(&UploadSession{}).
				Where("upload_id = ?", uploadID).
				UpdateColumn("uploaded_chunks", gorm.Expr("uploaded_chunks + 1")).Error
		}

		return nil
	})

	return added, err
}

// ListUploadChunkIndexes 获取已上传的分片序号列表
func (r *Repository) ListUploadChunkIndexes(ctx context.Context, uploadID string) ([]int, error) {
	var chunks []UploadChunk

	if err := r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Find(&chunks).Error; err != nil {
		return nil, err
	}

	out := make([]int, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, c.Index)
	}

	return out, nil
}

// CompleteUploadSession 标记分片上传会话为完成
func (r *Repository) CompleteUploadSession(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Model(&UploadSession{}).
		Where("upload_id = ?", uploadID).
		Update("status", 1).Error
}

// ListHistory 获取用户观看历史
func (r *Repository) ListHistory(ctx context.Context, userID uint64, page, pageSize int) ([]VideoRow, int64, error) {
	base := r.db.WithContext(ctx).
		Table("video_watch_histories wh").
		Joins("JOIN videos v ON v.id = wh.video_id").
		Joins("JOIN categories c ON c.id = v.category_id").
		Joins("JOIN users u ON u.id = v.user_id").
		Where("wh.user_id = ?", userID).
		Where("v.deleted_at IS NULL").
		Where("v.status = ?", 2)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []VideoRow
	err := selectVideoRows(base).
		Order("wh.last_watched_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&rows).Error

	return rows, total, err
}

// CreateDanmaku 创建弹幕
func (r *Repository) CreateDanmaku(ctx context.Context, d *VideoDanmaku) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// ListDanmaku 获取视频弹幕列表
func (r *Repository) ListDanmaku(ctx context.Context, videoID uint64) ([]VideoDanmaku, error) {
	var rows []VideoDanmaku

	err := r.db.WithContext(ctx).
		Where("video_id = ? AND status = ?", videoID, 1).
		Order("time_ms ASC, id ASC").
		Find(&rows).Error

	return rows, err
}

// IsNotFound 判断错误是否为记录不存在
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// ListPublicVideoPublicIDs 获取公开视频公开 ID，用于初始化 Redis Bitmap 布隆过滤器。
func (r *Repository) ListPublicVideoPublicIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := r.basePublicQuery(ctx).Pluck("v.public_id", &ids).Error
	return ids, err
}
