package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

// RedisLock 是基于 SET NX PX 和 Lua 安全释放实现的后台分布式锁。
type RedisLock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

func NewRedisLock(client *redis.Client, key string, ttl time.Duration) *RedisLock {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return &RedisLock{client: client, key: key, value: hex.EncodeToString(buf), ttl: ttl}
}

func (l *RedisLock) Lock(ctx context.Context) (bool, error) {
	return l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
}

func (l *RedisLock) Unlock(ctx context.Context) error {
	return l.client.Eval(ctx, unlockScript, []string{l.key}, l.value).Err()
}
