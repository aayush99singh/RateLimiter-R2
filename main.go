package main

import (
	"fmt"
	"ratelimiter/pkg/ratelimiter"
	"time"
)

func main() {
	limiter := ratelimiter.NewTokenBucketLimiter(5, 1*time.Second)
	defer limiter.Stop()

	clientID := "user123"

	fmt.Println("Simulating burst of 10 requests (Limit is 5/sec)...")
	for i := 1; i <= 10; i++ {
		allowed := limiter.Allow(clientID)
		status := "Allowed"
		if !allowed {
			status = "Rejected"
		}
		fmt.Printf("Request %d: %s\n", i, status)
	}

	fmt.Println("\nWaiting for 1 second...")
	time.Sleep(1 * time.Second)

	fmt.Println("Simulating 1 more request...")
	if limiter.Allow(clientID) {
		fmt.Println("Request Allowed")
	} else {
		fmt.Println("Request Rejected")
	}
}
