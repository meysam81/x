package ratelimit

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
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
			b := rl.TokenBucket(t.Context(), key)
			if i < 10 {
				if !b {
					t.Fatalf("[%s] should have been allowed but rejected: %d", testName, i)
				}
			} else {
				if b {
					t.Fatalf("[%s] should have been rejected but allowed: %d", testName, i)
				}
			}
		})
	}

}
