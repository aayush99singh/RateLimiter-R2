package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucketLimiter_Allow(t *testing.T) {
	limit := 5
	interval := 1 * time.Second
	limiter := NewTokenBucketLimiter(limit, interval)
	defer limiter.Stop()

	key := "test-client"

	for i := 0; i < limit; i++ {
		if !limiter.Allow(key) {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	if limiter.Allow(key) {
		t.Errorf("Request %d should be rejected", limit+1)
	}
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	limit := 1
	interval := 100 * time.Millisecond
	limiter := NewTokenBucketLimiter(limit, interval)
	defer limiter.Stop()

	key := "test-client-refill"

	if !limiter.Allow(key) {
		t.Fatal("First request should be allowed")
	}

	if limiter.Allow(key) {
		t.Fatal("Immediate second request should be rejected")
	}

	time.Sleep(150 * time.Millisecond)

	if !limiter.Allow(key) {
		t.Fatal("Request after refill should be allowed")
	}
}

func TestTokenBucketLimiter_Concurrency(t *testing.T) {
	limit := 1000
	interval := 1 * time.Second
	limiter := NewTokenBucketLimiter(limit, interval)
	defer limiter.Stop()

	key := "concurrent-client"
	var wg sync.WaitGroup
	workers := 10
	requestsPerWorker := 50

	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				if limiter.Allow(key) {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	expected := workers * requestsPerWorker
	if allowedCount != expected {
		t.Errorf("Expected %d allowed requests, got %d", expected, allowedCount)
	}
}

func TestTokenBucketLimiter_Cleanup(t *testing.T) {
	limit := 1
	interval := 1 * time.Second
	limiter := NewTokenBucketLimiter(limit, interval)
	limiter.cleanupTimeout = 200 * time.Millisecond
	limiter.cleanupTick.Stop()
	limiter.cleanupTick = time.NewTicker(50 * time.Millisecond)

	defer limiter.Stop()

	key := "cleanup-client"
	limiter.Allow(key)

	limiter.mu.Lock()
	if _, exists := limiter.buckets[key]; !exists {
		limiter.mu.Unlock()
		t.Fatal("Client bucket should exist immediately after request")
	}
	limiter.mu.Unlock()

	time.Sleep(400 * time.Millisecond)

	limiter.mu.Lock()
	if _, exists := limiter.buckets[key]; exists {
		limiter.mu.Unlock()
		t.Error("Client bucket should have been cleaned up")
	}
	limiter.mu.Unlock()
}
