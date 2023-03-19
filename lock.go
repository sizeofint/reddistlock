package reddistlock

import (
	"context"
	"fmt"
)

type Lock interface {
	Unlock(ctx context.Context) error
}

func (l *lock) Unlock(ctx context.Context) error {
	l.redisClient.LRem(ctx, fmt.Sprintf("unlock-wait-list-%s", l.key), 1, l.ticket)
	err := l.redisClient.Publish(ctx, "lock-released", l.key).Err()
	if err != nil {
		return err
	}
	return nil
}

type lock struct {
	ticket      string
	key         string
	redisClient RedisClient
}
