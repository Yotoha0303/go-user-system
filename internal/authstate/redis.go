package authstate

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client redis.UniversalClient
}

func NewRedisStore(client redis.UniversalClient) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) RevokeAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.client.Set(contextOrBackground(ctx), accessRevocationKey(jti), "1", ttl).Err()
}

func (s *RedisStore) IsAccessTokenRevoked(ctx context.Context, jti string) (bool, error) {
	count, err := s.client.Exists(contextOrBackground(ctx), accessRevocationKey(jti)).Result()
	return count > 0, err
}

func (s *RedisStore) CheckLoginLimit(ctx context.Context, account, ip string, accountLimit, ipLimit int64, _ time.Duration) (time.Duration, error) {
	accountKey := accountFailureKey(account)
	ipKey := ipFailureKey(ip)
	opCtx := contextOrBackground(ctx)

	var values *redis.SliceCmd
	var accountTTL *redis.DurationCmd
	var ipTTL *redis.DurationCmd
	_, err := s.client.TxPipelined(opCtx, func(pipe redis.Pipeliner) error {
		values = pipe.MGet(opCtx, accountKey, ipKey)
		accountTTL = pipe.PTTL(opCtx, accountKey)
		ipTTL = pipe.PTTL(opCtx, ipKey)
		return nil
	})
	if err != nil {
		return 0, err
	}

	counts, err := values.Result()
	if err != nil {
		return 0, err
	}
	accountCount, err := counterValue(counts[0])
	if err != nil {
		return 0, err
	}
	ipCount, err := counterValue(counts[1])
	if err != nil {
		return 0, err
	}

	var accountRetry time.Duration
	if accountCount >= accountLimit {
		accountRetry = positiveTTL(accountTTL.Val())
	}
	var ipRetry time.Duration
	if ipCount >= ipLimit {
		ipRetry = positiveTTL(ipTTL.Val())
	}
	return maxDuration(accountRetry, ipRetry), nil
}

func (s *RedisStore) RecordLoginFailure(ctx context.Context, account, ip string, accountLimit, ipLimit int64, window time.Duration) (time.Duration, error) {
	accountKey := accountFailureKey(account)
	ipKey := ipFailureKey(ip)
	opCtx := contextOrBackground(ctx)

	var accountCount *redis.IntCmd
	var ipCount *redis.IntCmd
	var accountTTL *redis.DurationCmd
	var ipTTL *redis.DurationCmd
	_, err := s.client.TxPipelined(opCtx, func(pipe redis.Pipeliner) error {
		accountCount = pipe.Incr(opCtx, accountKey)
		pipe.ExpireNX(opCtx, accountKey, window)
		ipCount = pipe.Incr(opCtx, ipKey)
		pipe.ExpireNX(opCtx, ipKey, window)
		accountTTL = pipe.PTTL(opCtx, accountKey)
		ipTTL = pipe.PTTL(opCtx, ipKey)
		return nil
	})
	if err != nil {
		return 0, err
	}

	var accountRetry time.Duration
	if accountCount.Val() >= accountLimit {
		accountRetry = positiveTTL(accountTTL.Val())
	}
	var ipRetry time.Duration
	if ipCount.Val() >= ipLimit {
		ipRetry = positiveTTL(ipTTL.Val())
	}
	return maxDuration(accountRetry, ipRetry), nil
}

func (s *RedisStore) ResetLoginAccount(ctx context.Context, account string) error {
	return s.client.Del(contextOrBackground(ctx), accountFailureKey(account)).Err()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(contextOrBackground(ctx)).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func counterValue(value interface{}) (int64, error) {
	if value == nil {
		return 0, nil
	}
	count, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse login counter: %w", err)
	}
	return count, nil
}

func positiveTTL(ttl time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	return time.Second
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
