package authstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const keyPrefix = "go-user-system:auth:"

type Store interface {
	RevokeAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsAccessTokenRevoked(ctx context.Context, jti string) (bool, error)
	CheckLoginLimit(ctx context.Context, account, ip string, accountLimit, ipLimit int64, window time.Duration) (time.Duration, error)
	RecordLoginFailure(ctx context.Context, account, ip string, accountLimit, ipLimit int64, window time.Duration) (time.Duration, error)
	ResetLoginAccount(ctx context.Context, account string) error
	Ping(ctx context.Context) error
	Close() error
}

func accessRevocationKey(jti string) string {
	return keyPrefix + "access:revoked:" + digest(jti)
}

func accountFailureKey(account string) string {
	return keyPrefix + "login:account:" + digest(account)
}

func ipFailureKey(ip string) string {
	return keyPrefix + "login:ip:" + digest(ip)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func maxDuration(first, second time.Duration) time.Duration {
	if first > second {
		return first
	}
	return second
}
