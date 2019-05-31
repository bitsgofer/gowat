package ratelimiter

import (
	"sync"
)

// Limiter implements a rate limiter with not-so-good "sliding" window.
// It's not the focus here, but writing it led to the bug in the test.
type Limiter struct {
	maxRPS       int64
	windowLength int64

	sync.Mutex
	buckets map[int64]int64 // <unix timestamp, unit: second> -> count
}

func New(windowLength, maxRequestsPerSecond int64) *Limiter {
	return &Limiter{
		maxRPS:       maxRequestsPerSecond,
		windowLength: windowLength,
		buckets:      make(map[int64]int64),
	}
}

// IsAllow returns whether the call at <now> is allowed by the rate limiter.
// It expects the values of <now> to be ever increasing.
// It will reject a call if including that call, there are > r.maxRPS calls
// in the range [now - r.windowLength, now] (inclusive).
func (r *Limiter) IsAllow(now int64) bool {
	r.Lock()

	r.buckets[now]++
	var sum int64
	for ts, count := range r.buckets {
		if (now-r.windowLength) <= ts && ts <= now {
			sum += count
			continue
		}

		delete(r.buckets, ts) // clear now
	}

	r.Unlock()

	rate := sum / r.windowLength // int division -> might exceed by X requests (X < r.windowLength)
	if rate > r.maxRPS {
		return false
	}

	return true
}
