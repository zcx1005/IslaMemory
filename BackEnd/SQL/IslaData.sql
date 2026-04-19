SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `video_comments`;
DROP TABLE IF EXISTS `video_favorites`;
DROP TABLE IF EXISTS `video_likes`;
DROP TABLE IF EXISTS `videos`;
DROP TABLE IF EXISTS `categories`;
DROP TABLE IF EXISTS `login_logs`;
DROP TABLE IF EXISTS `users`;

-- =========================
-- 用户表（支持软删除）
-- =========================
CREATE TABLE `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    `account` VARCHAR(64) NOT NULL COMMENT '登录账号',
    `username` VARCHAR(64) NOT NULL COMMENT '显示用户名',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希',
    `avatar_url` VARCHAR(255) DEFAULT NULL COMMENT '头像地址',
    `role` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色: 0普通用户 1管理员',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1正常 0禁用',
    `can_upload` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '是否允许上传 1是，0否',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `password_changed_at` DATETIME DEFAULT NULL COMMENT '密码最后修改时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除操作者ID',
    `delete_reason` VARCHAR(255) DEFAULT NULL COMMENT '删除原因',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_account` (`account`),
    UNIQUE KEY `uk_users_username` (`username`),
    KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- =========================
-- 视频分类表
-- =========================
CREATE TABLE `categories` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '分类ID',
    `name` VARCHAR(100) NOT NULL COMMENT '分类名称',
    `slug` varchar(100) NOT NULL COMMENT '分类英文表示',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序值',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0停用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_categories_name` (`name`),
    UNIQUE KEY `uk_categories_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频分类表';


-- =========================
-- 标签字典表(未实现)
-- =========================
CREATE TABLE `tags` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签ID',
    `name` VARCHAR(50) NOT NULL COMMENT '标签名称',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tags_name` (`name`) -- 保证标签名不重复
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='标签表';

-- =========================
-- 视频-标签 关联表（中间表）（未实现）
-- =========================
CREATE TABLE `video_tag_relations` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id` BIGINT UNSIGNED NOT NULL COMMENT '视频ID',
    `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_tag` (`video_id`, `tag_id`), -- 防止同一个视频重复绑定同一个标签
    KEY `idx_tag_id` (`tag_id`), -- 方便通过标签查视频
    CONSTRAINT `fk_rel_video_id` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_rel_tag_id` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频标签关联表';



-- =========================
-- 视频主表（支持软删除）
-- =========================
CREATE TABLE `videos` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '视频ID',
    `public_id` VARCHAR(32) NOT NULL COMMENT '对外公开视频号',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '上传者ID',
    `category_id` BIGINT UNSIGNED NOT NULL COMMENT '分类ID',
    `title` VARCHAR(200) NOT NULL COMMENT '视频标题',
    `description` TEXT DEFAULT NULL COMMENT '视频简介',
    `source_video_url` VARCHAR(255) NOT NULL COMMENT '原始上传视频地址',
    `playback_url` VARCHAR(255) DEFAULT NULL COMMENT '实际播放地址，如m3u8',
    `playback_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0原始文件 1HLS',
    `transcode_status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0待处理 1处理中 2成功 3失败',
    `transcode_progress` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '转码进度 0-100',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0草稿 1待审核 2已发布 3已下架',
    `transcode_error` VARCHAR(500) DEFAULT NULL COMMENT '转码失败原因',
    `cover_url` VARCHAR(255) DEFAULT NULL COMMENT '封面地址',
    `duration_seconds` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '视频时长秒数',
    `width` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '视频宽度',
    `height` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '视频高度',
    `file_size_bytes` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '原始文件大小(字节)',
    `play_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '播放量',
    `like_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点赞数',
    `favorite_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '收藏数',
    `comment_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '评论数',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除操作者ID',
    `delete_reason` VARCHAR(255) DEFAULT NULL COMMENT '删除原因',
    `published_at` DATETIME DEFAULT NULL COMMENT '发布时间',
    `reviewed_at` DATETIME DEFAULT NULL COMMENT '审核时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_videos_public_id` (`public_id`),
    KEY `idx_videos_user_id` (`user_id`),
    KEY `idx_videos_category_id` (`category_id`),
    KEY `idx_videos_created_at` (`created_at`),
    KEY `idx_videos_deleted_at` (`deleted_at`),
    KEY `idx_videos_category_created_at` (`category_id`, `created_at`),
    KEY `idx_videos_category_play_count` (`category_id`, `play_count`),
    CONSTRAINT `fk_videos_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
    CONSTRAINT `fk_videos_category_id` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`),
    CONSTRAINT `fk_videos_deleted_by` FOREIGN KEY (`deleted_by`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频表';

-- =========================
-- 视频点赞表
-- 一个用户对一个视频只能点一次赞
-- 取消点赞直接物理删除
-- =========================
CREATE TABLE `video_likes` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id` BIGINT UNSIGNED NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_likes_video_user` (`video_id`, `user_id`),
    KEY `idx_video_likes_user_id` (`user_id`),
    CONSTRAINT `fk_video_likes_video_id` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_video_likes_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频点赞表';

-- =========================
-- 视频收藏表
-- 一个用户对一个视频只能收藏一次
-- 取消收藏直接物理删除
-- =========================
CREATE TABLE `video_favorites` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `video_id` BIGINT UNSIGNED NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_video_favorites_video_user` (`video_id`, `user_id`),
    KEY `idx_video_favorites_user_id` (`user_id`),
    CONSTRAINT `fk_video_favorites_video_id` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_video_favorites_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频收藏表';

-- =========================
-- 视频评论表（支持软删除）
-- parent_id 为空：一级评论
-- parent_id 不为空：回复评论
-- root_id 指向所属一级评论
-- =========================
CREATE TABLE `video_comments` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '评论ID',
    `video_id` BIGINT UNSIGNED NOT NULL COMMENT '视频ID',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '评论用户ID',
    `parent_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '父评论ID',
    `root_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '根评论ID',
    `reply_to_user_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '被回复用户ID',
    `content` TEXT NOT NULL COMMENT '评论内容',
    `like_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '评论点赞数',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1正常 0隐藏',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除操作者ID',
    `delete_reason` VARCHAR(255) DEFAULT NULL COMMENT '删除原因',
    PRIMARY KEY (`id`),
    KEY `idx_video_comments_video_id` (`video_id`),
    KEY `idx_video_comments_user_id` (`user_id`),
    KEY `idx_video_comments_parent_id` (`parent_id`),
    KEY `idx_video_comments_root_id` (`root_id`),
    KEY `idx_video_comments_created_at` (`created_at`),
    KEY `idx_video_comments_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_video_comments_video_id` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_video_comments_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_video_comments_parent_id` FOREIGN KEY (`parent_id`) REFERENCES `video_comments` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_video_comments_reply_to_user_id` FOREIGN KEY (`reply_to_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
    CONSTRAINT `fk_video_comments_deleted_by` FOREIGN KEY (`deleted_by`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频评论表';

-- =========================
-- 登录日志表
-- =========================
CREATE TABLE `login_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED DEFAULT NULL,
    `account_snapshot` VARCHAR(64) DEFAULT NULL COMMENT '账号快照',
    `username_snapshot` VARCHAR(64) DEFAULT NULL COMMENT '用户名快照',
    `ip` VARCHAR(64) DEFAULT NULL COMMENT 'IP',
    `user_agent` VARCHAR(255) DEFAULT NULL COMMENT '浏览器UA',
    `login_status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1成功 0失败',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_login_logs_user_id` (`user_id`),
    CONSTRAINT `fk_login_logs_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录日志表';

SET FOREIGN_KEY_CHECKS = 1;