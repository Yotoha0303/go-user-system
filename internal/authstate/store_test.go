package authstate

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryStoreRevocationExpires(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(func() time.Time { return now })

	if err := store.RevokeAccessToken(context.Background(), "jti-1", time.Minute); err != nil {
		t.Fatalf("revoke access token failed: %v", err)
	}
	revoked, err := store.IsAccessTokenRevoked(context.Background(), "jti-1")
	if err != nil || !revoked {
		t.Fatalf("expected token revoked, revoked=%v err=%v", revoked, err)
	}

	now = now.Add(time.Minute)
	revoked, err = store.IsAccessTokenRevoked(context.Background(), "jti-1")
	if err != nil || revoked {
		t.Fatalf("expected expired revocation removed, revoked=%v err=%v", revoked, err)
	}
}

func TestMemoryStoreLimitsAccountAndResetsOnlyAccountCounter(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore(func() time.Time { return now })
	ctx := context.Background()

	if retry, err := store.RecordLoginFailure(ctx, "alice", "192.0.2.10", 2, 10, time.Minute); err != nil || retry != 0 {
		t.Fatalf("expected first failure allowed, retry=%s err=%v", retry, err)
	}
	retry, err := store.RecordLoginFailure(ctx, "alice", "192.0.2.10", 2, 10, time.Minute)
	if err != nil || retry <= 0 {
		t.Fatalf("expected account limited, retry=%s err=%v", retry, err)
	}

	if err := store.ResetLoginAccount(ctx, "alice"); err != nil {
		t.Fatalf("reset account failures failed: %v", err)
	}
	retry, err = store.CheckLoginLimit(ctx, "alice", "192.0.2.10", 2, 10, time.Minute)
	if err != nil || retry != 0 {
		t.Fatalf("expected account reset while IP remains below limit, retry=%s err=%v", retry, err)
	}
}

func TestMemoryStoreLimitsSharedIPAcrossAccounts(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if retry, err := store.RecordLoginFailure(ctx, "alice", "192.0.2.10", 10, 2, time.Minute); err != nil || retry != 0 {
		t.Fatalf("expected first failure allowed, retry=%s err=%v", retry, err)
	}
	retry, err := store.RecordLoginFailure(ctx, "bob", "192.0.2.10", 10, 2, time.Minute)
	if err != nil || retry <= 0 {
		t.Fatalf("expected shared IP limited, retry=%s err=%v", retry, err)
	}
}

func TestMemoryStoreIsSafeForConcurrentFailureUpdates(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	const attempts = 100

	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.RecordLoginFailure(ctx, "alice", "192.0.2.10", attempts, attempts+1, time.Minute)
		}()
	}
	wg.Wait()

	retry, err := store.CheckLoginLimit(ctx, "alice", "192.0.2.10", attempts, attempts+1, time.Minute)
	if err != nil || retry <= 0 {
		t.Fatalf("expected concurrent account failures to reach limit, retry=%s err=%v", retry, err)
	}
}

func TestAuthenticationKeysDoNotContainRawIdentifiers(t *testing.T) {
	tests := []struct {
		key   string
		plain string
	}{
		{key: accessRevocationKey("jti-secret"), plain: "jti-secret"},
		{key: accountFailureKey("alice@example.com"), plain: "alice@example.com"},
		{key: ipFailureKey("192.0.2.10"), plain: "192.0.2.10"},
	}
	for _, test := range tests {
		if strings.Contains(test.key, test.plain) {
			t.Fatalf("expected key to hide raw identifier %q, got %q", test.plain, test.key)
		}
	}
}

func TestRedisStorePersistsTTLAndLoginLimits(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := NewRedisStore(client)
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()

	if err := store.RevokeAccessToken(ctx, "jti-secret", time.Minute); err != nil {
		t.Fatalf("revoke access token failed: %v", err)
	}
	if ttl := server.TTL(accessRevocationKey("jti-secret")); ttl != time.Minute {
		t.Fatalf("expected revocation TTL 1m, got %s", ttl)
	}

	if retry, err := store.RecordLoginFailure(ctx, "alice", "192.0.2.10", 2, 10, time.Minute); err != nil || retry != 0 {
		t.Fatalf("expected first failure allowed, retry=%s err=%v", retry, err)
	}
	retry, err := store.RecordLoginFailure(ctx, "alice", "192.0.2.10", 2, 10, time.Minute)
	if err != nil || retry <= 0 {
		t.Fatalf("expected account limited, retry=%s err=%v", retry, err)
	}

	if err := store.ResetLoginAccount(ctx, "alice"); err != nil {
		t.Fatalf("reset account failures failed: %v", err)
	}
	if server.Exists(accountFailureKey("alice")) {
		t.Fatal("expected account failure key removed")
	}
	if !server.Exists(ipFailureKey("192.0.2.10")) {
		t.Fatal("expected IP failure key retained")
	}
}

func TestRedisStoreReturnsErrorWhenRedisIsUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr:         server.Addr(),
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		MaxRetries:   0,
	})
	store := NewRedisStore(client)
	t.Cleanup(func() { _ = store.Close() })
	server.Close()

	if _, err := store.IsAccessTokenRevoked(context.Background(), "jti-1"); err == nil {
		t.Fatal("expected Redis error instead of allowing authentication")
	}
}
