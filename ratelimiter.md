# Rate Limiter Documentation

## `pkg/ratelimiter/limiter.go`

### Types

**RateLimiter Interface**
- `Allow` checks if a request from the given clientID is allowed.

**tokenBucket Struct**
- Represents the state for a single client in the token bucket algorithm.

**TokenBucketLimiter Struct**
- Implements `RateLimiter` using the Token Bucket algorithm.
- It is thread-safe.
- `rate`: Tokens added per second.
- `capacity`: Maximum number of tokens in the bucket.
- `cleanupTick`: Ticker for cleanup routine.
- `stopCleanup`: Channel to stop cleanup routine.
- `cleanupTimeout`: Duration after which an inactive client is removed.

### Functions

**NewTokenBucketLimiter**
- Creates a new `TokenBucketLimiter`.
- `limit`: number of requests allowed per interval.
- `interval`: the time interval for the limit (e.g., 1 minute).
- This implementation allows a burst equal to the limit.
- Panics if limit is non-positive or interval is non-positive.
- Sets `cleanupTimeout` to 10 minutes (Remove clients inactive for 10 minutes).

**TokenBucketLimiter.Allow**
- Checks if the request is allowed for the clientID.
- Calculate tokens to add based on time passed.

**TokenBucketLimiter.cleanupLoop**
- Periodically removes old client entries to prevent memory leaks.
- If the bucket is full (meaning no activity for a while) and enough time has passed (specifically, we can check how long it takes to fill from empty to full: capacity / rate. But simpler is to check if lastRefill is too old), it deletes the client.

**TokenBucketLimiter.Stop**
- Stops the cleanup goroutine.

## `main.go`

### Functions

**main**
- Create a rate limiter: 5 requests per second.
- Simulate a burst of requests.
- Wait for a second to refill tokens.
- Simulate 1 more request.

## `pkg/ratelimiter/limiter_test.go`

### Tests

**TestTokenBucketLimiter_Allow**
- All first 5 requests should be allowed.
- Next request should be rejected.

**TestTokenBucketLimiter_Refill**
- First request should be allowed.
- Immediate second request should be rejected.
- Wait for refill.
- Request after refill should be allowed.

**TestTokenBucketLimiter_Concurrency**
- Since limit is 1000 and we send 500 requests, all should be allowed.

**TestTokenBucketLimiter_Cleanup**
- Use a short cleanup timeout for testing.
- Override ticker to be faster for test.
- Client should exist immediately after request.
- Wait for cleanup.
- Client should have been cleaned up.


## Design Decisions

### Algorithm Choice: Token Bucket

This implementation uses the **Token Bucket** algorithm.

#### Why Token Bucket?
1.  **Handling Bursts**: Unlike the Leaky Bucket algorithm which enforces a constant output rate, Token Bucket allows for bursts of traffic up to the bucket's capacity. This is often more suitable for real-world APIs where user activity is sporadic and can come in short spikes.
2.  **Memory Efficiency**: It requires very little memory per user (just the token count and the timestamp of the last refill).
3.  **Simplicity**: It is straightforward to implement and reason about.

#### Comparison with Other Approaches

*   **Fixed Window Counter**:
    *   *Pros*: Easiest to implement.
    *   *Cons*: Susceptible to the "double burst" issue where a user can consume the full limit at the end of one window and again at the beginning of the next, effectively allowing 2x limits in a short duration.
*   **Sliding Window Log**:
    *   *Pros*: Very accurate.
    *   *Cons*: High memory footprint as it needs to store a timestamp for every request. Not scalable for high-throughput systems or long windows.
*   **Sliding Window Counter**:
    *   *Pros*: Balances accuracy and memory.
    *   *Cons*: More complex to implement than Token Bucket. Token Bucket often provides a "good enough" approximation for most rate-limiting needs while naturally handling bursts.
