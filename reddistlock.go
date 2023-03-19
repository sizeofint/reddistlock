package reddistlock

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type RedDistLock interface {
	Lock(ctx context.Context, key string, timeout int) (Lock, error)
}

type redDistLock struct {
	redisClient RedisClient
}

type chResult struct {
	Result bool
	Error  error
}

func (r *redDistLock) createAndPushTicket(ctx context.Context, key string) (string, error) {
	randomBytes, err := generateRandomBytes(16)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	ticketString := fmt.Sprintf("%x", randomBytes)
	r.redisClient.RPush(ctx, fmt.Sprintf("unlock-wait-list-%s", key), ticketString)
	return ticketString, nil
}

func (r *redDistLock) waitForUnlock(ctx context.Context, sub *redis.PubSub, timeout int, ticketString string, key string) error {
	var newCtx context.Context
	if timeout != 0 {
		var cancel context.CancelFunc
		newCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	} else {
		newCtx = context.Background()
	}

	res := make(chan chResult)
	go func(sub *redis.PubSub, ch chan chResult) {
		for {
			isReleased, err := r.receiveMessageWithTimeout(ctx, sub, key, ticketString)
			if err != nil {
				res <- chResult{false, err}
				return
			}
			if isReleased {
				res <- chResult{true, nil}
				return
			}
		}
	}(sub, res)

	select {
	case r := <-res:
		if r.Error != nil {
			return r.Error
		}
	case <-newCtx.Done():
		r.redisClient.LRem(ctx, fmt.Sprintf("unlock-wait-list-%s", key), 1, ticketString)
		return errors.New("timout")
	}
	return nil
}

func (r *redDistLock) receiveMessageWithTimeout(ctx context.Context, sub *redis.PubSub, key string, ticketString string) (bool, error) {

	var err error
	var msg *redis.Message
	msg, err = sub.ReceiveMessage(ctx)
	if err != nil {
		log.Println(err.Error())
		return false, err
	}
	if msg.Payload == key {
		var waitingProcess []string
		waitingProcess, err = r.redisClient.LRange(ctx, fmt.Sprintf("unlock-wait-list-%s", key), 0, 0).Result()
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		if len(waitingProcess) == 0 || waitingProcess[0] == ticketString {
			return true, nil
		}
	}
	return false, nil
}

func (r *redDistLock) Lock(ctx context.Context, key string, timeout int) (Lock, error) {
	var ticketString string
	lk := &lock{
		key:         key,
		redisClient: r.redisClient,
	}
	var err error
	ticketString, err = r.createAndPushTicket(ctx, key)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	lk.ticket = ticketString
	waitingProcess, err := r.redisClient.LRange(ctx, fmt.Sprintf("unlock-wait-list-%s", key), 0, 0).Result()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	if ticketString != waitingProcess[0] {
		subscriber := r.redisClient.Subscribe(ctx, "lock-released")
		defer subscriber.Unsubscribe(ctx, "lock-released")
		err = r.waitForUnlock(ctx, subscriber, timeout, ticketString, key)
		if err != nil {
			return nil, err
		}
	}
	return lk, nil
}

func New(redisClient RedisClient) RedDistLock {
	return &redDistLock{redisClient}
}
