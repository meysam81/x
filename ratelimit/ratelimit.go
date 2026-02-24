// Package ratelimit provides Redis-backed rate limiting using common algorithms
// such as token bucket, leaky bucket, sliding window, and fixed window.
package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit holds the Redis client and configuration shared by all algorithms.
type RateLimit struct {
	Redis       *redis.Client
	MaxRequests int
	RefillRate  float32
	Window      time.Duration
}

// Result holds the outcome of a rate limit check, including whether the
// request was allowed and the remaining quota within the current window.
type Result struct {
	Allowed   bool
	Total     int64
	Remaining int64

	resetAt int64
}

// ResetAt returns the time at which the rate limit quota fully replenishes.
func (r *Result) ResetAt() time.Time {
	return time.Unix(0, r.resetAt)
}

// TokenBucket checks whether a request identified by key is allowed under
// token bucket rate limiting. Tokens refill at RefillRate per second up to
// MaxRequests capacity, allowing short bursts. Returns a *Result with quota
// details; on Redis error, returns denied.
func (config *RateLimit) TokenBucket(ctx context.Context, key string) *Result {
	key = "tb:" + key
	now := time.Now().UnixNano()

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local capacity = tonumber(ARGV[3])
		local tokens, last_refill = unpack(redis.call('HMGET', key, 'tokens', 'last_refill'))
		tokens = tonumber(tokens) or capacity
		last_refill = tonumber(last_refill) or now
		local elapsed = now - last_refill
		local new_tokens = math.min(capacity, tokens + (elapsed * rate / 1e9))
		local allowed = 0
		local remaining = new_tokens
		local reset_at = now + ((capacity - new_tokens) / rate * 1e9)
		if new_tokens >= 1 then
			allowed = 1
			remaining = new_tokens - 1
			redis.call('HMSET', key, 'tokens', remaining, 'last_refill', now)
			redis.call('EXPIRE', key, math.ceil(capacity / rate))
		end
		return {allowed, capacity, math.floor(remaining), reset_at}
	`

	result, err := config.Redis.Eval(ctx, script, []string{key},
		now, config.RefillRate, config.MaxRequests).Int64Slice()
	if err != nil {
		return &Result{Allowed: false, Total: int64(config.MaxRequests), Remaining: 0, resetAt: now}
	}

	return &Result{
		Allowed:   result[0] == 1,
		Total:     result[1],
		Remaining: result[2],
		resetAt:   result[3],
	}
}

// LeakyBucket checks whether a request identified by key is allowed under
// leaky bucket rate limiting. Requests fill a queue that drains at a constant
// rate, enforcing smooth throughput with no bursts. Returns true if allowed,
// or false if the queue is full or a Redis error occurs.
func (config *RateLimit) LeakyBucket(ctx context.Context, key string) bool {
	key = "lb:" + key
	now := time.Now().UnixNano()

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local capacity = tonumber(ARGV[3])
		local queue, last_leak = unpack(redis.call('HMGET', key, 'queue', 'last_leak'))
		queue = tonumber(queue) or 0
		last_leak = tonumber(last_leak) or now
		local leaked = math.floor((now - last_leak) * rate / 1e9)
		queue = math.max(0, queue - leaked)
		if queue < capacity then
			redis.call('HMSET', key, 'queue', queue + 1, 'last_leak', now)
			redis.call('EXPIRE', key, math.ceil(capacity / rate))
			return 1
		end
		return 0
	`

	result, err := config.Redis.Eval(ctx, script, []string{key},
		now, config.MaxRequests, config.MaxRequests).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// SlidingWindow checks whether a request identified by key is allowed under
// sliding window log rate limiting. It tracks individual timestamps, allowing
// at most MaxRequests within any rolling Window. Returns true if allowed, or
// false if the limit is reached or a Redis error occurs.
func (config *RateLimit) SlidingWindow(ctx context.Context, key string) bool {
	key = "sw:" + key
	now := time.Now().UnixNano()
	windowNanos := config.Window.Nanoseconds()

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local max_requests = tonumber(ARGV[3])
		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		local count = redis.call('ZCARD', key)
		if count < max_requests then
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, math.ceil(window / 1e9))
			return 1
		end
		return 0
	`

	result, err := config.Redis.Eval(ctx, script, []string{key},
		now, windowNanos, config.MaxRequests).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// FixedWindow checks whether a request identified by key is allowed under
// fixed window rate limiting. It counts requests in discrete, non-overlapping
// windows of length Window, allowing up to MaxRequests per window. Bursts up
// to 2x the limit are possible at window boundaries. Returns true if allowed.
func (config *RateLimit) FixedWindow(ctx context.Context, key string) bool {
	key = "fw:" + key
	now := time.Now().UnixNano()
	windowStart := (now / config.Window.Nanoseconds()) * config.Window.Nanoseconds()

	script := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local max_requests = tonumber(ARGV[2])
		local window_seconds = tonumber(ARGV[3])
		local window_key = key .. ":" .. window_start
		local count = redis.call('GET', window_key) or 0
		count = tonumber(count)
		if count < max_requests then
			redis.call('INCR', window_key)
			redis.call('EXPIRE', window_key, window_seconds)
			return 1
		end
		return 0
	`

	result, err := config.Redis.Eval(ctx, script, []string{key},
		windowStart, config.MaxRequests, int(config.Window.Seconds())).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// SlidingWindowCounter checks whether a request identified by key is allowed
// under sliding window counter rate limiting. It approximates a true sliding
// window by weighting the current and previous fixed window counters, offering
// a balance between accuracy and memory efficiency. Returns true if allowed.
func (config *RateLimit) SlidingWindowCounter(ctx context.Context, key string) bool {
	key = "swc:" + key
	now := time.Now().UnixNano()
	windowNanos := config.Window.Nanoseconds()
	currentWindow := (now / windowNanos) * windowNanos
	previousWindow := currentWindow - windowNanos

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local current_window = tonumber(ARGV[2])
		local previous_window = tonumber(ARGV[3])
		local window_nanos = tonumber(ARGV[4])
		local max_requests = tonumber(ARGV[5])
		local window_seconds = tonumber(ARGV[6])

		local current_key = key .. ":" .. current_window
		local previous_key = key .. ":" .. previous_window

		local current_count = tonumber(redis.call('GET', current_key) or 0)
		local previous_count = tonumber(redis.call('GET', previous_key) or 0)

		local elapsed_in_current = now - current_window
		local weight = elapsed_in_current / window_nanos
		local estimated_count = previous_count * (1 - weight) + current_count

		if estimated_count < max_requests then
			redis.call('INCR', current_key)
			redis.call('EXPIRE', current_key, window_seconds * 2)
			return 1
		end
		return 0
	`

	result, err := config.Redis.Eval(ctx, script, []string{key},
		now, currentWindow, previousWindow, windowNanos, config.MaxRequests, int(config.Window.Seconds())).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// DistributedSlidingWindow checks whether a request identified by key is
// allowed under sliding window rate limiting across multiple application
// instances. Each node tracks its own shard and aggregates counts from all
// shards to enforce a global limit. Returns true if allowed.
func (config *RateLimit) DistributedSlidingWindow(ctx context.Context, key string, nodeID string) bool {
	key = "dsw:" + key
	now := time.Now().UnixNano()
	windowNanos := config.Window.Nanoseconds()
	shardKey := key + ":shard:" + nodeID

	script := `
		local key = KEYS[1]
		local shard_key = KEYS[2]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local max_requests = tonumber(ARGV[3])
		local node_id = ARGV[4]
		local window_seconds = tonumber(ARGV[5])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		redis.call('ZREMRANGEBYSCORE', shard_key, 0, now - window)

		local global_count = redis.call('ZCARD', key)
		local shard_count = redis.call('ZCARD', shard_key)

		local shards_key = key .. ":shards"
		redis.call('SADD', shards_key, node_id)
		redis.call('EXPIRE', shards_key, window_seconds)

		local all_shards = redis.call('SMEMBERS', shards_key)
		local total_distributed_count = 0

		for i = 1, #all_shards do
			if all_shards[i] ~= node_id then
				local other_shard_key = key .. ":shard:" .. all_shards[i]
				local other_count = redis.call('ZCARD', other_shard_key)
				total_distributed_count = total_distributed_count + other_count
			end
		end

		local estimated_total = total_distributed_count + shard_count

		if estimated_total < max_requests then
			redis.call('ZADD', key, now, now .. ":" .. node_id)
			redis.call('ZADD', shard_key, now, now)
			redis.call('EXPIRE', key, window_seconds)
			redis.call('EXPIRE', shard_key, window_seconds)
			return 1
		end
		return 0
	`

	result, err := config.Redis.Eval(ctx, script, []string{key, shardKey},
		now, windowNanos, config.MaxRequests, nodeID, int(config.Window.Seconds())).Int()
	if err != nil {
		return false
	}
	return result == 1
}
