package authstate

import (
	"context"
	"sync"
	"time"
)

type memoryCounter struct {
	count     int64
	expiresAt time.Time
}

type MemoryStore struct {
	mu       sync.Mutex
	now      func() time.Time
	revoked  map[string]time.Time
	counters map[string]memoryCounter
}

func NewMemoryStore() *MemoryStore {
	return newMemoryStore(time.Now)
}

func newMemoryStore(now func() time.Time) *MemoryStore {
	return &MemoryStore{
		now:      now,
		revoked:  make(map[string]time.Time),
		counters: make(map[string]memoryCounter),
	}
}

func (s *MemoryStore) RevokeAccessToken(_ context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[accessRevocationKey(jti)] = s.now().Add(ttl)
	return nil
}

func (s *MemoryStore) IsAccessTokenRevoked(_ context.Context, jti string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := accessRevocationKey(jti)
	expiresAt, ok := s.revoked[key]
	if !ok {
		return false, nil
	}
	if !expiresAt.After(s.now()) {
		delete(s.revoked, key)
		return false, nil
	}
	return true, nil
}

func (s *MemoryStore) CheckLoginLimit(_ context.Context, account, ip string, accountLimit, ipLimit int64, _ time.Duration) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	accountRetry := s.counterRetryAfter(accountFailureKey(account), accountLimit, now)
	ipRetry := s.counterRetryAfter(ipFailureKey(ip), ipLimit, now)
	return maxDuration(accountRetry, ipRetry), nil
}

func (s *MemoryStore) RecordLoginFailure(_ context.Context, account, ip string, accountLimit, ipLimit int64, window time.Duration) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	accountCounter := s.incrementCounter(accountFailureKey(account), now, window)
	ipCounter := s.incrementCounter(ipFailureKey(ip), now, window)

	var accountRetry time.Duration
	if accountCounter.count >= accountLimit {
		accountRetry = accountCounter.expiresAt.Sub(now)
	}
	var ipRetry time.Duration
	if ipCounter.count >= ipLimit {
		ipRetry = ipCounter.expiresAt.Sub(now)
	}
	return maxDuration(accountRetry, ipRetry), nil
}

func (s *MemoryStore) ResetLoginAccount(_ context.Context, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.counters, accountFailureKey(account))
	return nil
}

func (s *MemoryStore) Ping(context.Context) error {
	return nil
}

func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) incrementCounter(key string, now time.Time, window time.Duration) memoryCounter {
	counter, ok := s.counters[key]
	if !ok || !counter.expiresAt.After(now) {
		counter = memoryCounter{expiresAt: now.Add(window)}
	}
	counter.count++
	s.counters[key] = counter
	return counter
}

func (s *MemoryStore) counterRetryAfter(key string, limit int64, now time.Time) time.Duration {
	counter, ok := s.counters[key]
	if !ok {
		return 0
	}
	if !counter.expiresAt.After(now) {
		delete(s.counters, key)
		return 0
	}
	if counter.count < limit {
		return 0
	}
	return counter.expiresAt.Sub(now)
}
