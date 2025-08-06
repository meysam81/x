//go:build goexperiment.synctest
// +build goexperiment.synctest

package ratelimit

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
)

func TestMain(m *testing.M) {
	redisClient = redis.NewClient(&redis.Options{
		Addr:       "localhost:6379",
		ClientName: "x/ratelimit/test",
		DB:         0,
	})
	defer func() {
		err := redisClient.Close()
		if err != nil {
			log.Fatalf("failed closing redis client: %s", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed pinging redis: %s", err)
	}

	os.Exit(m.Run())
}

func TestTokenBucket(t *testing.T) {
	rl := &RateLimit{
		Redis:       redisClient,
		MaxRequests: 10,
		RefillRate:  1,
	}

	key := fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())
	for i := range 100 {
		testName := fmt.Sprintf("%s-%d", key, i)
		t.Run(testName, func(t *testing.T) {
			result := rl.TokenBucket(t.Context(), key)
			if i < 10 {
				if !result.Allowed {
					t.Fatalf("[%s] should have been allowed but rejected: %d", testName, i)
				}
			} else {
				if result.Allowed {
					t.Fatalf("[%s] should have been rejected but allowed: %d", testName, i)
				}
			}
		})
	}

}

func TestTokenBucketSleep(t *testing.T) {
	rl := &RateLimit{
		Redis:       redisClient,
		MaxRequests: 10,
		RefillRate:  1,
	}

	key := fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())

	synctest.Run(func() {
		for i := range 10 {
			testName := fmt.Sprintf("%s-%d", key, i)
			result := rl.TokenBucket(t.Context(), key)
			if !result.Allowed {
				t.Fatalf("[%s] should have been allowed but rejected: %d", testName, i)
			}

		}
		time.Sleep(3 * time.Second)

		for i := range 10 {
			testName := fmt.Sprintf("%s-%d", key, i)
			result := rl.TokenBucket(t.Context(), key)
			if i < 3 {
				if !result.Allowed {
					t.Fatalf("[%s] should have been allowed but rejected: %d", testName, i)
				}
			} else {
				if result.Allowed {
					t.Fatalf("[%s] should have been rejected but allowed: %d", testName, i)
				}
			}
		}
	})
}

func BenchmarkTokenBucket(b *testing.B) {
	rl := &RateLimit{
		Redis:       redisClient,
		MaxRequests: 1000,
		RefillRate:  100,
	}

	hostname, _ := os.Hostname()
	key := fmt.Sprintf("bench-%s-%s-%d", hostname, b.Name(), time.Now().Unix())
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			rl.TokenBucket(ctx, key)
		}
	})
}
