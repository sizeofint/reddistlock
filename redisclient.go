package reddistlock

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type RedisClient interface {
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	LRem(ctx context.Context, key string, count int64, value interface{}) *redis.IntCmd
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	Unsubscribe(ctx context.Context, channels ...string) error
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}
