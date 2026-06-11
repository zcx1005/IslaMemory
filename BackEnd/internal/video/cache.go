package video

import (
	platformcache "IslaMemory/BackEnd/platform/cache"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	videoLatestZSetKey  = "cache:videos:latest:zset"
	videoLatestTotalKey = "cache:videos:latest:total"
	videoDetailKeyFmt   = "cache:videos:detail:%s"
	videoMissKeyFmt     = "cache:videos:miss:%s"
	videoCommentsFmt    = "cache:videos:comments:%s"
	videoDanmakuFmt     = "cache:videos:danmaku:%s"
	videoHistoryFmt     = "cache:users:%d:history:p:%d:s:%d"
	videoLockKeyFmt     = "lock:video:%s"

	videoListCacheTTL  = 10 * time.Minute
	videoItemCacheTTL  = 30 * time.Minute
	videoMissCacheTTL  = 2 * time.Minute
	videoLockTTL       = 3 * time.Minute
	videoMaxCacheItems = 1000
)

type cachedVideoListItem struct {
	ID   uint64        `json:"id"`
	Item VideoListItem `json:"item"`
}

type cachedVideoDetail struct {
	ID     uint64      `json:"id"`
	Detail VideoDetail `json:"detail"`
}

func (s *Service) redisEnabled() bool { return s.rdb != nil }

func (s *Service) cacheLatestPublicVideos(ctx context.Context, rows []VideoRow, total int64) {
	if !s.redisEnabled() || len(rows) == 0 {
		return
	}
	items := toVideoList(rows)
	pipe := s.rdb.Pipeline()
	for i, row := range rows {
		b, err := json.Marshal(cachedVideoListItem{ID: row.ID, Item: items[i]})
		if err != nil {
			continue
		}
		pipe.ZAdd(ctx, videoLatestZSetKey, redis.Z{Score: float64(row.ID), Member: string(b)})
	}
	pipe.ZRemRangeByRank(ctx, videoLatestZSetKey, 0, -videoMaxCacheItems-1)
	pipe.Set(ctx, videoLatestTotalKey, total, videoListCacheTTL)
	pipe.Expire(ctx, videoLatestZSetKey, videoListCacheTTL)
	_, _ = pipe.Exec(ctx)
}

func (s *Service) getLatestPublicVideosFromCache(ctx context.Context, page, pageSize int) ([]VideoListItem, int64, bool) {
	if !s.redisEnabled() {
		return nil, 0, false
	}
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1
	members, err := s.rdb.ZRevRange(ctx, videoLatestZSetKey, start, stop).Result()
	if err != nil || len(members) == 0 {
		return nil, 0, false
	}
	list := make([]VideoListItem, 0, len(members))
	for _, m := range members {
		var item cachedVideoListItem
		if err := json.Unmarshal([]byte(m), &item); err != nil {
			return nil, 0, false
		}
		list = append(list, item.Item)
	}
	total, err := s.rdb.Get(ctx, videoLatestTotalKey).Int64()
	if err != nil {
		total = int64(len(list))
	}
	return list, total, true
}

func (s *Service) getCachedDetail(ctx context.Context, publicID string) (*VideoDetail, bool, error) {
	if !s.redisEnabled() {
		return nil, false, nil
	}
	miss, err := s.rdb.Exists(ctx, fmt.Sprintf(videoMissKeyFmt, publicID)).Result()
	if err == nil && miss > 0 {
		return nil, true, ErrVideoNotFound
	}
	val, err := s.rdb.Get(ctx, fmt.Sprintf(videoDetailKeyFmt, publicID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, nil
	}
	var cached cachedVideoDetail
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, false, nil
	}
	return &cached.Detail, true, nil
}

func (s *Service) setCachedDetail(ctx context.Context, row *VideoRow, detail *VideoDetail) {
	if !s.redisEnabled() || row == nil || detail == nil {
		return
	}
	b, err := json.Marshal(cachedVideoDetail{ID: row.ID, Detail: *detail})
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, fmt.Sprintf(videoDetailKeyFmt, row.PublicID), b, videoItemCacheTTL).Err()
}

func (s *Service) setCachedVideoMiss(ctx context.Context, publicID string) {
	if s.redisEnabled() {
		_ = s.rdb.Set(ctx, fmt.Sprintf(videoMissKeyFmt, publicID), "1", videoMissCacheTTL).Err()
	}
}

func (s *Service) invalidateVideoCache(ctx context.Context, publicID string) {
	if !s.redisEnabled() {
		return
	}
	_ = s.rdb.Del(ctx,
		fmt.Sprintf(videoDetailKeyFmt, publicID),
		fmt.Sprintf(videoMissKeyFmt, publicID),
		fmt.Sprintf(videoCommentsFmt, publicID),
		fmt.Sprintf(videoDanmakuFmt, publicID),
		videoLatestZSetKey,
		videoLatestTotalKey,
	).Err()
}

func (s *Service) getJSONCache(ctx context.Context, key string, dest any) bool {
	if !s.redisEnabled() {
		return false
	}
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(val), dest) == nil
}

func (s *Service) setJSONCache(ctx context.Context, key string, value any, ttl time.Duration) {
	if !s.redisEnabled() {
		return
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, key, b, ttl).Err()
}

func (s *Service) deleteHistoryCache(ctx context.Context, userID uint64) {
	if !s.redisEnabled() {
		return
	}
	iter := s.rdb.Scan(ctx, 0, fmt.Sprintf("cache:users:%d:history:*", userID), 100).Iterator()
	keys := make([]string, 0, 100)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			_ = s.rdb.Del(ctx, keys...).Err()
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		_ = s.rdb.Del(ctx, keys...).Err()
	}
}

func (s *Service) withRedisLock(ctx context.Context, name string, ttl time.Duration, fn func() error) error {
	if !s.redisEnabled() {
		return fn()
	}
	lock := platformcache.NewRedisLock(s.rdb, fmt.Sprintf(videoLockKeyFmt, name), ttl)
	ok, err := lock.Lock(ctx)
	if err != nil || !ok {
		if err != nil {
			return err
		}
		return ErrInvalidInput
	}
	defer lock.Unlock(context.Background())
	return fn()
}

const (
	videoBloomKey      = "cache:videos:bloom"
	videoBloomReadyKey = "cache:videos:bloom:ready"
	videoBloomBits     = 1 << 22
)

func (s *Service) ensureVideoBloom(ctx context.Context) {
	if !s.redisEnabled() {
		return
	}
	ready, err := s.rdb.Exists(ctx, videoBloomReadyKey).Result()
	if err != nil || ready > 0 {
		return
	}
	ids, err := s.repo.ListPublicVideoPublicIDs(ctx)
	if err != nil {
		return
	}
	pipe := s.rdb.Pipeline()
	for _, id := range ids {
		for _, offset := range bloomOffsets(id) {
			pipe.SetBit(ctx, videoBloomKey, offset, 1)
		}
	}
	pipe.Set(ctx, videoBloomReadyKey, "1", 24*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *Service) addVideoBloom(ctx context.Context, publicID string) {
	if !s.redisEnabled() || publicID == "" {
		return
	}
	pipe := s.rdb.Pipeline()
	for _, offset := range bloomOffsets(publicID) {
		pipe.SetBit(ctx, videoBloomKey, offset, 1)
	}
	pipe.Set(ctx, videoBloomReadyKey, "1", 24*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *Service) mightContainVideo(ctx context.Context, publicID string) bool {
	if !s.redisEnabled() || publicID == "" {
		return true
	}
	s.ensureVideoBloom(ctx)
	ready, err := s.rdb.Exists(ctx, videoBloomReadyKey).Result()
	if err != nil || ready == 0 {
		return true
	}
	for _, offset := range bloomOffsets(publicID) {
		bit, err := s.rdb.GetBit(ctx, videoBloomKey, offset).Result()
		if err != nil || bit == 0 {
			return false
		}
	}
	return true
}

func bloomOffsets(value string) []int64 {
	var h1 uint64 = 1469598103934665603
	var h2 uint64 = 1099511628211
	for i := 0; i < len(value); i++ {
		h1 ^= uint64(value[i])
		h1 *= 1099511628211
		h2 += uint64(value[i]) + (h2 << 10)
		h2 ^= h2 >> 6
	}
	h2 += h2 << 3
	h2 ^= h2 >> 11
	h2 += h2 << 15
	return []int64{int64(h1 % videoBloomBits), int64(h2 % videoBloomBits), int64((h1 + h2) % videoBloomBits)}
}
