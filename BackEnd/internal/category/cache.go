package category

import (
	"context"
	"encoding/json"
	"time"
)

const (
	categoryEnabledKey = "cache:categories:enabled"
	categoryBySlugFmt  = "cache:categories:slug:%s"
	categoryCacheTTL   = 30 * time.Minute
)

func (s *Service) redisEnabled() bool { return s.rdb != nil }

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

func (s *Service) invalidateCategoryCache(ctx context.Context, slugs ...string) {
	if !s.redisEnabled() {
		return
	}
	keys := []string{categoryEnabledKey}
	for _, slug := range slugs {
		if slug != "" {
			keys = append(keys, "cache:categories:slug:"+slug)
		}
	}
	_ = s.rdb.Del(ctx, keys...).Err()
}
