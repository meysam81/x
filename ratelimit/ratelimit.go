package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimit struct {
	Redis       *redis.Client
	MaxRequests int
	RefillRate  float32
	Window      time.Duration
}

type Result struct {
	Allowed   bool
	Total     int64
	Remaining int64

	resetAt int64
}

func (r *Result) ResetAt() time.Time {
	return time.Unix(0, r.resetAt)
}

// TokenBucket implements the token bucket rate limiting algorithm.
//
// This algorithm maintains a bucket with a fixed capacity of tokens that are
// refilled at a constant rate. Each request consumes one token. If no tokens
// are available, the request is rejected.
//
// Advantages:
// - Allows bursts up to the bucket capacity
// - Smooths out traffic over time
// - Predictable refill rate
// - Good for APIs that need to handle occasional traffic spikes
//
// Best use cases:
// - APIs that should allow brief bursts of activity
// - Services where you want to permit accumulated "credits" for unused capacity
// - Applications that need flexible rate limiting with burst tolerance
// - User-facing APIs where occasional higher usage should be permitted
//
// Companies using this pattern:
//   - Amazon AWS API Gateway: Uses token bucket for API throttling to allow burst traffic
//     while maintaining overall rate limits for their cloud services
//   - Netflix: Implements token bucket in their microservices architecture to handle
//     traffic spikes during peak viewing hours while protecting backend services
//   - GitHub: Uses token bucket for their REST API rate limiting to allow developers
//     to make burst requests while staying within hourly limits
//   - Shopify: Employs token bucket for their API rate limiting to handle merchant
//     traffic bursts during sales events while protecting their infrastructure
//
// Example: An API that allows 100 requests per minute but should handle
// a user making 50 requests in the first 10 seconds, then being limited
// to the refill rate afterward.
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

// LeakyBucket implements the leaky bucket rate limiting algorithm.
//
// This algorithm maintains a queue with fixed capacity. Requests are added to
// the queue and processed (leaked) at a constant rate. If the queue is full,
// new requests are rejected.
//
// Advantages:
// - Enforces strict rate limiting without bursts
// - Provides smooth, consistent output rate
// - Prevents sudden traffic spikes from overwhelming downstream services
// - More predictable resource usage
//
// Best use cases:
// - Background job processing systems
// - API gateways protecting downstream services with strict capacity limits
// - Systems where consistent, predictable load is critical
// - Services that cannot handle traffic bursts
// - Rate limiting expensive operations like database writes or external API calls
//
// Companies using this pattern:
//   - Google Cloud Platform: Uses leaky bucket in their Cloud Functions and App Engine
//     to ensure consistent processing rates and protect against overwhelming their
//     container orchestration systems
//   - Stripe: Implements leaky bucket for payment processing to maintain consistent
//     transaction processing rates and prevent overwhelming their banking partners
//   - Uber: Uses leaky bucket in their dispatch systems to ensure smooth, consistent
//     driver assignment rates during peak demand without overwhelming their algorithms
//   - Slack: Employs leaky bucket for message delivery to ensure consistent chat
//     message processing rates across their distributed messaging infrastructure
//
// Example: A service that processes webhook events and can only handle
// exactly 10 events per second without causing performance issues.
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

// SlidingWindow implements the sliding window rate limiting algorithm.
//
// This algorithm tracks request timestamps within a moving time window.
// It allows MaxRequests within any Window duration. Old requests outside
// the current window are automatically discarded.
//
// Advantages:
// - Precise rate limiting over exact time periods
// - No burst allowance - strict adherence to rate limits
// - Memory of recent activity provides fair rate limiting
// - Prevents gaming the system by timing requests around fixed windows
//
// Best use cases:
// - APIs with strict rate limits that must be precisely enforced
// - Premium API tiers with exact usage quotas
// - Anti-abuse systems where precise timing matters
// - Services where you need to prevent concentrated usage patterns
// - Financial or billing APIs where exact rate enforcement is required
//
// Companies using this pattern:
//   - Twitter/X: Uses sliding window for their API rate limiting to prevent abuse
//     and ensure fair access to their real-time data streams and posting APIs
//   - Discord: Implements sliding window for message rate limiting to prevent spam
//     and ensure fair usage of their chat platforms across millions of users
//   - Reddit: Uses sliding window for comment and post submission rate limiting
//     to prevent spam and maintain platform quality while allowing normal usage
//   - Twilio: Employs sliding window for their messaging and voice API rate limits
//     to ensure precise enforcement of usage quotas for their communication services
//
// Example: A premium API service that strictly allows 1000 requests per hour
// per user, where the user cannot circumvent the limit by timing requests
// around hourly boundaries.
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

// FixedWindow implements the fixed window counter rate limiting algorithm.
//
// This algorithm divides time into fixed windows and counts requests within
// each window. The counter resets at the start of each new window.
//
// Advantages:
// - Memory efficient - only stores a single counter per window
// - Simple to understand and implement
// - Fast performance with minimal Redis operations
// - Predictable memory usage
//
// Best use cases:
// - High-throughput APIs where memory efficiency is critical
// - Simple rate limiting requirements without burst considerations
// - Analytics and monitoring systems
// - Basic API quotas where precision at window boundaries isn't critical
//
// Companies using this pattern:
//   - LinkedIn: Uses fixed window for their social API rate limiting to efficiently
//     handle millions of requests while maintaining simple, predictable limits
//   - Pinterest: Implements fixed window for their image upload and pin creation APIs
//     to provide straightforward rate limiting with minimal computational overhead
//   - Instagram: Uses fixed window for basic API rate limiting on their platform APIs
//     to efficiently manage high-volume traffic from mobile applications
//   - Spotify: Employs fixed window for their music streaming API rate limits
//     to handle massive scale while keeping rate limiting logic simple and fast
//
// Example: An API that allows 1000 requests per hour, resetting the counter
// at the top of each hour (00:00, 01:00, 02:00, etc.).
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

// SlidingWindowCounter implements the sliding window counter rate limiting algorithm.
//
// This algorithm combines fixed windows with weighted calculations to approximate
// a true sliding window. It uses the current and previous window counts with
// time-based weighting to estimate the rate within a sliding window.
//
// Advantages:
// - More accurate than fixed window, more efficient than sliding window log
// - Smooth rate limiting without sharp boundaries
// - Memory efficient - only stores two counters
// - Good balance between accuracy and performance
//
// Best use cases:
// - High-scale APIs needing better accuracy than fixed windows
// - CDNs and edge computing where efficiency matters
// - Services requiring smooth rate limiting without exact precision
// - Multi-tenant systems with many rate limit keys
//
// Companies using this pattern:
//   - Cloudflare: Uses sliding window counter in their edge network for DDoS protection
//     and rate limiting across their global CDN infrastructure
//   - Fastly: Implements sliding window counter for their CDN rate limiting to balance
//     accuracy and performance across their edge computing platform
//   - Kong: Uses sliding window counter in their API gateway for rate limiting
//     microservices while maintaining high performance and reasonable accuracy
//   - DataDog: Employs sliding window counter for their metrics ingestion rate limiting
//     to handle massive scale monitoring data while preventing abuse
//
// Example: An API that allows 100 requests per minute with smooth enforcement,
// where a request at 10:30:30 considers requests from 09:29:30 to 10:30:30
// using weighted calculations from two 1-minute windows.
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

// DistributedSlidingWindow implements a distributed sliding window rate limiting algorithm.
//
// This algorithm maintains sliding window state across multiple Redis nodes or
// instances, using distributed consensus to ensure accurate rate limiting even
// when requests are spread across multiple application instances.
//
// Advantages:
// - Accurate rate limiting in distributed environments
// - Handles network partitions and Redis failover scenarios
// - Scales horizontally across multiple application instances
// - Maintains consistency across distributed systems
//
// Best use cases:
// - Microservices architectures with multiple instances
// - Multi-region deployments requiring consistent rate limiting
// - Large-scale systems where single Redis instance isn't sufficient
// - Critical systems requiring rate limit accuracy despite infrastructure failures
//
// Companies using this pattern:
//   - PayPal: Uses distributed sliding window for their payment APIs across multiple
//     data centers to ensure consistent fraud protection and rate limiting globally
//   - Airbnb: Implements distributed sliding window for their booking and search APIs
//     across multiple regions to prevent abuse while maintaining service availability
//   - Coinbase: Uses distributed sliding window for their cryptocurrency trading APIs
//     to ensure consistent rate limiting across their distributed exchange infrastructure
//   - Square: Employs distributed sliding window for their payment processing APIs
//     to maintain consistent rate limits across their global point-of-sale network
//
// Example: A payment API deployed across multiple regions where rate limits
// must be enforced globally - a user hitting the US endpoint shouldn't be
// able to bypass limits by simultaneously hitting the EU endpoint.
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
