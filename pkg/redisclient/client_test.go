package redisclient

import (
	"context"
	"testing"
	"time"

	"go-user-system/config"

	"github.com/alicebob/miniredis/v2"
)

func TestNewPingsRedisBeforeReturning(t *testing.T) {
	server := miniredis.RunT(t)

	client, err := New(context.Background(), config.RedisConfig{
		Address:      server.Addr(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PingTimeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("initialize Redis client failed: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
}

func TestNewFailsWhenRedisIsUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	address := server.Addr()
	server.Close()

	_, err := New(context.Background(), config.RedisConfig{
		Address:      address,
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		PingTimeout:  50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected Redis startup health check to fail")
	}
}
