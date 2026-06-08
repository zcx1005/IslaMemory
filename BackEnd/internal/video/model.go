package video

import (
	"time"

	"gorm.io/gorm"
)

// Video 视频信息模型
// 对应数据库 videos 表
//
// 存储视频基础信息、转码信息、统计数据、审核状态等内容
type Video struct {
	// 主键ID
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 对外公开的视频ID
	// 用于前端访问，例如：IV2Qw8ErT5Yu
	PublicID string `gorm:"size:32;not null;uniqueIndex" json:"public_id"`

	// 上传用户ID
	UserID uint64 `gorm:"not null;index" json:"user_id"`

	// 视频分类ID
	CategoryID uint64 `gorm:"not null;index" json:"category_id"`

	// 视频标题
	Title string `gorm:"size:200;not null" json:"title"`

	// 视频描述
	Description string `gorm:"type:text" json:"description"`

	// 原始视频文件地址
	// 不返回给前端
	SourceVideoURL string `gorm:"size:255;not null" json:"-"`

	// 播放地址（HLS、MP4等）
	// 不直接暴露给前端
	PlaybackURL string `gorm:"size:255" json:"-"`

	// 播放类型
	// 0=原始文件
	// 1=HLS
	// 2=Dash
	PlaybackType uint8 `gorm:"not null;default:0" json:"playback_type"`

	// 转码状态
	// 0=待转码
	// 1=转码中
	// 2=转码成功
	// 3=转码失败
	TranscodeStatus uint8 `gorm:"not null;default:0" json:"transcode_status"`

	// 转码进度（百分比）
	TranscodeProgress uint32 `gorm:"not null;default:0" json:"transcode_progress"`

	// 视频状态
	// 0=待审核
	// 1=审核拒绝
	// 2=已发布
	// 3=已下架
	Status uint8 `gorm:"not null;default:0" json:"status"`

	// 转码失败原因
	TranscodeError string `gorm:"size:500" json:"-"`

	// 视频封面地址
	CoverURL string `gorm:"size:255" json:"cover_url"`

	// 视频时长（秒）
	DurationSeconds uint32 `gorm:"not null;default:0" json:"duration_seconds"`

	// 视频宽度
	Width uint32 `gorm:"not null;default:0" json:"width"`

	// 视频高度
	Height uint32 `gorm:"not null;default:0" json:"height"`

	// 文件大小（字节）
	FileSizeBytes uint64 `gorm:"not null;default:0" json:"file_size_bytes"`

	// 播放次数
	PlayCount uint64 `gorm:"not null;default:0" json:"play_count"`

	// 点赞数量
	LikeCount uint64 `gorm:"not null;default:0" json:"like_count"`

	// 收藏数量
	FavoriteCount uint64 `gorm:"not null;default:0" json:"favorite_count"`

	// 评论数量
	CommentCount uint64 `gorm:"not null;default:0" json:"comment_count"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// 软删除字段
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 删除操作人ID
	DeletedBy *uint64 `gorm:"index" json:"-"`

	// 删除原因
	DeleteReason string `gorm:"size:255" json:"-"`

	// 发布时间
	PublishedAt *time.Time `json:"published_at"`

	// 审核时间
	ReviewedAt *time.Time `json:"reviewed_at"`
}

// TableName 指定表名
func (Video) TableName() string {
	return "videos"
}

// UploadSession 分片上传会话
// 对应表：video_upload_sessions
//
// 用于记录大文件上传过程
type UploadSession struct {
	// 主键ID
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 上传任务唯一ID
	UploadID string `gorm:"size:40;not null;uniqueIndex" json:"upload_id"`

	// 上传用户ID
	UserID uint64 `gorm:"not null;index" json:"user_id"`

	// 视频标题
	Title string `gorm:"size:200;not null" json:"title"`

	// 视频描述
	Description string `gorm:"type:text" json:"description"`

	// 分类ID
	CategoryID uint64 `gorm:"not null;index" json:"category_id"`

	// 原始文件名
	Filename string `gorm:"size:255;not null" json:"filename"`

	// 文件扩展名
	Ext string `gorm:"size:16;not null" json:"ext"`

	// 文件总大小
	TotalSize uint64 `gorm:"not null" json:"total_size"`

	// 每个分片大小
	ChunkSize uint64 `gorm:"not null" json:"chunk_size"`

	// 总分片数量
	TotalChunks int `gorm:"not null" json:"total_chunks"`

	// 已上传分片数量
	UploadedChunks int `gorm:"not null;default:0" json:"uploaded_chunks"`

	// 上传状态
	// 0=上传中
	// 1=上传完成
	Status uint8 `gorm:"not null;default:0;index" json:"status"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (UploadSession) TableName() string {
	return "video_upload_sessions"
}

// UploadChunk 上传分片记录
// 对应表：video_upload_chunks
//
// 用于记录每个分片是否上传成功
type UploadChunk struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 上传任务ID
	UploadID string `gorm:"size:40;not null;uniqueIndex:uk_upload_chunk" json:"upload_id"`

	// 分片序号
	Index int `gorm:"not null;uniqueIndex:uk_upload_chunk" json:"index"`

	// 分片大小
	Size uint64 `gorm:"not null" json:"size"`

	CreatedAt time.Time
}

// TableName 指定表名
func (UploadChunk) TableName() string {
	return "video_upload_chunks"
}

// VideoPlayEvent 视频播放事件
// 对应表：video_play_events
//
// 用于防刷播放量统计
type VideoPlayEvent struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 视频ID
	VideoID uint64 `gorm:"not null;uniqueIndex:uk_video_viewer_day;index" json:"video_id"`

	// 登录用户ID
	// 游客为nil
	UserID *uint64 `gorm:"index" json:"user_id"`

	// 观看者唯一标识
	// 登录用户：u:123
	// 游客：a:hash
	ViewerKey string `gorm:"size:80;not null;uniqueIndex:uk_video_viewer_day" json:"viewer_key"`

	// 观看日期
	// 例如：2026-06-08
	ViewedOn string `gorm:"size:10;not null;uniqueIndex:uk_video_viewer_day" json:"viewed_on"`

	CreatedAt time.Time
}

// TableName 指定表名
func (VideoPlayEvent) TableName() string {
	return "video_play_events"
}

// VideoWatchHistory 用户观看历史
// 对应表：video_watch_histories
//
// 记录用户观看进度
type VideoWatchHistory struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 视频ID
	VideoID uint64 `gorm:"not null;uniqueIndex:uk_history_user_video;index" json:"video_id"`

	// 用户ID
	UserID uint64 `gorm:"not null;uniqueIndex:uk_history_user_video;index" json:"user_id"`

	// 当前观看进度（秒）
	ProgressSeconds uint32 `gorm:"not null;default:0" json:"progress_seconds"`

	// 最后观看时间
	LastWatchedAt time.Time `gorm:"not null;index" json:"last_watched_at"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (VideoWatchHistory) TableName() string {
	return "video_watch_histories"
}

// VideoDanmaku 视频弹幕
// 对应表：video_danmakus
//
// 存储用户发送的弹幕信息
type VideoDanmaku struct {
	// 主键ID
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 视频ID
	VideoID uint64 `gorm:"not null;index" json:"video_id"`

	// 发送用户ID
	UserID uint64 `gorm:"not null;index" json:"user_id"`

	// 弹幕内容
	Content string `gorm:"size:120;not null" json:"content"`

	// 出现时间（毫秒）
	TimeMs uint32 `gorm:"not null;index" json:"time_ms"`

	// 弹幕颜色
	// 默认白色
	Color string `gorm:"size:16;not null;default:'#ffffff'" json:"color"`

	// 弹幕模式
	// scroll=滚动弹幕
	// top=顶部弹幕
	// bottom=底部弹幕
	Mode string `gorm:"size:16;not null;default:'scroll'" json:"mode"`

	// 状态
	// 0=隐藏
	// 1=正常显示
	Status uint8 `gorm:"not null;default:1;index" json:"status"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (VideoDanmaku) TableName() string {
	return "video_danmakus"
}
