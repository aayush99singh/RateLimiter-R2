package ratelimiter

import (
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(clientID string) bool
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucketLimiter struct {
	mu             sync.Mutex
	clients        map[string]*tokenBucket
	rate           float64
	capacity       float64
	cleanupTick    *time.Ticker
	stopCleanup    chan struct{}
	cleanupTimeout time.Duration
}

func NewTokenBucketLimiter(limit int, interval time.Duration) *TokenBucketLimiter {
	if limit <= 0 {
		panic("limit must be positive")
	}
	if interval <= 0 {
		panic("interval must be positive")
	}

	rate := float64(limit) / interval.Seconds()
	l := &TokenBucketLimiter{
		clients:        make(map[string]*tokenBucket),
		rate:           rate,
		capacity:       float64(limit),
		cleanupTick:    time.NewTicker(1 * time.Minute),
		stopCleanup:    make(chan struct{}),
		cleanupTimeout: 10 * time.Minute,
	}

	go l.cleanupLoop()

	return l
}

func (l *TokenBucketLimiter) Allow(clientID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.clients[clientID]

	if !exists {
		bucket = &tokenBucket{
			tokens:     l.capacity,
			lastRefill: now,
		}
		l.clients[clientID] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	addedTokens := elapsed * l.rate

	bucket.tokens += addedTokens
	if bucket.tokens > l.capacity {
		bucket.tokens = l.capacity
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

func (l *TokenBucketLimiter) cleanupLoop() {
	for {
		select {
		case <-l.cleanupTick.C:
			l.mu.Lock()
			now := time.Now()
			for clientID, bucket := range l.clients {
				if now.Sub(bucket.lastRefill) > l.cleanupTimeout {
					delete(l.clients, clientID)
				}
			}
			l.mu.Unlock()
		case <-l.stopCleanup:
			l.cleanupTick.Stop()
			return
		}
	}
}

func (l *TokenBucketLimiter) Stop() {
	close(l.stopCleanup)
}
